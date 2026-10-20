package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"zimap/internal/download"
	"zimap/internal/setup"
)

type StorageMetrics struct {
	CacheSize int64  `json:"cacheSize"`
	TempSize  int64  `json:"tempSize"`
	MapsSize  int64  `json:"mapsSize"`
	ToolsSize int64  `json:"toolsSize"`
	TotalSize int64  `json:"totalSize"`
	DataPath  string `json:"dataPath"`
}

type StorageCleanOptions struct {
	ClearCache    bool `json:"clearCache"`
	ClearTemp     bool `json:"clearTemp"`
	ClearMaps     bool `json:"clearMaps"`
	ClearTools    bool `json:"clearTools"`
	ClearSettings bool `json:"clearSettings"`
}

func pathSize(p string) int64 {
	info, err := os.Stat(p)
	if err != nil {
		return 0
	}
	if !info.IsDir() {
		return info.Size()
	}
	var total int64
	_ = filepath.Walk(p, func(_ string, fi os.FileInfo, err error) error {
		if err == nil && fi != nil && !fi.IsDir() {
			total += fi.Size()
		}
		return nil
	})
	return total
}

func isOutputTempEntry(name string) bool {
	return strings.HasPrefix(name, ".download-") ||
		strings.HasPrefix(name, ".garmin-map-") ||
		strings.HasPrefix(name, ".zimap-") ||
		strings.HasPrefix(name, ".splitter") ||
		strings.HasPrefix(name, ".previous-map-") ||
		strings.HasSuffix(name, ".tmp") ||
		strings.HasSuffix(name, ".download")
}

func isToolsTempEntry(name string) bool {
	return strings.HasPrefix(name, ".mkgmap-update-") ||
		strings.HasPrefix(name, ".jdk-update-") ||
		strings.HasPrefix(name, ".splitter-update-") ||
		strings.HasPrefix(name, ".previous-") ||
		strings.HasSuffix(name, ".tmp")
}

func isCacheTempEntry(name string) bool {
	return strings.HasPrefix(name, ".update-") || strings.HasSuffix(name, ".tmp")
}

func isManagedToolEntry(name string) bool {
	return name == "jdk" || name == "mkgmap" || name == "splitter"
}

func isMapOutputEntry(name string, isDir bool) bool {
	return !isDir && (strings.EqualFold(filepath.Ext(name), ".img") || strings.HasSuffix(name, ".meta.json"))
}

func (a *App) GetStorageMetrics() (StorageMetrics, error) {
	var metrics StorageMetrics
	if rootDir, err := setup.ProjectRoot(); err == nil {
		metrics.DataPath = rootDir
	}

	// 1. Cache size (excluding proxy-config.json) and cache temp dirs
	cacheDir := getCacheDir()
	if entries, err := os.ReadDir(cacheDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if name == "proxy-config.json" {
				continue
			}
			sz := pathSize(filepath.Join(cacheDir, name))
			if isCacheTempEntry(name) {
				metrics.TempSize += sz
			} else {
				metrics.CacheSize += sz
			}
		}
	}

	// 2. Maps size (.img and companion .meta.json in outputDir) & output temp workspaces
	outDir, _ := setup.OutputDirectory()
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if isMapOutputEntry(name, e.IsDir()) {
				metrics.MapsSize += pathSize(filepath.Join(outDir, name))
			} else if isOutputTempEntry(name) {
				metrics.TempSize += pathSize(filepath.Join(outDir, name))
			}
		}
	}

	// 3. Tools size (jdk, mkgmap, splitter) & tool update temp dirs
	toolsDir, _ := setup.ToolsDirectory()
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if isToolsTempEntry(name) {
				metrics.TempSize += pathSize(filepath.Join(toolsDir, name))
			} else if isManagedToolEntry(name) {
				metrics.ToolsSize += pathSize(filepath.Join(toolsDir, name))
			}
		}
	}

	// 4. System temp entries
	if sysEntries, err := os.ReadDir(os.TempDir()); err == nil {
		for _, e := range sysEntries {
			name := e.Name()
			if strings.HasPrefix(name, "garmin-map-") || strings.HasPrefix(name, "zimap-") {
				metrics.TempSize += pathSize(filepath.Join(os.TempDir(), name))
			}
		}
	}

	metrics.TotalSize = metrics.CacheSize + metrics.TempSize + metrics.MapsSize + metrics.ToolsSize
	return metrics, nil
}

