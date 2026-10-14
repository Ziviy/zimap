package device

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sampleGarminDeviceXML = `<?xml version="1.0" encoding="UTF-8"?>
<Device xmlns="http://www.garmin.com/xmlschemas/GarminDevice/v2">
  <Model>
    <PartNumber>006-B3906-00</PartNumber>
    <SoftwareVersion>1577</SoftwareVersion>
    <Description>Fenix 7 Sapphire Solar</Description>
  </Model>
  <Id>3425167890</Id>
  <DisplayName>Fenix 7</DisplayName>
</Device>`

func TestParseGarminDeviceXML(t *testing.T) {
	doc, err := ParseGarminDeviceXML(strings.NewReader(sampleGarminDeviceXML))
	if err != nil {
		t.Fatalf("unexpected error parsing XML: %v", err)
	}

	if doc.Model.Description != "Fenix 7 Sapphire Solar" {
		t.Errorf("expected Description 'Fenix 7 Sapphire Solar', got '%s'", doc.Model.Description)
	}
	if doc.DisplayName != "Fenix 7" {
		t.Errorf("expected DisplayName 'Fenix 7', got '%s'", doc.DisplayName)
	}
	if doc.UnitID != "3425167890" {
		t.Errorf("expected UnitID '3425167890', got '%s'", doc.UnitID)
	}
	if doc.Model.SoftwareVersion != "1577" {
		t.Errorf("expected SoftwareVersion '1577', got '%s'", doc.Model.SoftwareVersion)
	}
}

func TestIsSystemMap(t *testing.T) {
	systemMaps := []string{
		"gmapbmap.img",
		"gmapbmap.sum",
		"GMAPBMAP.IMG",
		"gmaptz.img",
		"gmapdem.img",
		"gmapdem1.img",
		"gmapprom.img",
		"gmapprom1.img",
		"gmapgc.img",
		"gmap3d.img",
		"d6185070a.img",
		"D4713020A.IMG",
		"d1234567.img",
		"D9218040B.img",
		"DA599010A.img",
		"DA597010A.img",
		"system.bin",
		"notes.txt",
	}

	for _, name := range systemMaps {
		if !IsSystemMap(name) {
			t.Errorf("expected %s to be recognized as system map / protected", name)
		}
	}

	customMaps := []string{
		"gmapsupp.img",
		"GMAPSUPP.IMG",
		"gmapsupp1.img",
		"russia_central.img",
		"cyprus_osm.img",
		"alps_hiking_topo.img",
		"moscow.img",
		"denmark1.img",
	}

	for _, name := range customMaps {
		if IsSystemMap(name) {
			t.Errorf("expected %s to be recognized as custom map (not system)", name)
		}
	}
}

func TestParseMapInternalName(t *testing.T) {
	// Create a buffer with at least 0x49 + 30 bytes
	buf := make([]byte, 200)
	copy(buf[0x49:], "OSM Russia Central (ZiMap)")

	name, err := ParseMapInternalNameFromReader(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("unexpected error parsing internal name: %v", err)
	}
	if name != "OSM Russia Central (ZiMap)" {
		t.Errorf("expected 'OSM Russia Central (ZiMap)', got '%s'", name)
	}

	// Buffer too small
	shortBuf := make([]byte, 50)
	nameShort, err := ParseMapInternalNameFromReader(bytes.NewReader(shortBuf))
	if err != nil || nameShort != "" {
		t.Errorf("expected empty string for short buffer, got '%s', err: %v", nameShort, err)
	}
}

func TestFormatStorageBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{500, "500 B"},
		{2048, "2.0 KB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{16 * 1024 * 1024 * 1024, "16.0 GB"},
	}

	for _, tc := range tests {
		got := FormatStorageBytes(tc.bytes)
		if got != tc.expected {
			t.Errorf("FormatStorageBytes(%d) = %s, expected %s", tc.bytes, got, tc.expected)
		}
	}
}

