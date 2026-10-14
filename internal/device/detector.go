package device

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ScanDevices inspects candidate volumes across the operating system and returns all detected Garmin devices.
func ScanDevices() ([]GarminDevice, error) {
	var candidates []string

	// Allow environment override for local testing / mock volumes
	if mockDir := os.Getenv("ZIMAP_MOCK_GARMIN_DIR"); mockDir != "" {
		candidates = append(candidates, mockDir)
	}

	platformCandidates := scanCandidateMounts()
	candidates = append(candidates, platformCandidates...)

	var devices []GarminDevice
	seenPaths := make(map[string]bool)

	for _, mount := range candidates {
		cleanMount := filepath.Clean(mount)
		if seenPaths[cleanMount] {
			continue
		}
		seenPaths[cleanMount] = true

		dev, err := InspectGarminMount(cleanMount)
		if err == nil && dev != nil {
			devices = append(devices, *dev)
		}
	}

	// Also check for MTP devices (e.g. smartwatches on macOS)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if mtpDevs, err := ScanMTPDevices(ctx, nil); err == nil && len(mtpDevs) > 0 {
		for _, md := range mtpDevs {
			devices = append(devices, md)
		}
	}

	return devices, nil
}

// InspectGarminMount analyzes a specific filesystem mount directory to see if it is a Garmin device.
func InspectGarminMount(mountPath string) (*GarminDevice, error) {
	if IsMTPPath(mountPath) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		mtpDevs, err := ScanMTPDevices(ctx, nil)
		if err == nil {
			for _, d := range mtpDevs {
				if d.MountPath == mountPath || mountPath == "mtp://" {
					return &d, nil
				}
			}
			if len(mtpDevs) > 0 {
				return &mtpDevs[0], nil
			}
		}
		return nil, fmt.Errorf("MTP device not found: %s", mountPath)
	}

	info, err := os.Stat(mountPath)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", mountPath)
	}

	garminDir := filepath.Join(mountPath, "GARMIN")
	garminDirInfo, err := os.Stat(garminDir)
	hasGarminDir := err == nil && garminDirInfo.IsDir()

	// Check for GarminDevice.xml in GARMIN/ or directly in mount root
	xmlPath := filepath.Join(garminDir, "GarminDevice.xml")
	if _, err := os.Stat(xmlPath); os.IsNotExist(err) {
		altXML := filepath.Join(mountPath, "GarminDevice.xml")
		if _, err2 := os.Stat(altXML); err2 == nil {
			xmlPath = altXML
		} else {
			xmlPath = ""
		}
	}

	// Must have either GarminDevice.xml or a GARMIN directory
	if xmlPath == "" && !hasGarminDir {
		return nil, fmt.Errorf("not a Garmin volume: %s", mountPath)
	}

	volName := filepath.Base(mountPath)
	if volName == "" || volName == "/" || volName == "\\" {
		volName = "GARMIN"
	}

	dev := &GarminDevice{
		MountPath:   mountPath,
		VolumeName:  volName,
		ModelName:   "Garmin Device",
		DisplayName: volName,
		ConnectedAt: time.Now(),
		CustomMaps:  []DeviceMap{},
	}

	// Parse XML if available
	if xmlPath != "" {
		if xmlDoc, err := ParseGarminDeviceXMLFile(xmlPath); err == nil && xmlDoc != nil {
			if xmlDoc.Model.Description != "" {
				dev.ModelName = strings.TrimSpace(xmlDoc.Model.Description)
			}
			if xmlDoc.DisplayName != "" {
				dev.DisplayName = strings.TrimSpace(xmlDoc.DisplayName)
			} else {
				dev.DisplayName = dev.ModelName
			}
			dev.UnitID = strings.TrimSpace(xmlDoc.UnitID)
			dev.SoftwareVersion = strings.TrimSpace(xmlDoc.Model.SoftwareVersion)
		}
	}

	// Read storage capacity
	if storage, err := GetStorageInfo(mountPath); err == nil {
		dev.Storage = storage
	}

	// Scan maps in GARMIN and GARMIN/Map
	scanDirs := []string{}
	if hasGarminDir {
		scanDirs = append(scanDirs, garminDir)
		mapSubdir := filepath.Join(garminDir, "Map")
		if subInfo, err := os.Stat(mapSubdir); err == nil && subInfo.IsDir() {
			scanDirs = append(scanDirs, mapSubdir)
		}
	}

	for _, dir := range scanDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		relDir := "GARMIN"
		if strings.EqualFold(filepath.Base(dir), "Map") {
			relDir = "GARMIN/Map"
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.EqualFold(filepath.Ext(name), ".img") {
				continue
			}

			fullPath := filepath.Join(dir, name)
			fileInfo, err := entry.Info()
			if err != nil {
				continue
			}

			internalName, _ := ReadMapInternalName(fullPath)
			if internalName == "" {
				internalName = strings.TrimSuffix(name, filepath.Ext(name))
			}

			isPreinstalled, sigType := DetectPreinstalledMap(name, fullPath, internalName)
			dev.TotalMapsCount++

			if isPreinstalled {
				dev.PreinstalledMaps = append(dev.PreinstalledMaps, DeviceMap{
					FileName:       name,
					FilePath:       fullPath,
					InternalName:   ResolveGarminMapTitle(name, internalName),
					SizeBytes:      fileInfo.Size(),
					SizeFormatted:  FormatStorageBytes(uint64(fileInfo.Size())),
					ModTime:        fileInfo.ModTime(),
					IsCustom:       false,
					IsPreinstalled: true,
					SignatureType:  sigType,
					RelativeDir:    relDir,
				})
			} else {
				dev.CustomMaps = append(dev.CustomMaps, DeviceMap{
					FileName:       name,
					FilePath:       fullPath,
					InternalName:   internalName,
					SizeBytes:      fileInfo.Size(),
					SizeFormatted:  FormatStorageBytes(uint64(fileInfo.Size())),
					ModTime:        fileInfo.ModTime(),
					IsCustom:       true,
					IsPreinstalled: false,
					SignatureType:  "",
					RelativeDir:    relDir,
				})
			}
		}
	}

	// Sort custom maps newest first
	sort.Slice(dev.CustomMaps, func(i, j int) bool {
		return dev.CustomMaps[i].ModTime.After(dev.CustomMaps[j].ModTime)
	})

	return dev, nil
}
