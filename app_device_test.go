package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"zimap/internal/device"
)

func TestAppDeviceMethods(t *testing.T) {
	app := NewApp()
	tmpDir := t.TempDir()

	garminDir := filepath.Join(tmpDir, "GARMIN")
	if err := os.MkdirAll(garminDir, 0755); err != nil {
		t.Fatalf("failed to create GARMIN dir: %v", err)
	}

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Device xmlns="http://www.garmin.com/xmlschemas/GarminDevice/v2">
  <Model>
    <Description>Garmin Epix Pro</Description>
    <SoftwareVersion>1420</SoftwareVersion>
  </Model>
  <Id>9876543210</Id>
  <DisplayName>Epix Pro</DisplayName>
</Device>`
	if err := os.WriteFile(filepath.Join(garminDir, "GarminDevice.xml"), []byte(xmlContent), 0644); err != nil {
		t.Fatalf("failed to write XML: %v", err)
	}

	// Create a custom map with companion files
	customMapPath := filepath.Join(garminDir, "gmapsupp.img")
	customSumPath := filepath.Join(garminDir, "gmapsupp.sum")
	if err := os.WriteFile(customMapPath, []byte("custom-map-data"), 0644); err != nil {
		t.Fatalf("failed to write custom map: %v", err)
	}
	if err := os.WriteFile(customSumPath, []byte("sum"), 0644); err != nil {
		t.Fatalf("failed to write custom sum: %v", err)
	}

	// Create a system map
	systemMapPath := filepath.Join(garminDir, "gmapbmap.img")
	if err := os.WriteFile(systemMapPath, []byte("basemap"), 0644); err != nil {
		t.Fatalf("failed to write system map: %v", err)
	}

	t.Setenv("ZIMAP_MOCK_GARMIN_DIR", tmpDir)

	// Test GetConnectedGarminDevices
	devices, err := app.GetConnectedGarminDevices()
	if err != nil {
		t.Fatalf("GetConnectedGarminDevices error: %v", err)
	}
	if len(devices) == 0 {
		t.Fatalf("expected at least 1 device, got 0")
	}

	dev := devices[0]
	if dev.ModelName != "Garmin Epix Pro" {
		t.Errorf("expected ModelName 'Garmin Epix Pro', got '%s'", dev.ModelName)
	}
	if dev.UnitID != "9876543210" {
		t.Errorf("expected UnitID '9876543210', got '%s'", dev.UnitID)
	}
	if len(dev.CustomMaps) != 1 {
		t.Fatalf("expected 1 custom map, got %d", len(dev.CustomMaps))
	}
	if dev.CustomMaps[0].FileName != "gmapsupp.img" {
		t.Errorf("expected custom map 'gmapsupp.img', got '%s'", dev.CustomMaps[0].FileName)
	}

	// Test GetGarminUSBStatus
	status, err := app.GetGarminUSBStatus()
	if err != nil {
		t.Fatalf("GetGarminUSBStatus error: %v", err)
	}
	// Since 1 device is mounted, needsMTPNotice must be false
	if status.NeedsMTPNotice {
		t.Errorf("expected NeedsMTPNotice=false when device is mounted")
	}

	// Test DeleteDeviceMap on system map (must fail)
	err = app.DeleteDeviceMap(tmpDir, systemMapPath)
	if err == nil {
		t.Fatalf("expected error deleting system map, got nil")
	}

	// Test DeleteDeviceMap on custom map (must succeed)
	err = app.DeleteDeviceMap(tmpDir, customMapPath)
	if err != nil {
		t.Fatalf("failed to delete custom map: %v", err)
	}

	// Verify file was deleted and sum was also cleaned up
	if _, err := os.Stat(customMapPath); !os.IsNotExist(err) {
		t.Errorf("expected custom map to be deleted from disk")
	}
	if _, err := os.Stat(customSumPath); !os.IsNotExist(err) {
		t.Errorf("expected custom sum to be deleted from disk")
	}
	// Verify system map was preserved
	if _, err := os.Stat(systemMapPath); err != nil {
		t.Errorf("expected system map to remain on disk, got err: %v", err)
	}

	// Test RevealDevicePath with non-existent path
	if err := app.RevealDevicePath(filepath.Join(tmpDir, "nonexistent.file")); err == nil {
		t.Errorf("expected error revealing nonexistent file, got nil")
	}

	// Test RevealDevicePath with existing directory
	_ = app.RevealDevicePath(garminDir)
}

func TestAppDeviceFreshnessAndSync(t *testing.T) {
	app := NewApp()

	tmpHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpHome)

	outputDir := filepath.Join(tmpHome, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// 1. Create a local map on computer
	localMapPath := filepath.Join(outputDir, "russia-central.img")
	localContent := []byte("fresh-map-binary-data-for-central-russia")
	if err := os.WriteFile(localMapPath, localContent, 0644); err != nil {
		t.Fatalf("failed to write local map: %v", err)
	}
	freshModTime := time.Now().Truncate(2 * time.Second)
	_ = os.Chtimes(localMapPath, freshModTime, freshModTime)

	// 2. Create mock Garmin device
	tmpDevice := t.TempDir()
	garminDir := filepath.Join(tmpDevice, "GARMIN")
	_ = os.MkdirAll(garminDir, 0755)

	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<Device xmlns="http://www.garmin.com/xmlschemas/GarminDevice/v2">
  <Model><Description>Fenix 7</Description></Model>
  <Id>12345678</Id>
</Device>`
	_ = os.WriteFile(filepath.Join(garminDir, "GarminDevice.xml"), []byte(xmlContent), 0644)

	// Device has an older version of russia-central.img (3 days ago)
	deviceMapPath := filepath.Join(garminDir, "russia-central.img")
	oldContent := []byte("old-map-data")
	_ = os.WriteFile(deviceMapPath, oldContent, 0644)
	oldModTime := freshModTime.Add(-72 * time.Hour)
	_ = os.Chtimes(deviceMapPath, oldModTime, oldModTime)

	// 3. Check Freshness: should detect update_available
	report, err := app.CheckDeviceFreshness(tmpDevice)
	if err != nil {
		t.Fatalf("CheckDeviceFreshness error: %v", err)
	}
	if report.UpdateAvailableCount != 1 {
		t.Errorf("expected 1 update available, got %d", report.UpdateAvailableCount)
	}
	if len(report.Items) != 1 || report.Items[0].Status != "update_available" {
		t.Fatalf("expected item with status update_available, got %+v", report.Items)
	}

	// 4. Sync map to device
	err = app.SyncMapToDevice(tmpDevice, "russia-central.img")
	if err != nil {
		t.Fatalf("SyncMapToDevice error: %v", err)
	}

	// Wait for background goroutine to complete
	deadline := time.Now().Add(5 * time.Second)
	for app.getSyncManager().IsSyncing() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if app.getSyncManager().IsSyncing() {
		t.Fatalf("sync timed out")
	}

	// Verify device map was updated
	newDeviceData, err := os.ReadFile(deviceMapPath)
	if err != nil {
		t.Fatalf("failed to read updated device map: %v", err)
	}
	if string(newDeviceData) != string(localContent) {
		t.Errorf("expected device map data '%s', got '%s'", string(localContent), string(newDeviceData))
	}

	// 5. Re-check freshness: should now be up_to_date!
	report2, err := app.CheckDeviceFreshness(tmpDevice)
	if err != nil {
		t.Fatalf("second CheckDeviceFreshness error: %v", err)
	}
	if report2.UpdateAvailableCount != 0 {
		t.Errorf("expected 0 update available after sync, got %d", report2.UpdateAvailableCount)
	}
	if report2.UpToDateCount != 1 {
		t.Errorf("expected 1 up to date after sync, got %d", report2.UpToDateCount)
	}

	// Test CancelDeviceSync (safe even if not running)
	if err := app.CancelDeviceSync(); err != nil {
		t.Errorf("unexpected error on CancelDeviceSync: %v", err)
	}
}

