package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zimap/internal/setup"
)

type StorageMetrics struct {
	CacheSize int64 `json:"cacheSize"`
	TempSize  int64 `json:"tempSize"`
	MapsSize  int64 `json:"mapsSize"`
	TotalSize int64 `json:"totalSize"`
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

func (a *App) GetStorageMetrics() (StorageMetrics, error) {
	var metrics StorageMetrics

	// 1. Cache size (excluding proxy-config.json)
	cacheDir := getCacheDir()
	if entries, err := os.ReadDir(cacheDir); err == nil {
		for _, e := range entries {
			if e.Name() == "proxy-config.json" {
				continue
			}
			metrics.CacheSize += pathSize(filepath.Join(cacheDir, e.Name()))
		}
	}

	// 2. Maps size (.img and companion .meta.json in outputDir)
	outDir, _ := setup.OutputDirectory()
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if !e.IsDir() && (strings.EqualFold(filepath.Ext(name), ".img") || strings.HasSuffix(name, ".meta.json")) {
				metrics.MapsSize += pathSize(filepath.Join(outDir, name))
			}
		}
	}

	// 3. Temp size (workdirs, downloads, tool update dirs)
	if entries, err := os.ReadDir(outDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".download-") || strings.HasPrefix(name, ".garmin-map-") || strings.HasPrefix(name, ".splitter") || strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".download") {
				metrics.TempSize += pathSize(filepath.Join(outDir, name))
			}
		}
	}

	toolsDir, _ := setup.ToolsDirectory()
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, ".mkgmap-update-") || strings.HasPrefix(name, ".jdk-update-") || strings.HasPrefix(name, ".splitter-update-") || strings.HasPrefix(name, ".previous-") || strings.HasSuffix(name, ".tmp") {
				metrics.TempSize += pathSize(filepath.Join(toolsDir, name))
			}
		}
	}

	if sysEntries, err := os.ReadDir(os.TempDir()); err == nil {
		for _, e := range sysEntries {
			name := e.Name()
			if strings.HasPrefix(name, "garmin-map-") || strings.HasPrefix(name, "zimap-") {
				metrics.TempSize += pathSize(filepath.Join(os.TempDir(), name))
			}
		}
	}

	metrics.TotalSize = metrics.CacheSize + metrics.TempSize + metrics.MapsSize
	return metrics, nil
}

func (a *App) ClearTempAndCacheFiles(clearCache bool, clearTemp bool) (int64, error) {
	a.jobMutex.Lock()
	isRunning := a.cancelJob != nil
	a.jobMutex.Unlock()
	if isRunning && clearTemp {
		return 0, fmt.Errorf("cannot clear temporary files while a map build is running")
	}

	var freedBytes int64

	if clearTemp {
		outDir, _ := setup.OutputDirectory()
		if entries, err := os.ReadDir(outDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, ".download-") || strings.HasPrefix(name, ".garmin-map-") || strings.HasPrefix(name, ".splitter") || strings.HasSuffix(name, ".tmp") || strings.HasSuffix(name, ".download") {
					p := filepath.Join(outDir, name)
					sz := pathSize(p)
					if err := os.RemoveAll(p); err == nil {
						freedBytes += sz
					}
				}
			}
		}

		toolsDir, _ := setup.ToolsDirectory()
		if entries, err := os.ReadDir(toolsDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if strings.HasPrefix(name, ".mkgmap-update-") || strings.HasPrefix(name, ".jdk-update-") || strings.HasPrefix(name, ".splitter-update-") || strings.HasPrefix(name, ".previous-") || strings.HasSuffix(name, ".tmp") {
					p := filepath.Join(toolsDir, name)
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

	if clearCache {
		cacheDir := getCacheDir()
		if entries, err := os.ReadDir(cacheDir); err == nil {
			for _, e := range entries {
				name := e.Name()
				if name == "proxy-config.json" {
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

	return freedBytes, nil
}
