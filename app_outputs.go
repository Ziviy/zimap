package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"zimap/internal/setup"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type MapMetadata struct {
	Name        string    `json:"name"`
	SourceType  string    `json:"sourceType"` // "geofabrik" | "bbbike" | "osm-france" | "local"
	SourceURL   string    `json:"sourceURL"`
	RegionID    string    `json:"regionID,omitempty"`
	HeapSize    string    `json:"heapSize,omitempty"`
	DetailLevel string    `json:"detailLevel,omitempty"` // "standard" | "high" | "topo"
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type OutputItem struct {
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Size      int64        `json:"size"`
	ModTime   time.Time    `json:"modTime"`
	SourceURL string       `json:"sourceUrl,omitempty"`
	Meta      *MapMetadata `json:"meta,omitempty"`
}

func normalizeRegionStr(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func (a *App) findRegionInCatalog(cat *CatalogData, predicate func(r RegionData) bool) *RegionData {
	if cat == nil {
		return nil
	}
	var search func(regions []RegionData) *RegionData
	search = func(regions []RegionData) *RegionData {
		for _, r := range regions {
			if predicate(r) {
				matched := r
				return &matched
			}
			if found := search(r.Children); found != nil {
				return found
			}
		}
		return nil
	}
	return search(cat.Roots)
}

func (a *App) findRegion(sourceURL, name string) *RegionData {
	a.catalogMutex.RLock()
	cat := a.catalogCache
	a.catalogMutex.RUnlock()

	if cat == nil {
		cat, _ = a.GetCatalog()
		if cat == nil {
			return nil
		}
	}

	if sourceURL != "" {
		if found := a.findRegionInCatalog(cat, func(r RegionData) bool {
			return r.PBFURL != "" && (r.PBFURL == sourceURL || strings.EqualFold(r.PBFURL, sourceURL))
		}); found != nil {
			return found
		}
	}

	clean := strings.TrimSuffix(name, filepath.Ext(name))
	cleanLower := strings.ToLower(clean)
	normClean := normalizeRegionStr(clean)

	return a.findRegionInCatalog(cat, func(r RegionData) bool {
		return strings.EqualFold(r.ID, clean) ||
			strings.EqualFold(r.Name, clean) ||
			normalizeRegionStr(r.ID) == normClean ||
			normalizeRegionStr(r.Name) == normClean ||
			strings.ToLower(r.ID) == cleanLower
	})
}

func (a *App) findRegionID(sourceURL, name string) string {
	if r := a.findRegion(sourceURL, name); r != nil {
		return r.ID
	}
	return ""
}

func (a *App) saveCompanionMetadata(name, sourceType, sourceURL, heapSize string, detailLevel string) error {
	outDir, err := setup.OutputDirectory()
	if err != nil {
		return err
	}

	cleanName := strings.TrimSuffix(name, filepath.Ext(name))
	metaFile := filepath.Join(outDir, cleanName+".meta.json")
	now := time.Now()
	createdAt := now

	var existing MapMetadata
	if data, err := os.ReadFile(metaFile); err == nil {
		if json.Unmarshal(data, &existing) == nil && !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
	} else {
		altFile := filepath.Join(outDir, cleanName+".img.meta.json")
		if altData, err := os.ReadFile(altFile); err == nil {
			if json.Unmarshal(altData, &existing) == nil && !existing.CreatedAt.IsZero() {
				createdAt = existing.CreatedAt
			}
		}
	}

	var regionID string
	if sourceType == "geofabrik" {
		regionID = a.findRegionID(sourceURL, cleanName)
	}
	if regionID == "" && existing.RegionID != "" {
		regionID = existing.RegionID
	}

	meta := MapMetadata{
		Name:        cleanName,
		SourceType:  sourceType,
		SourceURL:   sourceURL,
		RegionID:    regionID,
		HeapSize:    heapSize,
		DetailLevel: detailLevel,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}

	bytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaFile, bytes, 0644)
}

func (a *App) ListOutputs() ([]OutputItem, error) {
	outputDir, err := setup.OutputDirectory()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []OutputItem{}, nil
		}
		return nil, err
	}

	items := make([]OutputItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".img") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			item := OutputItem{
				Name:    entry.Name(),
				Path:    filepath.Join(outputDir, entry.Name()),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}

			cleanName := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			metaPathA := filepath.Join(outputDir, cleanName+".meta.json")
			metaPathB := filepath.Join(outputDir, entry.Name()+".meta.json")

			var metaData []byte
			if data, err := os.ReadFile(metaPathA); err == nil {
				metaData = data
			} else if data, err := os.ReadFile(metaPathB); err == nil {
				metaData = data
			}

			if len(metaData) > 0 {
				var m MapMetadata
				if err := json.Unmarshal(metaData, &m); err == nil {
					item.Meta = &m
					item.SourceURL = m.SourceURL
				}
			}

			if item.Meta == nil {
				// Fallback search a.catalogCache
				if r := a.findRegion("", cleanName); r != nil {
					syncedMeta := MapMetadata{
						Name:       cleanName,
						SourceType: "geofabrik",
						SourceURL:  r.PBFURL,
						RegionID:   r.ID,
						CreatedAt:  info.ModTime(),
						UpdatedAt:  info.ModTime(),
					}
					// Persist synthesized .meta.json to disk so it's cached
					if bytes, err := json.MarshalIndent(syncedMeta, "", "  "); err == nil {
						_ = os.WriteFile(metaPathA, bytes, 0644)
					}
					item.Meta = &syncedMeta
					item.SourceURL = syncedMeta.SourceURL
				}
			}

			items = append(items, item)
		}
	}
	return items, nil
}