func TestAppMTPDeviceOperations(t *testing.T) {
	bin := filepath.Join("tools", "mtp", "zimap-mtp")
	absBin, err := filepath.Abs(bin)
	if err != nil {
		t.Fatalf("failed to resolve zimap-mtp path: %v", err)
	}
	if _, err := os.Stat(absBin); err != nil {
		t.Skip("zimap-mtp binary not found, skipping App MTP integration test")
	}

	app := NewApp()
	t.Setenv("ZIMAP_MTP_BIN", absBin)
	t.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")

	tmpHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpHome)

	outputDir := filepath.Join(tmpHome, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Create local map
	localMap := filepath.Join(outputDir, "russia-northwestern.img")
	_ = os.WriteFile(localMap, []byte("mtp-test-data"), 0644)

	// 1. Test GetConnectedGarminDevices with MTP
	devs, err := app.GetConnectedGarminDevices()
	if err != nil {
		t.Fatalf("GetConnectedGarminDevices failed: %v", err)
	}
	if len(devs) == 0 {
		t.Fatalf("expected at least 1 MTP device, got 0")
	}

	var mtpDev *device.GarminDevice
	for i := range devs {
		if devs[i].IsMTP {
			mtpDev = &devs[i]
			break
		}
	}
	if mtpDev == nil {
		t.Fatalf("expected to find an MTP device in GetConnectedGarminDevices")
	}
	if mtpDev.MountPath != "mtp://storage-65537" {
		t.Errorf("expected MountPath mtp://storage-65537, got %s", mtpDev.MountPath)
	}

	// 2. Test CheckDeviceFreshness with MTP
	report, err := app.CheckDeviceFreshness(mtpDev.MountPath)
	if err != nil {
		t.Fatalf("CheckDeviceFreshness failed for MTP: %v", err)
	}
	if report == nil {
		t.Fatal("expected freshness report not to be nil")
	}

	// 3. Test SyncMapToDevice with MTP
	err = app.SyncMapToDevice(mtpDev.MountPath, "russia-northwestern.img")
	if err != nil {
		t.Fatalf("SyncMapToDevice failed on MTP: %v", err)
	}

	// Wait for background sync to complete
	time.Sleep(1200 * time.Millisecond)

	// 4. Test DeleteDeviceMap with MTP
	err = app.DeleteDeviceMap(mtpDev.MountPath, "russia-northwestern.img")
	if err != nil {
		t.Fatalf("DeleteDeviceMap failed on MTP: %v", err)
	}

	// System map delete rejected
	err = app.DeleteDeviceMap(mtpDev.MountPath, "gmapbmap.img")
	if err == nil {
		t.Fatalf("expected error deleting system map on MTP, got nil")
	}

	// Eject MTP (clean no-op)
	if err := app.EjectGarminDevice(mtpDev.MountPath); err != nil {
		t.Errorf("unexpected error on EjectGarminDevice for MTP: %v", err)
	}

	// Reveal path on MTP (expected graceful error)
	if err := app.RevealDevicePath(mtpDev.MountPath); err == nil {
		t.Errorf("expected error on RevealDevicePath for MTP path, got nil")
	}
}


