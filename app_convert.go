package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"zimap/internal/archive"
	"zimap/internal/converter"
	"zimap/internal/download"
	"zimap/internal/provider"
	"zimap/internal/setup"
)

// BuildBBBikeStyleURL resolves the pre-compiled BBBike Garmin style download URL for a
// given city name and style id ("onroad", "ontrail", "opentopo", "osm"). It applies the
// canonical BBBike slug alias resolution, so cities like "Munich" map to "Muenchen".
func (a *App) BuildBBBikeStyleURL(cityName string, styleID string) string {
	return provider.BuildBBBikeStyleURL(cityName, styleID)
}

func (a *App) ConvertURL(downloadURL string, name string, heapSize string, detailLevel string) error {
	if strings.Contains(downloadURL, "bbbike.org/osm/bbbike/") {
		downloadURL = provider.NormalizeBBBikeURL(downloadURL)
	}

	a.jobMutex.Lock()
	if a.cancelJob != nil {
		a.jobMutex.Unlock()
		return fmt.Errorf("a job is already running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelJob = cancel
	a.jobMutex.Unlock()

	var success bool
	var runErr error
	defer func() {
		a.jobMutex.Lock()
		a.cancelJob = nil
		a.jobMutex.Unlock()
		if success {
			a.emitEvent("convert:done")
			_ = a.SendNotification("ZiMap Downloader", fmt.Sprintf("Карта %s успешно собрана!", name))
		} else if runErr != nil {
			_ = a.SendNotification("ZiMap Downloader", fmt.Sprintf("Ошибка сборки карты %s: %s", name, runErr.Error()))
		}
	}()

	a.emitEvent("convert:stage", "download")

	// Detect BBBike pre-compiled Garmin style archive (direct gmapsupp.img extraction)
	if styleID := provider.DetectBBBikeStyleID(downloadURL); styleID != "" {
		styleErr := a.downloadPrecompiledBBBike(ctx, downloadURL, name, heapSize, detailLevel)
		if styleErr != nil {
			runErr = styleErr
			return styleErr
		}
		success = true
		return nil
	}

	client := download.NewStreaming()

	// Create temp file for download
	outDir, err := setup.OutputDirectory()
	if err != nil {
		runErr = err
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}

	tempDir, err := os.MkdirTemp(outDir, ".download-*")
	if err != nil {
		runErr = err
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}
	defer os.RemoveAll(tempDir)

	fileName := "map.osm.pbf"
	parsed, err := url.Parse(downloadURL)
	if err == nil {
		fileName = path.Base(parsed.Path)
	}
	pbfPath := filepath.Join(tempDir, fileName)

	opts := download.DownloadOptions{
		MaxRetries:     3,
		InitialBackoff: 2 * time.Second,
		LogCallback: func(msg string) {
			a.emitEvent("convert:log", msg)
		},
		ProgressCallback: a.emitDownloadProgress,
	}

	err = client.DownloadWithOptions(ctx, downloadURL, pbfPath, opts)
	if err != nil {
		runErr = err
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}

	err = a.runConversion(ctx, pbfPath, name, heapSize, detailLevel)
	if err != nil {
		runErr = err
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}

	sourceType := "geofabrik"
	if strings.Contains(downloadURL, "bbbike.org") {
		sourceType = "bbbike"
	} else if strings.Contains(downloadURL, "openstreetmap.fr") {
		sourceType = "osm-france"
	}

	if metaErr := a.saveCompanionMetadata(name, sourceType, downloadURL, heapSize, detailLevel); metaErr != nil {
		a.emitEvent("convert:log", fmt.Sprintf("[WARN] Failed to save metadata: %v", metaErr))
	}

	success = true
	return nil
}

// emitDownloadProgress streams download transfer metrics to the frontend (shared by
// URL and BBBike precompiled downloads).
func (a *App) emitDownloadProgress(downloaded int64, total int64, speedBps float64) {
	percent := 0
	if total > 0 {
		percent = int((downloaded * 100) / total)
	}
	eta := 0
	if speedBps > 0 && total > downloaded {
		eta = int(float64(total-downloaded) / speedBps)
	}
	a.emitEvent("convert:progress", percent)
	a.emitEvent("convert:progress_detail", ProgressDetail{
		Downloaded: downloaded,
		Total:      total,
		Percent:    percent,
		SpeedBps:   speedBps,
		SpeedText:  formatSpeed(speedBps),
		EtaSeconds: eta,
	})
}

// downloadPrecompiledBBBike handles BBBike pre-compiled Garmin style archives
// (garmin-onroad-latin1.zip, garmin-ontrail-latin1.zip, garmin-opentopo-latin1.zip,
// garmin-osm.zip) by streaming the ZIP, extracting the gmapsupp.img inside
// <CityName>-garmin-<style>-latin1/gmapsupp.img, and renaming it to the output file.
// No local splitter/mkgmap compilation is needed.
func (a *App) downloadPrecompiledBBBike(ctx context.Context, downloadURL, name, heapSize, detailLevel string) error {
	a.emitEvent("convert:log", fmt.Sprintf("[BBBike Style] Downloading pre-compiled style from: %s", downloadURL))

	outDir, err := setup.OutputDirectory()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp(outDir, ".bbbike-precompiled-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	zipPath := filepath.Join(tempDir, "style.zip")

	client := download.NewStreaming()
	opts := download.DownloadOptions{
		MaxRetries:     3,
		InitialBackoff: 2 * time.Second,
		LogCallback: func(msg string) {
			a.emitEvent("convert:log", msg)
		},
		ProgressCallback: a.emitDownloadProgress,
	}

	a.emitEvent("convert:stage", "download")
	if err := client.DownloadWithOptions(ctx, downloadURL, zipPath, opts); err != nil {
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}

	a.emitEvent("convert:log", "[BBBike Style] Extracting archive...")
	a.emitEvent("convert:stage", "compile")
	a.emitEvent("convert:progress", 90)

	extractDir := filepath.Join(tempDir, "extracted")
	if err := archive.ExtractArchive(zipPath, extractDir); err != nil {
		a.emitEvent("convert:error", fmt.Sprintf("extract BBBike archive: %v", err))
		a.emitEvent("convert:stage", "error")
		return fmt.Errorf("extract BBBike archive: %w", err)
	}

	// Walk the extracted directory to find gmapsupp.img (it lives inside a city-named subfolder)
	var gmapsuppPath string
	var findErr error
	filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() && strings.EqualFold(info.Name(), "gmapsupp.img") {
			gmapsuppPath = path
			return filepath.SkipDir // found it, stop walking
		}
		return nil
	})
	if gmapsuppPath == "" {
		findErr = fmt.Errorf("gmapsupp.img not found inside BBBike archive")
		a.emitEvent("convert:error", findErr.Error())
		a.emitEvent("convert:stage", "error")
		return findErr
	}

	finalMapPath := filepath.Join(outDir, name+".img")
	if err := os.Rename(gmapsuppPath, finalMapPath); err != nil {
		// Fallback: copy if rename across filesystems fails
		if copyErr := copyFile(gmapsuppPath, finalMapPath); copyErr != nil {
			a.emitEvent("convert:error", fmt.Sprintf("publish map: %v", copyErr))
			a.emitEvent("convert:stage", "error")
			return fmt.Errorf("publish map: %w", copyErr)
		}
		_ = os.Remove(gmapsuppPath)
	}

	a.emitEvent("convert:progress", 100)
	a.emitEvent("convert:stage", "finalize")

	sourceType := "bbbike"
	if metaErr := a.saveCompanionMetadata(name, sourceType, downloadURL, heapSize, detailLevel); metaErr != nil {
		a.emitEvent("convert:log", fmt.Sprintf("[WARN] Failed to save metadata: %v", metaErr))
	}

	return nil
}

func (a *App) ConvertLocal(filePath string, name string, heapSize string, detailLevel string) error {
	a.jobMutex.Lock()
	if a.cancelJob != nil {
		a.jobMutex.Unlock()
		return fmt.Errorf("a job is already running")
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelJob = cancel
	a.jobMutex.Unlock()

	var success bool
	var runErr error
	defer func() {
		a.jobMutex.Lock()
		a.cancelJob = nil
		a.jobMutex.Unlock()
		if success {
			a.emitEvent("convert:done")
			_ = a.SendNotification("ZiMap Downloader", fmt.Sprintf("Карта %s успешно собрана!", name))
		} else if runErr != nil {
			_ = a.SendNotification("ZiMap Downloader", fmt.Sprintf("Ошибка сборки карты %s: %s", name, runErr.Error()))
		}
	}()

	err := a.runConversion(ctx, filePath, name, heapSize, detailLevel)
	if err != nil {
		runErr = err
		a.emitEvent("convert:error", err.Error())
		a.emitEvent("convert:stage", "error")
		return err
	}

	if metaErr := a.saveCompanionMetadata(name, "local", filePath, heapSize, detailLevel); metaErr != nil {
		a.emitEvent("convert:log", fmt.Sprintf("[WARN] Failed to save metadata: %v", metaErr))
	}

	success = true
	return nil
}

func (a *App) runConversion(ctx context.Context, pbfPath string, name string, heapSize string, detailLevel string) error {
	if heapSize != "" {
		os.Setenv("ZIMAP_JAVA_HEAP", heapSize)
	}

	outDir, err := setup.OutputDirectory()
	if err != nil {
		a.emitEvent("convert:error", err.Error())
		return err
	}

	workDir, err := os.MkdirTemp(outDir, ".zimap-"+name+"-*")
	if err != nil {
		a.emitEvent("convert:error", err.Error())
		return err
	}
	defer os.RemoveAll(workDir)

	dl, err := converter.NormalizeDetailLevel(detailLevel)
	if err != nil {
		a.emitEvent("convert:error", err.Error())
		return err
	}
	a.emitEvent("convert:log", fmt.Sprintf("[INFO] Map detail level: %s", dl))
	opts := converter.ConvertOptions{
		MapName:     name,
		DetailLevel: dl,
		Progress: func(stage string, percent int) {
			a.emitEvent("convert:stage", stage)
			a.emitEvent("convert:progress", percent)
		},
		Log: func(msg string) {
			a.emitEvent("convert:log", msg)
		},
	}

	a.emitEvent("convert:stage", "splitter") // initial
	_, err = converter.RunWithContext(ctx, pbfPath, workDir, opts)
	if err != nil {
		a.emitEvent("convert:error", err.Error())
		return err
	}

	finalMapPath := filepath.Join(outDir, name+".img")

	// rename workDir/gmapsupp.img to finalMapPath
	err = os.Rename(filepath.Join(workDir, "gmapsupp.img"), finalMapPath)
	if err != nil {
		a.emitEvent("convert:error", err.Error())
		return err
	}

	return nil
}

func (a *App) CancelActiveJob() error {
	a.jobMutex.Lock()
	defer a.jobMutex.Unlock()
	if a.cancelJob != nil {
		a.cancelJob()
		a.cancelJob = nil
	}
	return nil
}
