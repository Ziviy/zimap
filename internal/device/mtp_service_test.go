package device

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func getTestMTPBinary(t *testing.T) string {
	t.Helper()
	// Check relative tools/mtp/zimap-mtp
	candidates := []string{
		filepath.Join("..", "..", "tools", "mtp", "zimap-mtp"),
		filepath.Join("tools", "mtp", "zimap-mtp"),
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	t.Skip("zimap-mtp binary not built or not found in tools/mtp, skipping MTP integration test")
	return ""
}

func TestMTPService_BinaryResolution(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	resolved, err := svc.FindBinaryPath()
	if err != nil {
		t.Fatalf("expected binary to be found, got error: %v", err)
	}
	if resolved != bin {
		t.Errorf("expected %s, got %s", bin, resolved)
	}
	if !svc.IsAvailable() {
		t.Errorf("expected IsAvailable() to be true")
	}
}

func TestMTPService_DetectDevice_Mock(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := svc.DetectDevice(ctx)
	if err != nil {
		t.Fatalf("DetectDevice failed: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected success to be true, got error: %s", res.Error)
	}
	if res.Device == nil {
		t.Fatal("expected device to not be nil")
	}
	if !res.Device.Connected {
		t.Errorf("expected device to be connected")
	}
	if res.Device.VendorID != "091e" {
		t.Errorf("expected vendor ID 091e, got %s", res.Device.VendorID)
	}
	if res.Device.Storage.StorageID != 65537 {
		t.Errorf("expected storageId 65537, got %d", res.Device.Storage.StorageID)
	}
	if res.Device.Storage.TotalBytes == 0 {
		t.Errorf("expected totalBytes > 0")
	}
	if res.Device.Storage.FreeBytes == 0 {
		t.Errorf("expected freeBytes > 0")
	}
}

func TestMTPService_ListMaps_Mock(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	maps, err := svc.ListMaps(ctx, 65537)
	if err != nil {
		t.Fatalf("ListMaps failed: %v", err)
	}
	if len(maps) == 0 {
		t.Fatalf("expected at least 1 mock map, got 0")
	}

	foundNorthwest := false
	for _, m := range maps {
		if m.FileName == "russia-northwestern.img" {
			foundNorthwest = true
			if m.InternalName != "OSM Russia Northwestern" {
				t.Errorf("unexpected internal name: %s", m.InternalName)
			}
			if !m.IsCustom {
				t.Errorf("expected custom map to be true")
			}
			if m.ObjectID != 1042 {
				t.Errorf("expected objectId 1042, got %d", m.ObjectID)
			}
		}
	}
	if !foundNorthwest {
		t.Errorf("expected russia-northwestern.img in mock maps list")
	}
}

func TestMTPService_UploadMap_Mock(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	// Create temporary dummy .img file to upload
	tmpDir := t.TempDir()
	dummyImg := filepath.Join(tmpDir, "test-map.img")
	_ = os.WriteFile(dummyImg, make([]byte, 1024*1024), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var mu sync.Mutex
	var events []SyncProgressEvent

	err := svc.UploadMap(ctx, 65537, dummyImg, "GARMIN", func(e SyncProgressEvent) {
		mu.Lock()
		events = append(events, e)
		mu.Unlock()
	})

	if err != nil {
		t.Fatalf("UploadMap failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) == 0 {
		t.Fatalf("expected progress events to be emitted, got 0")
	}

	lastEvt := events[len(events)-1]
	if lastEvt.Percentage != 100 {
		t.Errorf("expected final percentage 100, got %d", lastEvt.Percentage)
	}
	if lastEvt.FileName != "test-map.img" {
		t.Errorf("expected filename test-map.img, got %s", lastEvt.FileName)
	}
}

func TestMTPService_UploadMap_Cancel(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	tmpDir := t.TempDir()
	dummyImg := filepath.Join(tmpDir, "cancel-map.img")
	_ = os.WriteFile(dummyImg, make([]byte, 10*1024*1024), 0644)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := svc.UploadMap(ctx, 65537, dummyImg, "GARMIN", nil)
	if err == nil {
		t.Fatalf("expected error upon cancellation, got nil")
	}
}

func TestMTPService_DeleteMap_Mock(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.DeleteMap(ctx, 65537, 1042, "russia-northwestern.img")
	if err != nil {
		t.Fatalf("DeleteMap failed: %v", err)
	}
}

func TestMTPService_DeleteSystemMap_Rejected(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := svc.DeleteMap(ctx, 65537, 0, "gmapbmap.img")
	if err == nil {
		t.Fatalf("expected error deleting system map, got nil")
	}
}

func TestMTPDetector_ScanMTPDevices(t *testing.T) {
	bin := getTestMTPBinary(t)
	svc := NewMTPService()
	svc.SetBinaryPath(bin)

	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	devs, err := ScanMTPDevices(ctx, svc)
	if err != nil {
		t.Fatalf("ScanMTPDevices failed: %v", err)
	}
	if len(devs) != 1 {
		t.Fatalf("expected 1 MTP device, got %d", len(devs))
	}

	d := devs[0]
	if !d.IsMTP {
		t.Errorf("expected IsMTP to be true")
	}
	if d.MountPath != "mtp://storage-65537" {
		t.Errorf("expected MountPath mtp://storage-65537, got %s", d.MountPath)
	}
	if d.Storage.TotalBytes == 0 || d.Storage.FreeBytes == 0 {
		t.Errorf("expected non-zero storage capacity")
	}
	if len(d.CustomMaps) == 0 {
		t.Errorf("expected custom maps to be populated")
	}
}

func TestMTP_InspectGarminMount_Transparent(t *testing.T) {
	bin := getTestMTPBinary(t)
	os.Setenv("ZIMAP_MTP_BIN", bin)
	os.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	defer os.Unsetenv("ZIMAP_MTP_BIN")
	defer os.Unsetenv("ZIMAP_MOCK_GARMIN_MTP")

	dev, err := InspectGarminMount("mtp://storage-65537")
	if err != nil {
		t.Fatalf("InspectGarminMount failed on MTP path: %v", err)
	}
	if dev == nil {
		t.Fatal("expected dev not to be nil")
	}
	if !dev.IsMTP {
		t.Errorf("expected IsMTP to be true")
	}
	if dev.MountPath != "mtp://storage-65537" {
		t.Errorf("expected mount path mtp://storage-65537, got %s", dev.MountPath)
	}
}