func (a *App) ClearTempAndCacheFiles(clearCache bool, clearTemp bool) (int64, error) {
	return a.CleanStorage(StorageCleanOptions{
		ClearCache: clearCache,
		ClearTemp:  clearTemp,
	})
}

func (a *App) CleanStorage(opts StorageCleanOptions) (int64, error) {
	a.jobMutex.Lock()
	isRunning := a.cancelJob != nil
	a.jobMutex.Unlock()
	if isRunning && (opts.ClearTemp || opts.ClearMaps || opts.ClearTools) {
		return 0, fmt.Errorf("cannot clear temporary files while a map build is running")
	}

	var freedBytes int64
	outDir, _ := setup.OutputDirectory()
	toolsDir, _ := setup.ToolsDirectory()
	cacheDir := getCacheDir()

	if opts.ClearTemp {
		if entries, err := os.ReadDir(outDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if isOutputTempEntry(name) {
					p := filepath.Join(outDir, name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}

		if entries, err := os.ReadDir(toolsDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if isToolsTempEntry(name) {
					p := filepath.Join(toolsDir, name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}

		if entries, err := os.ReadDir(cacheDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if isCacheTempEntry(name) {
					p := filepath.Join(cacheDir, name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}

		if sysEntries, err := os.ReadDir(os.TempDir()); err == nil {
			for _, e := range sysEntries {
				name := e.Name()
				if strings.HasPrefix(name, "garmin-map-") || strings.HasPrefix(name, "zimap-") {
					p := filepath.Join(os.TempDir(), name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}
	}

	if opts.ClearCache {
		if entries, err := os.ReadDir(cacheDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if name == "proxy-config.json" || isCacheTempEntry(name) {
					continue
				}
				p := filepath.Join(cacheDir, name)
				sz := pathSize(p)
				if err := os.RemoveAll(p); err == nil {
					freedBytes += sz
				}
			}
		}

		a.catalogMutex.Lock()
		a.catalogCache = nil
		a.providerCatalogs = make(map[string]*CatalogData)
		a.providerTimes = make(map[string]time.Time)
		a.catalogMutex.Unlock()
	}

	if opts.ClearMaps {
		if entries, err := os.ReadDir(outDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if isMapOutputEntry(name, e.IsDir()) {
					p := filepath.Join(outDir, name)
					sz := pathSize(p)
					if err := os.Remove(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}
	}

	if opts.ClearTools {
		if entries, err := os.ReadDir(toolsDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if isManagedToolEntry(name) {
					p := filepath.Join(toolsDir, name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}
	}

	if opts.ClearSettings {
		cfgFile := filepath.Join(cacheDir, "proxy-config.json")
		if sz := pathSize(cfgFile); sz > 0 {
			if err := os.Remove(cfgFile); err == nil {
				freedBytes += sz
			}
		} else {
			_ = os.Remove(cfgFile)
		}
		download.SetGlobalProxy(download.ProxyConfig{
			UseSystemProxy: true,
			CustomProxy:    "",
		})
	}

	// If full cleanup is requested, remove empty subdirectories and root data dir
	// (os.Remove only succeeds if a directory is completely empty).
	if opts.ClearCache && opts.ClearTemp && opts.ClearMaps && opts.ClearTools && opts.ClearSettings {
		_ = os.Remove(outDir)
		_ = os.Remove(cacheDir)
		_ = os.Remove(toolsDir)
		if rootDir, err := setup.ProjectRoot(); err == nil && rootDir != "" {
			_ = os.Remove(rootDir)
		}
	}

	return freedBytes, nil
}

func (a *App) RevealDataDirectory() error {
	rootDir, err := setup.ProjectRoot()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return exec.Command("explorer", rootDir).Run()
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", rootDir).Run()
	}
	if runtime.GOOS == "linux" {
		if p, err := exec.LookPath("xdg-open"); err == nil {
			return exec.Command(p, rootDir).Run()
		}
	}
	return nil
}
