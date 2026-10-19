package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"zimap/internal/archive"
	"zimap/internal/download"
	"zimap/internal/setup"
)

type UpdateInfo struct {
	CurrentVersion string `json:"currentVersion"`
	Version        string `json:"version"`
	ReleaseNotes   string `json:"releaseNotes"`
	DownloadURL    string `json:"downloadURL"`
	AssetURL       string `json:"assetURL"`
	AssetName      string `json:"assetName"`
	AssetSize      int64  `json:"assetSize"`
	IsNewer        bool   `json:"isNewer"`
	CanAutoUpdate  bool   `json:"canAutoUpdate"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Body    string        `json:"body"`
	HTMLURL string        `json:"html_url"`
	Assets  []GitHubAsset `json:"assets"`
}

func parseSemVer(v string) [3]int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	for i, r := range v {
		if (r < '0' || r > '9') && r != '.' {
			v = v[:i]
			break
		}
	}
	parts := strings.Split(v, ".")
	var res [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		fmt.Sscanf(parts[i], "%d", &res[i])
	}
	return res
}

func isPrerelease(v string) bool {
	vLower := strings.ToLower(v)
	return strings.Contains(vLower, "beta") ||
		strings.Contains(vLower, "alpha") ||
		strings.Contains(vLower, "rc") ||
		strings.Contains(vLower, "dev")
}

func isVersionNewer(remote, current string) bool {
	r := parseSemVer(remote)
	c := parseSemVer(current)
	for i := 0; i < 3; i++ {
		if r[i] > c[i] {
			return true
		}
		if r[i] < c[i] {
			return false
		}
	}
	currentIsPrerelease := isPrerelease(current)
	remoteIsPrerelease := isPrerelease(remote)
	if currentIsPrerelease && !remoteIsPrerelease {
		return true
	}
	return false
}

func scoreAsset(name, goos, goarch string) int {
	nameLower := strings.ToLower(name)
	if strings.HasSuffix(nameLower, ".txt") ||
		strings.HasSuffix(nameLower, ".sha256") ||
		strings.HasSuffix(nameLower, ".md5") ||
		strings.HasSuffix(nameLower, ".sig") {
		return -1
	}

	score := 0
	switch goos {
	case "darwin":
		if strings.Contains(nameLower, "windows") || strings.Contains(nameLower, "linux") {
			return -1
		}
		if strings.Contains(nameLower, "darwin") || strings.Contains(nameLower, "macos") ||
			strings.Contains(nameLower, "mac") || strings.Contains(nameLower, "osx") {
			score += 10
		} else if strings.HasSuffix(nameLower, ".dmg") || strings.HasSuffix(nameLower, ".app.zip") {
			score += 5
		} else {
			return -1
		}

		if goarch == "arm64" {
			if strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch64") ||
				strings.Contains(nameLower, "apple-silicon") || strings.Contains(nameLower, "m1") {
				score += 10
			} else if strings.Contains(nameLower, "universal") {
				score += 8
			} else if strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "x86_64") ||
				strings.Contains(nameLower, "x64") || strings.Contains(nameLower, "intel") {
				return -1
			}
		} else if goarch == "amd64" {
			if strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "x86_64") ||
				strings.Contains(nameLower, "x64") || strings.Contains(nameLower, "intel") {
				score += 10
			} else if strings.Contains(nameLower, "universal") {
				score += 8
			} else if strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch64") {
				return -1
			}
		}

	case "windows":
		if strings.Contains(nameLower, "darwin") || strings.Contains(nameLower, "mac") || strings.Contains(nameLower, "linux") {
			return -1
		}
		if strings.Contains(nameLower, "windows") || strings.Contains(nameLower, "_win") ||
			strings.Contains(nameLower, "-win") || strings.Contains(nameLower, ".win") ||
			strings.HasSuffix(nameLower, ".exe") {
			score += 10
		} else {
			return -1
		}
		if goarch == "amd64" {
			if strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "x64") || strings.Contains(nameLower, "x86_64") {
				score += 10
			} else if strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "386") {
				return -1
			}
		}

	case "linux":
		if strings.Contains(nameLower, "darwin") || strings.Contains(nameLower, "mac") || strings.Contains(nameLower, "windows") {
			return -1
		}
		if strings.Contains(nameLower, "linux") {
			score += 10
		} else {
			return -1
		}
		if goarch == "amd64" {
			if strings.Contains(nameLower, "amd64") || strings.Contains(nameLower, "x86_64") {
				score += 10
			} else if strings.Contains(nameLower, "arm64") {
				return -1
			}
		} else if goarch == "arm64" {
			if strings.Contains(nameLower, "arm64") || strings.Contains(nameLower, "aarch64") {
				score += 10
			} else if strings.Contains(nameLower, "amd64") {
				return -1
			}
		}
	default:
		return -1
	}

	if strings.HasSuffix(nameLower, ".zip") {
		score += 5
	} else if strings.HasSuffix(nameLower, ".tar.gz") || strings.HasSuffix(nameLower, ".tgz") {
		score += 4
	} else if strings.HasSuffix(nameLower, ".exe") {
		score += 5
	} else if strings.HasSuffix(nameLower, ".dmg") {
		score += 3
	}

	return score
}

func findMatchingAsset(assets []GitHubAsset, goos, goarch string) *GitHubAsset {
	var bestAsset *GitHubAsset
	highestScore := -1

	for i := range assets {
		score := scoreAsset(assets[i].Name, goos, goarch)
		if score > highestScore {
			highestScore = score
			bestAsset = &assets[i]
		}
	}

	if highestScore > 0 {
		return bestAsset
	}
	return nil
}

var releasesLatestURL = "https://api.github.com/repos/Ziviy/zimap/releases/latest"

func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	parentCtx := a.ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(parentCtx, 15*time.Second)
	defer cancel()

	url := releasesLatestURL
	if custom := os.Getenv("ZIMAP_UPDATE_URL"); custom != "" {
		url = custom
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ZiMap-Downloader/"+CurrentAppVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := download.New(nil).Client
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &UpdateInfo{
			CurrentVersion: CurrentAppVersion,
			Version:        CurrentAppVersion,
			ReleaseNotes:   "",
			DownloadURL:    "https://github.com/Ziviy/zimap/releases",
			AssetURL:       "",
			AssetName:      "",
			AssetSize:      0,
			IsNewer:        false,
			CanAutoUpdate:  false,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("failed to parse release information: %w", err)
	}

	isNewer := isVersionNewer(rel.TagName, CurrentAppVersion)
	downloadURL := rel.HTMLURL
	if downloadURL == "" {
		downloadURL = "https://github.com/Ziviy/zimap/releases"
	}

	matchedAsset := findMatchingAsset(rel.Assets, runtime.GOOS, runtime.GOARCH)
	var assetURL, assetName string
	var assetSize int64
	canAuto := false

	if matchedAsset != nil {
		assetURL = matchedAsset.BrowserDownloadURL
		assetName = matchedAsset.Name
		assetSize = matchedAsset.Size
		canAuto = true
	}

	return &UpdateInfo{
		CurrentVersion: CurrentAppVersion,
		Version:        rel.TagName,
		ReleaseNotes:   rel.Body,
		DownloadURL:    downloadURL,
		AssetURL:       assetURL,
		AssetName:      assetName,
		AssetSize:      assetSize,
		IsNewer:        isNewer,
		CanAutoUpdate:  canAuto,
	}, nil
}

func (a *App) SimulateUpdateCheck() (*UpdateInfo, error) {
	return &UpdateInfo{
		CurrentVersion: CurrentAppVersion,
		Version:        "0.0.3 BETA",
		ReleaseNotes:   "• Автоматическое обновление приложения прямо через интерфейс (In-App Update)\n• Реальное отображение скорости загрузки, размера и времени (ETA)\n• Перезапуск приложения в один клик после успешной установки\n• Обновленная версия 0.0.1 BETA\n• Исправления и улучшения дизайна Apple HIG",
		DownloadURL:    "https://github.com/Ziviy/zimap/releases",
		AssetURL:       "simulate",
		AssetName:      fmt.Sprintf("ZiMap-%s-%s.zip", runtime.GOOS, runtime.GOARCH),
		AssetSize:      26843545,
		IsNewer:        true,
		CanAutoUpdate:  true,
	}, nil
}

func (a *App) InstallUpdate(assetURL string) error {
	a.updateMutex.Lock()
	if a.updateRunning {
		a.updateMutex.Unlock()
		return fmt.Errorf("update is already running")
	}
	a.updateRunning = true
	ctx, cancel := context.WithCancel(context.Background())
	a.updateCancel = cancel
	a.updateMutex.Unlock()

	go func() {
		defer func() {
			a.updateMutex.Lock()
			a.updateRunning = false
			a.updateCancel = nil
			a.updateMutex.Unlock()
		}()

		if assetURL == "simulate" || strings.HasPrefix(assetURL, "simulate:") {
			a.runSimulatedUpdate(ctx)
			return
		}

		err := a.runRealUpdate(ctx, assetURL)
		if err != nil {
			if ctx.Err() != nil {
				a.emitEvent("update:stage", "cancelled")
				a.emitEvent("update:log", "Update cancelled by user")
			} else {
				a.emitEvent("update:stage", "error")
				a.emitEvent("update:error", err.Error())
				a.emitEvent("update:log", "Update error: "+err.Error())
			}
		}
	}()

	return nil
}

func (a *App) runRealUpdate(ctx context.Context, assetURL string) error {
	a.emitEvent("update:stage", "download")
	a.emitEvent("update:progress", 0)
	a.emitEvent("update:log", "Downloading update asset: "+assetURL)

	cacheDir, err := setup.CacheDirectory()
	if err != nil || cacheDir == "" {
		cacheDir = "cache"
	}
	workDir := filepath.Join(cacheDir, fmt.Sprintf("update-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create update workspace: %w", err)
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	filename := filepath.Base(assetURL)
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}
	downloadPath := filepath.Join(workDir, filename)

	downloader := download.NewStreaming()
	err = downloader.DownloadWithOptions(ctx, assetURL, downloadPath, download.DownloadOptions{
		ProgressCallback: func(downloaded, total int64, speedBps float64) {
			pct := 0
			if total > 0 {
				pct = int(float64(downloaded) / float64(total) * 100)
			}
			var eta string
			if speedBps > 0 && total > downloaded {
				remSec := float64(total-downloaded) / speedBps
				eta = formatEta(remSec)
			}
			a.emitEvent("update:progress", pct)
			a.emitEvent("update:metrics", map[string]string{
				"speed":      formatSpeed(speedBps),
				"downloaded": formatBytes(downloaded),
				"total":      formatBytes(total),
				"eta":        eta,
			})
		},
		LogCallback: func(msg string) {
			a.emitEvent("update:log", msg)
		},
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("download update asset failed: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Extraction
	a.emitEvent("update:stage", "extract")
	a.emitEvent("update:progress", 100)
	a.emitEvent("update:log", "Extracting package contents...")

	unpackedDir := filepath.Join(workDir, "unpacked")
	if err := os.MkdirAll(unpackedDir, 0755); err != nil {
		return fmt.Errorf("failed to create unpack directory: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(downloadPath))
	if ext == ".zip" {
		extracted := false
		if runtime.GOOS == "darwin" {
			cmd := exec.CommandContext(ctx, "/usr/bin/ditto", "-xk", downloadPath, unpackedDir)
			if err := cmd.Run(); err == nil {
				extracted = true
			}
		}
		if !extracted {
			if err := archive.ExtractArchive(downloadPath, unpackedDir); err != nil {
				return fmt.Errorf("failed to extract zip update: %w", err)
			}
		}
	} else if ext == ".gz" || ext == ".tgz" || ext == ".tar" {
		if err := archive.ExtractArchive(downloadPath, unpackedDir); err != nil {
			return fmt.Errorf("failed to extract tar update: %w", err)
		}
	} else {
		// Single executable
		dest := filepath.Join(unpackedDir, filename)
		if err := copyFile(downloadPath, dest); err != nil {
			return fmt.Errorf("failed to stage executable: %w", err)
		}
		_ = os.Chmod(dest, 0755)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Locate extracted payload
	a.emitEvent("update:stage", "install")
	a.emitEvent("update:log", "Locating executable payload...")

	var foundAppBundle string
	var foundExecutable string

	_ = filepath.Walk(unpackedDir, func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() && strings.HasSuffix(strings.ToLower(fi.Name()), ".app") {
			if foundAppBundle == "" {
				foundAppBundle = p
			}
			return filepath.SkipDir
		}
		if !fi.IsDir() {
			if runtime.GOOS == "windows" && strings.HasSuffix(strings.ToLower(fi.Name()), ".exe") {
				if foundExecutable == "" {
					foundExecutable = p
				}
			} else if runtime.GOOS != "windows" && (fi.Mode()&0111 != 0) {
				if foundExecutable == "" {
					foundExecutable = p
				}
			}
		}
		return nil
	})

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine current executable path: %w", err)
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	var installTarget string
	if runtime.GOOS == "darwin" && foundAppBundle != "" {
		idx := strings.Index(execPath, ".app/Contents/MacOS")
		if idx != -1 {
			installTarget = execPath[:idx+4]
		} else {
			installTarget = filepath.Join(filepath.Dir(execPath), filepath.Base(foundAppBundle))
		}

		backupTarget := installTarget + ".old"
		_ = os.RemoveAll(backupTarget)
		if err := os.Rename(installTarget, backupTarget); err != nil {
			return fmt.Errorf("failed to backup current bundle: %w", err)
		}

		if err := copyOrMoveDir(foundAppBundle, installTarget); err != nil {
			_ = os.Rename(backupTarget, installTarget)
			return fmt.Errorf("failed to install new app bundle: %w", err)
		}
		_ = os.RemoveAll(backupTarget)
		_ = exec.Command("xattr", "-rd", "com.apple.quarantine", installTarget).Run()
	} else {
		// Standalone executable
		sourceExec := foundExecutable
		if sourceExec == "" {
			return fmt.Errorf("no valid executable found in update archive")
		}
		installTarget = execPath
		backupTarget := execPath + ".old"
		_ = os.Remove(backupTarget)
		_ = os.Rename(execPath, backupTarget)

		if err := copyFile(sourceExec, execPath); err != nil {
			_ = os.Rename(backupTarget, execPath)
			return fmt.Errorf("failed to replace executable: %w", err)
		}
		_ = os.Chmod(execPath, 0755)
		if runtime.GOOS != "windows" {
			_ = os.Remove(backupTarget)
		}
	}

	a.emitEvent("update:stage", "ready")
	a.emitEvent("update:progress", 100)
	a.emitEvent("update:done", map[string]any{"ready": true, "target": installTarget})
	a.emitEvent("update:log", "Update installation completed successfully. Ready to restart.")
	return nil
}

func (a *App) runSimulatedUpdate(ctx context.Context) {
	a.emitEvent("update:stage", "download")
	a.emitEvent("update:log", "Simulating update download...")

	totalBytes := int64(26843545) // ~25.6 MB
	var downloaded int64
	chunk := totalBytes / 20

	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for downloaded < totalBytes {
		select {
		case <-ctx.Done():
			a.emitEvent("update:stage", "cancelled")
			a.emitEvent("update:log", "Simulated update cancelled")
			return
		case <-ticker.C:
			downloaded += chunk
			if downloaded > totalBytes {
				downloaded = totalBytes
			}
			pct := int(float64(downloaded) / float64(totalBytes) * 100)
			speed := float64(14 * 1024 * 1024)
			rem := float64(totalBytes-downloaded) / speed
			a.emitEvent("update:progress", pct)
			a.emitEvent("update:metrics", map[string]string{
				"speed":      "14.2 MB/s",
				"downloaded": formatBytes(downloaded),
				"total":      formatBytes(totalBytes),
				"eta":        fmt.Sprintf("%.0fs", rem),
			})
		}
	}

	select {
	case <-ctx.Done():
		a.emitEvent("update:stage", "cancelled")
		return
	default:
	}

	a.emitEvent("update:stage", "extract")
	a.emitEvent("update:log", "Simulating archive extraction...")
	time.Sleep(600 * time.Millisecond)

	select {
	case <-ctx.Done():
		a.emitEvent("update:stage", "cancelled")
		return
	default:
	}

	a.emitEvent("update:stage", "install")
	a.emitEvent("update:log", "Simulating app installation...")
	time.Sleep(600 * time.Millisecond)

	a.emitEvent("update:stage", "ready")
	a.emitEvent("update:progress", 100)
	a.emitEvent("update:done", map[string]any{"ready": true, "simulated": true})
	a.emitEvent("update:log", "Update ready to restart!")
}

func (a *App) CancelUpdate() error {
	a.updateMutex.Lock()
	defer a.updateMutex.Unlock()
	if a.updateCancel != nil {
		a.updateCancel()
		a.updateCancel = nil
	}
	a.updateRunning = false
	a.emitEvent("update:stage", "cancelled")
	return nil
}

func (a *App) RestartApp() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}
	execPath, _ = filepath.EvalSymlinks(execPath)

	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		idx := strings.Index(execPath, ".app/Contents/MacOS")
		if idx != -1 {
			bundlePath := execPath[:idx+4]
			cmd = exec.Command("open", "-n", bundlePath)
		} else {
			cmd = exec.Command(execPath)
		}
	} else {
		cmd = exec.Command(execPath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to restart application: %w", err)
	}

	if a.ctx != nil {
		wailsruntime.Quit(a.ctx)
	}
	os.Exit(0)
	return nil
}

func copyOrMoveDir(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	return copyDirectory(src, dst)
}

func copyDirectory(src, dst string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("/bin/cp", "-R", src, dst)
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			linkDest, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(linkDest, target)
		}
		return copyFile(path, target)
	})
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatEta(seconds float64) string {
	if seconds < 0 || seconds > 86400 {
		return "--"
	}
	s := int(seconds)
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	m := s / 60
	s = s % 60
	return fmt.Sprintf("%dm %ds", m, s)
}