func (a *App) DeleteOutput(path string) error {
	outputDir, err := setup.OutputDirectory()
	if err != nil {
		return err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("failed to resolve output directory: %w", err)
	}

	var absTarget string
	if filepath.IsAbs(path) {
		absTarget = filepath.Clean(path)
	} else {
		absTarget = filepath.Clean(filepath.Join(absOutputDir, path))
	}

	rel, err := filepath.Rel(absOutputDir, absTarget)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("access denied: path is outside output directory: %s", path)
	}

	// Check file exists and is not a directory
	info, err := os.Stat(absTarget)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("target is a directory, not a file: %s", path)
	}

	// Check file has .img extension (case-insensitive)
	if !strings.EqualFold(filepath.Ext(absTarget), ".img") {
		return fmt.Errorf("file is not an .img map: %s", path)
	}

	// Remove the .img file
	if err := os.Remove(absTarget); err != nil {
		return fmt.Errorf("failed to delete map file: %w", err)
	}

	// Clean up companion .meta.json files
	meta1 := absTarget + ".meta.json"
	meta2 := strings.TrimSuffix(absTarget, filepath.Ext(absTarget)) + ".meta.json"

	_ = os.Remove(meta1)
	if meta2 != meta1 {
		_ = os.Remove(meta2)
	}

	return nil
}

func (a *App) RevealFile(path string) error {
	if !filepath.IsAbs(path) {
		outputDir, err := setup.OutputDirectory()
		if err == nil {
			candidate := filepath.Join(outputDir, path)
			if _, err2 := os.Stat(candidate); err2 == nil {
				path = candidate
			} else if !strings.EqualFold(filepath.Ext(candidate), ".img") {
				if _, err3 := os.Stat(candidate + ".img"); err3 == nil {
					path = candidate + ".img"
				}
			}
		}
	}

	if runtime.GOOS == "windows" {
		return exec.Command("explorer", "/select,", path).Run()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", "-R", path).Run()
	}
	if runtime.GOOS == "linux" {
		if p, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(p, filepath.Dir(path)).Run()
		}
	}
	return nil
}

func (a *App) SelectLocalPBFDialog() (string, error) {
	if a.ctx == nil {
		return "", fmt.Errorf("window context not initialized")
	}
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select OpenStreetMap File",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "OpenStreetMap Data (*.pbf, *.osm, *.gz, *.bz2)", Pattern: "*.osm.pbf;*.pbf;*.osm;*.osm.gz;*.osm.bz2"},
			{DisplayName: "OSM PBF (*.pbf)", Pattern: "*.osm.pbf;*.pbf"},
			{DisplayName: "OSM XML (*.osm, *.gz, *.bz2)", Pattern: "*.osm;*.osm.gz;*.osm.bz2"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
}

// copyFile streams content from src to dst and ensures persistence with Sync.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

// ExportMap prompts the user with a native Save File Dialog and copies the compiled map (.img)
// from the app's output directory to the user-chosen location.
// Returns the destination file path if exported, empty string if user cancelled, or an error.
func (a *App) ExportMap(path string) (string, error) {
	outputDir, err := setup.OutputDirectory()
	if err != nil {
		return "", err
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve output directory: %w", err)
	}

	var absSource string
	if filepath.IsAbs(path) {
		absSource = filepath.Clean(path)
	} else {
		candidate := filepath.Clean(filepath.Join(absOutputDir, path))
		if _, err := os.Stat(candidate); os.IsNotExist(err) && !strings.EqualFold(filepath.Ext(candidate), ".img") {
			if _, err2 := os.Stat(candidate + ".img"); err2 == nil {
				candidate += ".img"
			}
		}
		absSource = candidate
	}

	// Security validation: verify path is strictly within output directory
	rel, err := filepath.Rel(absOutputDir, absSource)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("access denied: path is outside output directory: %s", path)
	}

	info, err := os.Stat(absSource)
	if err != nil {
		return "", fmt.Errorf("map file does not exist: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("target is a directory, not a file: %s", path)
	}
	if !strings.EqualFold(filepath.Ext(absSource), ".img") {
		return "", fmt.Errorf("file is not an .img map: %s", path)
	}

	if a.ctx == nil {
		return "", fmt.Errorf("window context not initialized")
	}

	defaultName := filepath.Base(absSource)

	destPath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "Save Map As...",
		DefaultFilename: defaultName,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "GPS Map Image (*.img)",
				Pattern:     "*.img",
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to open save dialog: %w", err)
	}

	if destPath == "" {
		// User cancelled dialog
		return "", nil
	}

	if !strings.EqualFold(filepath.Ext(destPath), ".img") {
		destPath += ".img"
	}

	destAbs, err := filepath.Abs(destPath)
	if err != nil {
		return "", err
	}

	if strings.EqualFold(destAbs, absSource) {
		return destAbs, nil
	}
	if destInfo, err := os.Stat(destAbs); err == nil {
		if os.SameFile(info, destInfo) {
			return destAbs, nil
		}
	}

	if err := copyFile(absSource, destAbs); err != nil {
		return "", fmt.Errorf("failed to export map to %s: %w", destAbs, err)
	}

	return destAbs, nil
}
