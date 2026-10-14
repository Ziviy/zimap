package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"zimap/internal/device"
	"zimap/internal/setup"
)

// GetConnectedDevices scans mounted volumes and returns all detected GPS devices
// and their custom user-installed maps.
func (a *App) GetConnectedDevices() ([]device.GarminDevice, error) {
	devices, err := device.ScanDevices()
	if err != nil {
		return []device.GarminDevice{}, err
	}
	if devices == nil {
		devices = []device.GarminDevice{}
	}
	return devices, nil
}

// GetConnectedGarminDevices is retained for backwards compatibility with existing bindings.
func (a *App) GetConnectedGarminDevices() ([]device.GarminDevice, error) {
	return a.GetConnectedDevices()
}

// GetDeviceUSBStatus checks if a compatible GPS device is connected via USB even if its filesystem
// is not mounted (e.g. MTP mode on macOS), returning diagnostic guidance.
func (a *App) GetDeviceUSBStatus() (device.USBStatus, error) {
	devices, _ := device.ScanDevices()
	status := device.CheckGarminUSB(len(devices))
	return status, nil
}

// GetGarminUSBStatus is retained for backwards compatibility with existing bindings.
func (a *App) GetGarminUSBStatus() (device.USBStatus, error) {
	return a.GetDeviceUSBStatus()
}

// DeleteDeviceMap safely removes a custom user map (.img) from a mounted Garmin device.
// Rejects any attempt to delete factory system maps (e.g. gmapbmap, gmaptz, TopoActive tiles)
// or files outside the GARMIN directory.
func (a *App) DeleteDeviceMap(mountPath, filePath string) error {
	if device.IsMTPPath(mountPath) {
		baseFile := filepath.Base(filePath)
		if device.IsSystemMap(baseFile) {
			return device.ErrSystemMapProtected
		}
		storageID := device.ParseMTPStorageID(mountPath)
		return a.getMTPService().DeleteMap(context.Background(), storageID, 0, baseFile)
	}

	validatedPath, err := device.ValidateDeletePath(mountPath, filePath)
	if err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	if err := os.Remove(validatedPath); err != nil {
		return fmt.Errorf("failed to delete map file: %w", err)
	}

	// Clean up optional companion .sum or .meta.json files if present
	baseNoExt := strings.TrimSuffix(validatedPath, filepath.Ext(validatedPath))
	_ = os.Remove(baseNoExt + ".sum")
	_ = os.Remove(baseNoExt + ".meta.json")

	return nil
}

// EjectDevice cleanly unmounts and ejects the specified device volume.
func (a *App) EjectDevice(mountPath string) error {
	if mountPath == "" {
		return errors.New("mount path cannot be empty")
	}

	if device.IsMTPPath(mountPath) {
		return nil
	}

	return device.EjectDevice(mountPath)
}

// EjectGarminDevice is retained for backwards compatibility with existing bindings.
func (a *App) EjectGarminDevice(mountPath string) error {
	return a.EjectDevice(mountPath)
}

// RevealDevicePath opens the specified map file or directory on the Garmin device in Finder or Explorer.
func (a *App) RevealDevicePath(filePath string) error {
	if filePath == "" {
		return errors.New("path cannot be empty")
	}

	if device.IsMTPPath(filePath) {
		return errors.New("MTP devices do not support direct file reveal in Finder")
	}

	cleanPath := filepath.Clean(filePath)
	if _, err := os.Stat(cleanPath); err != nil {
		return fmt.Errorf("target does not exist: %w", err)
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", "/select,", cleanPath).Run()
	case "darwin":
		return exec.Command("open", "-R", cleanPath).Run()
	case "linux":
		if p, err := exec.LookPath("xdg-open"); err == nil {
			info, err := os.Stat(cleanPath)
			if err == nil && info.IsDir() {
				return exec.Command(p, cleanPath).Run()
			}
			return exec.Command(p, filepath.Dir(cleanPath)).Run()
		}
	}
	return nil
}

