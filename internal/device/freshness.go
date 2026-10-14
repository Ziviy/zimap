package device

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FAT32TimestampTolerance is the minimum time difference required to treat files as different,
// reflecting the 2-second FAT32 timestamp resolution on Garmin devices.
const FAT32TimestampTolerance = 2 * time.Second

// DeviceSafetyMarginBytes is the reserve memory (20 MB) kept for Garmin logs, GPX, and FIT files.
const DeviceSafetyMarginBytes = 20 * 1024 * 1024

// cleanMapName strips .img extension and cleans up hyphens/underscores for matching.
func cleanMapName(s string) string {
	s = strings.TrimSuffix(s, filepath.Ext(s))
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// CompareMaps correlates locally compiled maps with maps present on the Garmin device.
func CompareMaps(deviceMaps []DeviceMap, localMaps []LocalMapItem, deviceFreeBytes uint64) []MapFreshnessItem {
	var items []MapFreshnessItem
	matchedDeviceIndices := make(map[int]bool)

	// Step 1: Try to correlate each local map with a device map
	for _, local := range localMaps {
		matchedIndex := -1
		localClean := cleanMapName(local.Name)
		localRegionClean := cleanMapName(local.RegionID)
		localTitleClean := cleanMapName(local.Title)

		// Tier 1: Exact filename match (case-insensitive)
		for idx, devMap := range deviceMaps {
			if matchedDeviceIndices[idx] {
				continue
			}
			if strings.EqualFold(local.Name, devMap.FileName) {
				matchedIndex = idx
				break
			}
		}

		// Tier 2: Match by clean name, RegionID, or Title against device fileName / internalName
		if matchedIndex == -1 {
			for idx, devMap := range deviceMaps {
				if matchedDeviceIndices[idx] {
					continue
				}
				devClean := cleanMapName(devMap.FileName)
				devInternalClean := cleanMapName(devMap.InternalName)

				if (localRegionClean != "" && (localRegionClean == devClean || localRegionClean == devInternalClean)) ||
					(localClean != "" && (localClean == devClean || localClean == devInternalClean)) ||
					(localTitleClean != "" && (localTitleClean == devClean || localTitleClean == devInternalClean)) {
					matchedIndex = idx
					break
				}
			}
		}

		// Tier 3: If device map is generic "gmapsupp.img", match against its internal name
		if matchedIndex == -1 {
			for idx, devMap := range deviceMaps {
				if matchedDeviceIndices[idx] {
					continue
				}
				if strings.EqualFold(devMap.FileName, "gmapsupp.img") {
					devInternalClean := cleanMapName(devMap.InternalName)
					if devInternalClean != "" && (devInternalClean == localClean || devInternalClean == localRegionClean || devInternalClean == localTitleClean) {
						matchedIndex = idx
						break
					}
				}
			}
		}

		displayName := local.Title
		if displayName == "" {
			displayName = strings.TrimSuffix(local.Name, filepath.Ext(local.Name))
		}

		item := MapFreshnessItem{
			ID:           localClean,
			MapName:      displayName,
			FileName:     local.Name,
			LocalPath:    local.Path,
			LocalSize:    local.Size,
			LocalSizeStr: FormatStorageBytes(uint64(local.Size)),
			LocalModTime: local.ModTime,
		}

		if matchedIndex != -1 {
			matchedDeviceIndices[matchedIndex] = true
			devMap := deviceMaps[matchedIndex]

			item.DevicePath = devMap.FilePath
			item.DeviceSize = devMap.SizeBytes
			item.DeviceSizeStr = devMap.SizeFormatted
			item.DeviceModTime = devMap.ModTime
			item.DeviceDir = devMap.RelativeDir
			item.InternalName = devMap.InternalName

			item.IsPreinstalled = devMap.IsPreinstalled
			item.SignatureType = devMap.SignatureType

			if devMap.IsPreinstalled {
				// Safety: never overwrite preinstalled factory maps
				item.Status = StatusUpToDate
				item.StatusMessage = "preinstalled_protected"
				item.BytesToTransfer = 0
			} else {
				// Compare timestamps with FAT32 2-second tolerance
				timeDiff := local.ModTime.Sub(devMap.ModTime)
				if timeDiff > FAT32TimestampTolerance {
					item.Status = StatusUpdateAvailable
					days := int(timeDiff.Hours() / 24)
					item.DaysOutdated = days
					item.BytesToTransfer = local.Size
					item.StatusMessage = "update_available"
				} else {
					item.Status = StatusUpToDate
					item.BytesToTransfer = 0
					item.StatusMessage = "up_to_date"
				}
			}
		} else {
			// Not on device yet
			item.Status = StatusNotOnDevice
			item.BytesToTransfer = local.Size
			item.StatusMessage = "not_on_device"
		}

		// Calculate if device has enough storage (localSize + 20MB safety margin)
		needed := uint64(item.BytesToTransfer + DeviceSafetyMarginBytes)
		if item.BytesToTransfer == 0 || deviceFreeBytes >= needed {
			item.HasEnoughSpace = true
		} else {
			item.HasEnoughSpace = false
		}

		items = append(items, item)
	}

	// Step 2: Unmatched device maps are marked as only_on_device
	for idx, devMap := range deviceMaps {
		if matchedDeviceIndices[idx] {
			continue
		}

		displayName := devMap.InternalName
		if displayName == "" {
			displayName = ResolveGarminMapTitle(devMap.FileName, "")
		}

		items = append(items, MapFreshnessItem{
			ID:              cleanMapName(devMap.FileName),
			MapName:         displayName,
			FileName:        devMap.FileName,
			Status:          StatusOnlyOnDevice,
			StatusMessage:   "only_on_device",
			DevicePath:      devMap.FilePath,
			DeviceSize:      devMap.SizeBytes,
			DeviceSizeStr:   devMap.SizeFormatted,
			DeviceModTime:   devMap.ModTime,
			DeviceDir:       devMap.RelativeDir,
			InternalName:    devMap.InternalName,
			BytesToTransfer: 0,
			HasEnoughSpace:  true,
			IsPreinstalled:  devMap.IsPreinstalled,
			SignatureType:   devMap.SignatureType,
		})
	}

	// Step 3: Sort items by priority:
	// 1. update_available
	// 2. not_on_device
	// 3. up_to_date
	// 4. only_on_device
	// Within the same group, sort alphabetically by MapName
	statusPriority := map[FreshnessStatus]int{
		StatusUpdateAvailable: 1,
		StatusNotOnDevice:     2,
		StatusUpToDate:        3,
		StatusOnlyOnDevice:    4,
	}

	sort.Slice(items, func(i, j int) bool {
		pi := statusPriority[items[i].Status]
		pj := statusPriority[items[j].Status]
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(items[i].MapName) < strings.ToLower(items[j].MapName)
	})

	return items
}

// BuildFreshnessReport aggregates MapFreshnessItems into a full DeviceFreshnessReport.
func BuildFreshnessReport(
	mountPath string,
	deviceName string,
	deviceMaps []DeviceMap,
	localMaps []LocalMapItem,
	deviceFreeBytes uint64,
) DeviceFreshnessReport {
	items := CompareMaps(deviceMaps, localMaps, deviceFreeBytes)

	var upToDate, updateAvailable, notOnDevice, onlyOnDevice int
	var totalSyncBytes int64

	for _, it := range items {
		switch it.Status {
		case StatusUpToDate:
			upToDate++
		case StatusUpdateAvailable:
			updateAvailable++
			totalSyncBytes += it.BytesToTransfer
		case StatusNotOnDevice:
			notOnDevice++
			totalSyncBytes += it.BytesToTransfer
		case StatusOnlyOnDevice:
			onlyOnDevice++
		}
	}

	return DeviceFreshnessReport{
		MountPath:            mountPath,
		DeviceName:           deviceName,
		CheckedAt:            time.Now(),
		Items:                items,
		UpToDateCount:        upToDate,
		UpdateAvailableCount: updateAvailable,
		NotOnDeviceCount:     notOnDevice,
		OnlyOnDeviceCount:    onlyOnDevice,
		TotalSyncBytes:       totalSyncBytes,
		TotalSyncBytesStr:    FormatStorageBytes(uint64(totalSyncBytes)),
		DeviceFreeBytes:      deviceFreeBytes,
		DeviceFreeStr:        FormatStorageBytes(deviceFreeBytes),
	}
}