func TestInspectGarminMountAndValidateDelete(t *testing.T) {
	tmpDir := t.TempDir()

	garminDir := filepath.Join(tmpDir, "GARMIN")
	mapDir := filepath.Join(garminDir, "Map")
	if err := os.MkdirAll(mapDir, 0755); err != nil {
		t.Fatalf("failed to create mock dirs: %v", err)
	}

	// Write GarminDevice.xml
	if err := os.WriteFile(filepath.Join(garminDir, "GarminDevice.xml"), []byte(sampleGarminDeviceXML), 0644); err != nil {
		t.Fatalf("failed to write XML: %v", err)
	}

	// Write system map
	if err := os.WriteFile(filepath.Join(garminDir, "gmapbmap.img"), []byte("basemap"), 0644); err != nil {
		t.Fatalf("failed to write basemap: %v", err)
	}

	// Write custom map 1 with header
	map1Data := make([]byte, 200)
	copy(map1Data[0x49:], "Cyprus Topo OSM")
	customPath1 := filepath.Join(garminDir, "gmapsupp.img")
	if err := os.WriteFile(customPath1, map1Data, 0644); err != nil {
		t.Fatalf("failed to write custom map 1: %v", err)
	}

	// Write custom map 2 in Map/ folder
	time.Sleep(10 * time.Millisecond) // distinct mod time
	customPath2 := filepath.Join(mapDir, "alps.img")
	if err := os.WriteFile(customPath2, []byte("alps map"), 0644); err != nil {
		t.Fatalf("failed to write custom map 2: %v", err)
	}

	// Inspect mount
	dev, err := InspectGarminMount(tmpDir)
	if err != nil {
		t.Fatalf("failed to inspect mock mount: %v", err)
	}

	if dev.ModelName != "Fenix 7 Sapphire Solar" {
		t.Errorf("expected ModelName 'Fenix 7 Sapphire Solar', got '%s'", dev.ModelName)
	}
	if dev.UnitID != "3425167890" {
		t.Errorf("expected UnitID '3425167890', got '%s'", dev.UnitID)
	}
	if dev.TotalMapsCount != 3 {
		t.Errorf("expected TotalMapsCount 3, got %d", dev.TotalMapsCount)
	}
	if len(dev.CustomMaps) != 2 {
		t.Fatalf("expected 2 custom maps, got %d", len(dev.CustomMaps))
	}

	// Verify custom map 1 details
	var map1, map2 *DeviceMap
	for i := range dev.CustomMaps {
		if dev.CustomMaps[i].FileName == "gmapsupp.img" {
			map1 = &dev.CustomMaps[i]
		}
		if dev.CustomMaps[i].FileName == "alps.img" {
			map2 = &dev.CustomMaps[i]
		}
	}

	if map1 == nil || map2 == nil {
		t.Fatalf("expected both gmapsupp.img and alps.img in CustomMaps")
	}

	if map1.InternalName != "Cyprus Topo OSM" {
		t.Errorf("expected internal name 'Cyprus Topo OSM', got '%s'", map1.InternalName)
	}
	if map1.RelativeDir != "GARMIN" {
		t.Errorf("expected RelativeDir 'GARMIN', got '%s'", map1.RelativeDir)
	}
	if map2.RelativeDir != "GARMIN/Map" {
		t.Errorf("expected RelativeDir 'GARMIN/Map', got '%s'", map2.RelativeDir)
	}

	// Test ValidateDeletePath
	// 1. Valid custom map deletion
	validTarget, err := ValidateDeletePath(tmpDir, customPath1)
	if err != nil || validTarget != customPath1 {
		t.Errorf("expected valid delete target for %s, got %s, err: %v", customPath1, validTarget, err)
	}

	// 2. Relative path deletion
	relTarget, err := ValidateDeletePath(tmpDir, filepath.Join("GARMIN", "Map", "alps.img"))
	if err != nil || relTarget != customPath2 {
		t.Errorf("expected valid relative target %s, got %s, err: %v", customPath2, relTarget, err)
	}

	// 3. System map deletion must be blocked
	_, err = ValidateDeletePath(tmpDir, filepath.Join(garminDir, "gmapbmap.img"))
	if err != ErrSystemMapProtected {
		t.Errorf("expected ErrSystemMapProtected for gmapbmap.img, got %v", err)
	}

	// 4. Path traversal attempt must be blocked
	_, err = ValidateDeletePath(tmpDir, filepath.Join(tmpDir, "..", "outside.img"))
	if err == nil {
		t.Errorf("expected error for path traversal attempt, got nil")
	}

	// 5. Test ScanDevices with ZIMAP_MOCK_GARMIN_DIR
	t.Setenv("ZIMAP_MOCK_GARMIN_DIR", tmpDir)
	devices, err := ScanDevices()
	if err != nil {
		t.Fatalf("ScanDevices error: %v", err)
	}
	if len(devices) == 0 {
		t.Fatalf("expected at least 1 device from mock dir, got 0")
	}
}

func TestCheckGarminUSB(t *testing.T) {
	// Without mock
	status := CheckGarminUSB(0)
	_ = status // verify it runs without crashing

	// With MTP mock
	t.Setenv("ZIMAP_MOCK_GARMIN_MTP", "1")
	mtpStatus := CheckGarminUSB(0)
	if !mtpStatus.GarminUSBAttached || !mtpStatus.NeedsMTPNotice {
		t.Errorf("expected MTP notice when 0 devices mounted, got %+v", mtpStatus)
	}

	mtpMountedStatus := CheckGarminUSB(1)
	if !mtpMountedStatus.GarminUSBAttached || mtpMountedStatus.NeedsMTPNotice {
		t.Errorf("expected no MTP notice when 1 device is mounted, got %+v", mtpMountedStatus)
	}
}
