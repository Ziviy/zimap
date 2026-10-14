package device

import (
	"testing"
	"time"
)

func TestCompareMaps_AllStatuses(t *testing.T) {
	now := time.Now()

	// 1. Up to date map: device timestamp is within 2s of local build
	// 2. Outdated map: device timestamp is 5 days older than local build
	// 3. Local only: not present on device
	// 4. Device only: not present in local outputs
	localMaps := []LocalMapItem{
		{
			Name:     "spb.img",
			Path:     "/local/spb.img",
			Size:     100 * 1024 * 1024,
			ModTime:  now,
			RegionID: "russia-spb",
			Title:    "Санкт-Петербург",
		},
		{
			Name:     "moscow.img",
			Path:     "/local/moscow.img",
			Size:     200 * 1024 * 1024,
			ModTime:  now,
			RegionID: "russia-moscow",
			Title:    "Москва",
		},
		{
			Name:     "crimea.img",
			Path:     "/local/crimea.img",
			Size:     50 * 1024 * 1024,
			ModTime:  now,
			RegionID: "crimea",
			Title:    "Крым",
		},
	}

	deviceMaps := []DeviceMap{
		{
			FileName:      "spb.img",
			FilePath:      "/device/GARMIN/spb.img",
			SizeBytes:     100 * 1024 * 1024,
			SizeFormatted: "100.0 MB",
			ModTime:       now.Add(-1 * time.Second), // within 2s FAT32 tolerance
			RelativeDir:   "GARMIN",
			InternalName:  "OSM St. Petersburg",
		},
		{
			FileName:      "moscow.img",
			FilePath:      "/device/GARMIN/Map/moscow.img",
			SizeBytes:     190 * 1024 * 1024,
			SizeFormatted: "190.0 MB",
			ModTime:       now.Add(-72 * time.Hour), // 3 days older
			RelativeDir:   "GARMIN/Map",
			InternalName:  "OSM Moscow",
		},
		{
			FileName:      "alps.img",
			FilePath:      "/device/GARMIN/alps.img",
			SizeBytes:     300 * 1024 * 1024,
			SizeFormatted: "300.0 MB",
			ModTime:       now.Add(-240 * time.Hour),
			RelativeDir:   "GARMIN",
			InternalName:  "Alps Hiking",
		},
	}

	freeBytes := uint64(500 * 1024 * 1024) // 500 MB free
	report := BuildFreshnessReport("/device", "Garmin Fenix 7", deviceMaps, localMaps, freeBytes)

	if report.UpToDateCount != 1 {
		t.Errorf("expected 1 up to date map, got %d", report.UpToDateCount)
	}
	if report.UpdateAvailableCount != 1 {
		t.Errorf("expected 1 update available map, got %d", report.UpdateAvailableCount)
	}
	if report.NotOnDeviceCount != 1 {
		t.Errorf("expected 1 not on device map, got %d", report.NotOnDeviceCount)
	}
	if report.OnlyOnDeviceCount != 1 {
		t.Errorf("expected 1 only on device map, got %d", report.OnlyOnDeviceCount)
	}

	expectedSyncBytes := int64(200*1024*1024 + 50*1024*1024) // moscow (200MB) + crimea (50MB)
	if report.TotalSyncBytes != expectedSyncBytes {
		t.Errorf("expected total sync bytes %d, got %d", expectedSyncBytes, report.TotalSyncBytes)
	}

	// Verify sorting order: update_available (moscow), not_on_device (crimea), up_to_date (spb), only_on_device (alps)
	if len(report.Items) != 4 {
		t.Fatalf("expected 4 items in report, got %d", len(report.Items))
	}

	if report.Items[0].Status != StatusUpdateAvailable || report.Items[0].FileName != "moscow.img" {
		t.Errorf("expected first item to be moscow.img with update_available, got %s (%s)", report.Items[0].FileName, report.Items[0].Status)
	}
	if report.Items[0].DaysOutdated != 3 {
		t.Errorf("expected moscow to be 3 days outdated, got %d", report.Items[0].DaysOutdated)
	}

	if report.Items[1].Status != StatusNotOnDevice || report.Items[1].FileName != "crimea.img" {
		t.Errorf("expected second item to be crimea.img with not_on_device, got %s (%s)", report.Items[1].FileName, report.Items[1].Status)
	}

	if report.Items[2].Status != StatusUpToDate || report.Items[2].FileName != "spb.img" {
		t.Errorf("expected third item to be spb.img with up_to_date, got %s (%s)", report.Items[2].FileName, report.Items[2].Status)
	}

	if report.Items[3].Status != StatusOnlyOnDevice || report.Items[3].FileName != "alps.img" {
		t.Errorf("expected fourth item to be alps.img with only_on_device, got %s (%s)", report.Items[3].FileName, report.Items[3].Status)
	}
}

func TestCompareMaps_GmapsuppMatching(t *testing.T) {
	now := time.Now()

	localMaps := []LocalMapItem{
		{
			Name:     "russia-central.img",
			Path:     "/local/russia-central.img",
			Size:     100 * 1024 * 1024,
			ModTime:  now,
			RegionID: "russia-central",
			Title:    "Central District",
		},
	}

	deviceMaps := []DeviceMap{
		{
			FileName:      "gmapsupp.img",
			FilePath:      "/device/GARMIN/gmapsupp.img",
			SizeBytes:     100 * 1024 * 1024,
			ModTime:       now,
			InternalName:  "Central District",
			SizeFormatted: "100.0 MB",
		},
	}

	items := CompareMaps(deviceMaps, localMaps, 1024*1024*1024)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != StatusUpToDate {
		t.Errorf("expected gmapsupp.img to match Central District and be up_to_date, got %s", items[0].Status)
	}
}

func TestCompareMaps_CapacityCheck(t *testing.T) {
	now := time.Now()

	localMaps := []LocalMapItem{
		{
			Name:    "huge.img",
			Path:    "/local/huge.img",
			Size:    500 * 1024 * 1024, // 500 MB
			ModTime: now,
		},
	}

	// Case 1: Device has 510 MB free (needed: 500 MB + 20 MB safety = 520 MB) -> Not enough
	items1 := CompareMaps([]DeviceMap{}, localMaps, 510*1024*1024)
	if items1[0].HasEnoughSpace {
		t.Errorf("expected HasEnoughSpace to be false when free < size + 20MB")
	}

	// Case 2: Device has 600 MB free -> Has enough
	items2 := CompareMaps([]DeviceMap{}, localMaps, 600*1024*1024)
	if !items2[0].HasEnoughSpace {
		t.Errorf("expected HasEnoughSpace to be true when free >= size + 20MB")
	}
}