// CheckDeviceFreshness inspects maps on the connected Garmin device and compares them
// against local compiled maps in output/ to determine version freshness.
func (a *App) CheckDeviceFreshness(mountPath string) (*device.DeviceFreshnessReport, error) {
	if mountPath == "" {
		return nil, errors.New("mount path cannot be empty")
	}

	dev, err := device.InspectGarminMount(mountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect Garmin device: %w", err)
	}

	outputs, err := a.ListOutputs()
	if err != nil {
		outputs = []OutputItem{}
	}

	localMaps := make([]device.LocalMapItem, 0, len(outputs))
	for _, out := range outputs {
		title := ""
		regionID := ""
		if out.Meta != nil {
			title = out.Meta.Name
			regionID = out.Meta.RegionID
		}
		if title == "" {
			title = strings.TrimSuffix(out.Name, filepath.Ext(out.Name))
		}
		localMaps = append(localMaps, device.LocalMapItem{
			Name:      out.Name,
			Path:      out.Path,
			Size:      out.Size,
			ModTime:   out.ModTime,
			SourceURL: out.SourceURL,
			RegionID:  regionID,
			Title:     title,
		})
	}

	report := device.BuildFreshnessReport(
		dev.MountPath,
		dev.DisplayName,
		dev.CustomMaps,
		localMaps,
		dev.Storage.FreeBytes,
	)
	report.PreinstalledMaps = dev.PreinstalledMaps
	report.PreinstalledCount = len(dev.PreinstalledMaps)

	return &report, nil
}

// SyncMapToDevice streams a locally compiled map (.img) to the Garmin device in a background goroutine,
// emitting real-time progress events.
func (a *App) SyncMapToDevice(mountPath string, fileName string) error {
	if mountPath == "" || fileName == "" {
		return errors.New("mount path and file name cannot be empty")
	}

	outDir, err := setup.OutputDirectory()
	if err != nil {
		return fmt.Errorf("failed to get output directory: %w", err)
	}

	cleanFile := filepath.Base(fileName)
	if !strings.EqualFold(filepath.Ext(cleanFile), ".img") {
		cleanFile += ".img"
	}

	srcPath := filepath.Join(outDir, cleanFile)
	if _, err := os.Stat(srcPath); err != nil {
		return fmt.Errorf("local map file not found: %s", cleanFile)
	}

	sm := a.getSyncManager()
	if sm.IsSyncing() {
		return device.ErrSyncInProgress
	}

	// Route MTP Watch Transfers
	if device.IsMTPPath(mountPath) {
		storageID := device.ParseMTPStorageID(mountPath)

		a.emitEvent("device:sync_start", map[string]interface{}{
			"mapId":    cleanFile,
			"fileName": cleanFile,
		})

		return sm.StartSyncMTP(context.Background(), a.getMTPService(), storageID, srcPath, "GARMIN", func(e device.SyncProgressEvent) {
			a.emitEvent("device:sync_progress", e)
		}, func(err error) {
			if err != nil {
				if errors.Is(err, context.Canceled) {
					a.emitEvent("device:sync_cancelled", map[string]interface{}{
						"mapId":    cleanFile,
						"fileName": cleanFile,
					})
				} else {
					a.emitEvent("device:sync_error", map[string]interface{}{
						"mapId":    cleanFile,
						"fileName": cleanFile,
						"error":    err.Error(),
					})
				}
				return
			}

			a.emitEvent("device:sync_done", map[string]interface{}{
				"mapId":    cleanFile,
				"fileName": cleanFile,
			})
		})
	}

	destPath, err := device.ResolveDestinationPath(mountPath, cleanFile)
	if err != nil {
		return fmt.Errorf("failed to resolve device destination: %w", err)
	}

	// Rule 2: Non-blocking IPC. Start background sync with immediate busy state.
	a.emitEvent("device:sync_start", map[string]interface{}{
		"mapId":    cleanFile,
		"fileName": cleanFile,
	})

	return sm.StartSync(context.Background(), srcPath, destPath, func(e device.SyncProgressEvent) {
		a.emitEvent("device:sync_progress", e)
	}, func(err error) {
		if err != nil {
			if errors.Is(err, context.Canceled) {
				a.emitEvent("device:sync_cancelled", map[string]interface{}{
					"mapId":    cleanFile,
					"fileName": cleanFile,
				})
			} else {
				a.emitEvent("device:sync_error", map[string]interface{}{
					"mapId":    cleanFile,
					"fileName": cleanFile,
					"error":    err.Error(),
				})
			}
			return
		}

		a.emitEvent("device:sync_done", map[string]interface{}{
			"mapId":    cleanFile,
			"fileName": cleanFile,
		})
	})
}

