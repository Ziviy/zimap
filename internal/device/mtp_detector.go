package device

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// IsMTPPath returns true if the specified mount path represents an MTP device.
func IsMTPPath(mountPath string) bool {
	return strings.HasPrefix(mountPath, "mtp://")
}

// ScanMTPDevices inspects the USB bus using MTPService to locate any connected Garmin smartwatches.
func ScanMTPDevices(ctx context.Context, svc *MTPService) ([]GarminDevice, error) {
	if svc == nil {
		svc = NewMTPService()
	}

	if !svc.IsAvailable() {
		return nil, nil
	}

	res, err := svc.DetectDevice(ctx)
	if err != nil || res == nil || res.Device == nil || !res.Device.Connected {
		return nil, nil
	}

	d := res.Device
	storageID := d.Storage.StorageID
	if storageID == 0 {
		storageID = 65537
	}

	allMaps, _ := svc.ListMaps(ctx, storageID)
	if allMaps == nil {
		allMaps = []DeviceMap{}
	}

	var customMaps []DeviceMap
	var preinstalledMaps []DeviceMap
	for _, m := range allMaps {
		if m.IsPreinstalled || !m.IsCustom {
			preinstalledMaps = append(preinstalledMaps, m)
		} else {
			customMaps = append(customMaps, m)
		}
	}


	total := d.Storage.TotalBytes
	free := d.Storage.FreeBytes
	used := uint64(0)
	pct := float64(0)
	if total > free {
		used = total - free
		pct = (float64(used) / float64(total)) * 100.0
	}

	conflictMsg := ""
	if len(res.Conflicts) > 0 {
		conflictMsg = strings.Join(res.Conflicts, ", ")
	}

	friendlyName := strings.TrimSpace(d.FriendlyName)
	if friendlyName == "" {
		friendlyName = strings.TrimSpace(d.Model)
	}
	if friendlyName == "" {
		friendlyName = "Garmin Watch (MTP)"
	}

	dev := GarminDevice{
		MountPath:       fmt.Sprintf("mtp://storage-%d", storageID),
		VolumeName:      "MTP",
		ModelName:       d.Model,
		DisplayName:     friendlyName,
		UnitID:          d.SerialNumber,
		SoftwareVersion: d.DeviceVersion,
		Storage: StorageInfo{
			TotalBytes:     total,
			FreeBytes:      free,
			UsedBytes:      used,
			UsedPercentage: pct,
			FormattedTotal: FormatStorageBytes(total),
			FormattedFree:  FormatStorageBytes(free),
			FormattedUsed:  FormatStorageBytes(used),
		},
		CustomMaps:       customMaps,
		PreinstalledMaps: preinstalledMaps,
		TotalMapsCount:   len(allMaps),
		ConnectedAt:    time.Now(),
		IsMTP:          true,
		StorageID:      storageID,
		MTPConflict:    conflictMsg,
	}

	return []GarminDevice{dev}, nil
}