// SyncAllOutdatedMaps sequentially copies all outdated and missing maps to the Garmin device.
func (a *App) SyncAllOutdatedMaps(mountPath string) error {
	if mountPath == "" {
		return errors.New("mount path cannot be empty")
	}

	report, err := a.CheckDeviceFreshness(mountPath)
	if err != nil {
		return err
	}

	var syncList []device.MapFreshnessItem
	for _, it := range report.Items {
		if it.Status == device.StatusUpdateAvailable || it.Status == device.StatusNotOnDevice {
			syncList = append(syncList, it)
		}
	}

	if len(syncList) == 0 {
		return nil
	}

	sm := a.getSyncManager()
	if sm.IsSyncing() {
		return device.ErrSyncInProgress
	}

	outDir, err := setup.OutputDirectory()
	if err != nil {
		return err
	}

	// Route MTP Batch Sync
	if device.IsMTPPath(mountPath) {
		storageID := device.ParseMTPStorageID(mountPath)

		go func() {
			totalCount := len(syncList)
			a.emitEvent("device:sync_all_start", map[string]interface{}{
				"total": totalCount,
			})

			for idx, it := range syncList {
				cleanFile := filepath.Base(it.FileName)
				srcPath := filepath.Join(outDir, cleanFile)

				a.emitEvent("device:sync_start", map[string]interface{}{
					"mapId":        cleanFile,
					"fileName":     cleanFile,
					"mapName":      it.MapName,
					"currentIndex": idx + 1,
					"total":        totalCount,
				})

				err = sm.SyncMTP(context.Background(), a.getMTPService(), storageID, srcPath, "GARMIN", func(e device.SyncProgressEvent) {
					a.emitEvent("device:sync_progress", e)
				})

				if err != nil {
					if errors.Is(err, context.Canceled) {
						a.emitEvent("device:sync_cancelled", map[string]interface{}{
							"mapId":    cleanFile,
							"fileName": cleanFile,
						})
					} else {
						a.emitEvent("device:sync_error", map[string]interface{}{
							"mapId":    cleanFile,
							"fileName": cleanFile,
							"error":    err.Error(),
						})
					}
					return
				}

				a.emitEvent("device:sync_done", map[string]interface{}{
					"mapId":        cleanFile,
					"fileName":     cleanFile,
					"currentIndex": idx + 1,
					"total":        totalCount,
				})
			}

			a.emitEvent("device:sync_all_done", map[string]interface{}{
				"total": totalCount,
			})
		}()

		return nil
	}

	go func() {
		totalCount := len(syncList)
		a.emitEvent("device:sync_all_start", map[string]interface{}{
			"total": totalCount,
		})

		for idx, it := range syncList {
			cleanFile := filepath.Base(it.FileName)
			srcPath := filepath.Join(outDir, cleanFile)
			destPath, err := device.ResolveDestinationPath(mountPath, cleanFile)
			if err != nil {
				a.emitEvent("device:sync_error", map[string]interface{}{
					"mapId":    cleanFile,
					"fileName": cleanFile,
					"error":    err.Error(),
				})
				return
			}

			a.emitEvent("device:sync_start", map[string]interface{}{
				"mapId":        cleanFile,
				"fileName":     cleanFile,
				"mapName":      it.MapName,
				"currentIndex": idx + 1,
				"total":        totalCount,
			})

			err = sm.Sync(context.Background(), srcPath, destPath, func(e device.SyncProgressEvent) {
				a.emitEvent("device:sync_progress", e)
			})

			if err != nil {
				if errors.Is(err, context.Canceled) {
					a.emitEvent("device:sync_cancelled", map[string]interface{}{
						"mapId":    cleanFile,
						"fileName": cleanFile,
					})
				} else {
					a.emitEvent("device:sync_error", map[string]interface{}{
						"mapId":    cleanFile,
						"fileName": cleanFile,
						"error":    err.Error(),
					})
				}
				return
			}

			a.emitEvent("device:sync_done", map[string]interface{}{
				"mapId":        cleanFile,
				"fileName":     cleanFile,
				"currentIndex": idx + 1,
				"total":        totalCount,
			})
		}

		a.emitEvent("device:sync_all_done", map[string]interface{}{
			"total": totalCount,
		})
	}()

	return nil
}

// CancelDeviceSync cancels any ongoing map transfer to the Garmin device.
func (a *App) CancelDeviceSync() error {
	a.getSyncManager().Cancel()
	return nil
}

