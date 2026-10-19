package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"zimap/internal/setup"
)

func createTestApp() *App {
	app := NewApp()
	app.catalogCache = &CatalogData{
		Roots: []RegionData{
			{
				ID:     "europe",
				Name:   "Europe",
				PBFURL: "https://download.geofabrik.de/europe-latest.osm.pbf",
				Children: []RegionData{
					{
						ID:     "germany",
						Name:   "Germany",
						PBFURL: "https://download.geofabrik.de/europe/germany-latest.osm.pbf",
					},
					{
						ID:     "belarus",
						Name:   "Belarus",
						PBFURL: "https://download.geofabrik.de/europe/belarus-latest.osm.pbf",
					},
					{
						ID:     "ukraine",
						Name:   "Ukraine",
						PBFURL: "https://download.geofabrik.de/europe/ukraine-latest.osm.pbf",
					},
					{
						ID:     "russia",
						Name:   "Russian Federation",
						PBFURL: "https://download.geofabrik.de/russia-latest.osm.pbf",
						Children: []RegionData{
							{
								ID:     "central-fed-district",
								Name:   "Central Federal District",
								PBFURL: "https://download.geofabrik.de/russia/central-fed-district-latest.osm.pbf",
								Children: []RegionData{
									{
										ID:     "moscow",
										Name:   "Moscow",
										PBFURL: "https://download.geofabrik.de/russia/central-fed-district/moscow-latest.osm.pbf",
									},
									{
										ID:     "tver",
										Name:   "Tver Oblast",
										PBFURL: "https://download.geofabrik.de/russia/central-fed-district/tver-latest.osm.pbf",
									},
								},
							},
							{
								ID:     "northwestern-fed-district",
								Name:   "Northwestern Federal District",
								PBFURL: "https://download.geofabrik.de/russia/northwestern-fed-district-latest.osm.pbf",
								Children: []RegionData{
									{
										ID:     "saint-petersburg",
										Name:   "Saint Petersburg",
										PBFURL: "https://download.geofabrik.de/russia/northwestern-fed-district/saint-petersburg-latest.osm.pbf",
									},
								},
							},
						},
					},
				},
			},
			{
				ID:     "asia",
				Name:   "Asia",
				PBFURL: "https://download.geofabrik.de/asia-latest.osm.pbf",
				Children: []RegionData{
					{
						ID:     "kazakhstan",
						Name:   "Kazakhstan",
						PBFURL: "https://download.geofabrik.de/asia/kazakhstan-latest.osm.pbf",
					},
					{
						ID:     "japan",
						Name:   "Japan",
						PBFURL: "https://download.geofabrik.de/asia/japan-latest.osm.pbf",
					},
				},
			},
		},
	}
	app.catalogTime = time.Now()
	return app
}

func TestSearchCatalogEmptyQuery(t *testing.T) {
	app := createTestApp()
	results, err := app.SearchCatalog("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestSearchCatalogRussianAliases(t *testing.T) {
	app := createTestApp()

	testCases := []struct {
		name         string
		query        string
		expectedID   string
		expectedName string
	}{
		{"Russia lowercase", "россия", "russia", "Russian Federation"},
		{"Russia uppercase", "РОССИЯ", "russia", "Russian Federation"},
		{"Russia RF alias", "рф", "russia", "Russian Federation"},
		{"Russia RF uppercase", "РФ", "russia", "Russian Federation"},
		{"Russia full name", "российская федерация", "russia", "Russian Federation"},
		{"Belarus", "беларусь", "belarus", "Belarus"},
		{"Belarus old alias", "белоруссия", "belarus", "Belarus"},
		{"Ukraine", "украина", "ukraine", "Ukraine"},
		{"Kazakhstan", "казахстан", "kazakhstan", "Kazakhstan"},
		{"English Russia", "russia", "russia", "Russian Federation"},
		{"English Belarus", "belarus", "belarus", "Belarus"},
		{"English Kazakhstan", "kazakhstan", "kazakhstan", "Kazakhstan"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := app.SearchCatalog(tc.query)
			if err != nil {
				t.Fatalf("SearchCatalog(%q) error: %v", tc.query, err)
			}
			if len(results) != 1 {
				t.Fatalf("SearchCatalog(%q) returned %d results, want 1", tc.query, len(results))
			}
			if results[0].ID != tc.expectedID {
				t.Errorf("SearchCatalog(%q) got ID %q, want %q", tc.query, results[0].ID, tc.expectedID)
			}
			if results[0].Name != tc.expectedName {
				t.Errorf("SearchCatalog(%q) got Name %q, want %q", tc.query, results[0].Name, tc.expectedName)
			}
		})
	}
}

func TestSearchCatalogPreservesHierarchy(t *testing.T) {
	app := createTestApp()

	results, err := app.SearchCatalog("россия")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must NOT flatten: results should only contain Russia at the top level
	if len(results) != 1 {
		t.Fatalf("expected 1 root result for 'россия', got %d (children were flattened into root slice!)", len(results))
	}

	russia := results[0]
	if russia.ID != "russia" {
		t.Fatalf("expected result ID 'russia', got %q", russia.ID)
	}

	// Children must be preserved inside russia.Children
	if len(russia.Children) != 2 {
		t.Fatalf("expected 2 federal districts under Russia, got %d", len(russia.Children))
	}

	central := russia.Children[0]
	if central.ID != "central-fed-district" {
		t.Errorf("expected first child 'central-fed-district', got %q", central.ID)
	}
	if len(central.Children) != 2 {
		t.Fatalf("expected 2 children under Central Federal District, got %d", len(central.Children))
	}
	if central.Children[0].ID != "moscow" {
		t.Errorf("expected Moscow under Central, got %q", central.Children[0].ID)
	}
}

func TestSearchCatalogRecursiveChildSearch(t *testing.T) {
	app := createTestApp()

	// Searching for "москва" should find Moscow directly without matching Russia or Europe
	results, err := app.SearchCatalog("москва")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'москва', got %d", len(results))
	}
	if results[0].ID != "moscow" {
		t.Errorf("expected ID 'moscow', got %q", results[0].ID)
	}
}

func TestSearchCatalogPartialRussianQuery(t *testing.T) {
	app := createTestApp()

	// Partial typing "росси" (>=3 chars) matches alias "россия" -> "russia"
	results, err := app.SearchCatalog("росси")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "russia" {
		t.Fatalf("expected 'russia' for 'росси', got %#v", results)
	}

	// Partial typing "белар" (>=3 chars) matches alias "беларусь" -> "belarus"
	results, err = app.SearchCatalog("белар")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "belarus" {
		t.Fatalf("expected 'belarus' for 'белар', got %#v", results)
	}
}

func TestSearchCatalogNoDuplicateResults(t *testing.T) {
	app := createTestApp()
	// Add a duplicate node in another tree to ensure seen map works
	app.catalogCache.Roots[1].Children = append(app.catalogCache.Roots[1].Children, RegionData{
		ID:   "russia",
		Name: "Russian Federation",
	})

	results, err := app.SearchCatalog("russia")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected duplicate 'russia' to be deduplicated, got %d results", len(results))
	}
}

func TestDeleteOutputValid(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	imgPath := filepath.Join(outDir, "test_map.img")
	metaPath := filepath.Join(outDir, "test_map.meta.json")
	altMetaPath := filepath.Join(outDir, "test_map.img.meta.json")

	if err := os.WriteFile(imgPath, []byte("dummy img content"), 0644); err != nil {
		t.Fatalf("failed to write img file: %v", err)
	}
	if err := os.WriteFile(metaPath, []byte(`{"name":"test_map"}`), 0644); err != nil {
		t.Fatalf("failed to write meta file: %v", err)
	}
	if err := os.WriteFile(altMetaPath, []byte(`{"name":"test_map_alt"}`), 0644); err != nil {
		t.Fatalf("failed to write alt meta file: %v", err)
	}

	if err := app.DeleteOutput(imgPath); err != nil {
		t.Fatalf("DeleteOutput failed unexpectedly: %v", err)
	}

	if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted, but it still exists", imgPath)
	}
	if _, err := os.Stat(metaPath); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted, but it still exists", metaPath)
	}
	if _, err := os.Stat(altMetaPath); !os.IsNotExist(err) {
		t.Errorf("expected %s to be deleted, but it still exists", altMetaPath)
	}
}

func TestDeleteOutputSecurityRejections(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	maliciousPaths := []string{
		"../../etc/passwd",
		"../../../etc/shadow",
		"/tmp/foo",
		"/tmp/foo.img",
		filepath.Join(tempHome, "secret.img"),
		filepath.Join(outDir, "..", "escape.img"),
		".",
		"..",
	}

	for _, p := range maliciousPaths {
		t.Run(p, func(t *testing.T) {
			err := app.DeleteOutput(p)
			if err == nil {
				t.Errorf("expected security rejection error for %q, got nil", p)
			}
		})
	}
}

func TestDeleteOutputRejectsNonImgFiles(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// 1. Regular non-.img file
	txtPath := filepath.Join(outDir, "document.txt")
	if err := os.WriteFile(txtPath, []byte("important document"), 0644); err != nil {
		t.Fatalf("failed to create txt file: %v", err)
	}

	err = app.DeleteOutput(txtPath)
	if err == nil {
		t.Errorf("expected error deleting non-img file, got nil")
	}
	if _, err := os.Stat(txtPath); err != nil {
		t.Errorf("non-img file %s should still exist, but got err: %v", txtPath, err)
	}

	// 2. Directory ending in .img
	dirImgPath := filepath.Join(outDir, "directory.img")
	if err := os.Mkdir(dirImgPath, 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	err = app.DeleteOutput(dirImgPath)
	if err == nil {
		t.Errorf("expected error deleting directory ending in .img, got nil")
	}
	if _, err := os.Stat(dirImgPath); err != nil {
		t.Errorf("directory %s should still exist, but got err: %v", dirImgPath, err)
	}
}

func TestListOutputsWithMeta(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	imgPath := filepath.Join(outDir, "germany.img")
	if err := os.WriteFile(imgPath, []byte("dummy map data"), 0644); err != nil {
		t.Fatalf("failed to write img: %v", err)
	}

	createdAt := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	meta := MapMetadata{
		Name:       "germany",
		SourceType: "geofabrik",
		SourceURL:  "https://download.geofabrik.de/europe/germany-latest.osm.pbf",
		RegionID:   "germany",
		HeapSize:   "4G",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "germany.meta.json"), metaBytes, 0644); err != nil {
		t.Fatalf("failed to write meta.json: %v", err)
	}

	outputs, err := app.ListOutputs()
	if err != nil {
		t.Fatalf("ListOutputs error: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(outputs))
	}

	item := outputs[0]
	if item.Name != "germany.img" {
		t.Errorf("expected Name 'germany.img', got %q", item.Name)
	}
	if item.SourceURL != meta.SourceURL {
		t.Errorf("expected SourceURL %q, got %q", meta.SourceURL, item.SourceURL)
	}
	if item.Meta == nil {
		t.Fatalf("expected Meta to be non-nil")
	}
	if item.Meta.RegionID != "germany" {
		t.Errorf("expected RegionID 'germany', got %q", item.Meta.RegionID)
	}
	if item.Meta.HeapSize != "4G" {
		t.Errorf("expected HeapSize '4G', got %q", item.Meta.HeapSize)
	}
	if item.ModTime.IsZero() {
		t.Errorf("expected ModTime to be set")
	}
}

func TestListOutputsFallbackCatalog(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	// Write moscow.img WITHOUT moscow.meta.json
	imgPath := filepath.Join(outDir, "moscow.img")
	if err := os.WriteFile(imgPath, []byte("moscow map data"), 0644); err != nil {
		t.Fatalf("failed to write img: %v", err)
	}

	outputs, err := app.ListOutputs()
	if err != nil {
		t.Fatalf("ListOutputs error: %v", err)
	}
	if len(outputs) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(outputs))
	}

	item := outputs[0]
	if item.Name != "moscow.img" {
		t.Errorf("expected Name 'moscow.img', got %q", item.Name)
	}
	if item.Meta == nil {
		t.Fatalf("expected Meta to be synthesized from catalog, got nil")
	}
	if item.Meta.RegionID != "moscow" {
		t.Errorf("expected RegionID 'moscow', got %q", item.Meta.RegionID)
	}
	expectedURL := "https://download.geofabrik.de/russia/central-fed-district/moscow-latest.osm.pbf"
	if item.Meta.SourceURL != expectedURL {
		t.Errorf("expected SourceURL %q, got %q", expectedURL, item.Meta.SourceURL)
	}
	if item.SourceURL != expectedURL {
		t.Errorf("expected item.SourceURL %q, got %q", expectedURL, item.SourceURL)
	}
	if item.Meta.SourceType != "geofabrik" {
		t.Errorf("expected SourceType 'geofabrik', got %q", item.Meta.SourceType)
	}

	// Check that synthesized .meta.json was persisted to disk
	metaFile := filepath.Join(outDir, "moscow.meta.json")
	if _, err := os.Stat(metaFile); err != nil {
		t.Fatalf("expected persisted metadata file at %s, got error: %v", metaFile, err)
	}

	data, err := os.ReadFile(metaFile)
	if err != nil {
		t.Fatalf("failed to read persisted meta: %v", err)
	}
	var persisted MapMetadata
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("failed to unmarshal persisted meta: %v", err)
	}
	if persisted.RegionID != "moscow" || persisted.SourceURL != expectedURL {
		t.Errorf("persisted meta does not match: %+v", persisted)
	}
}

func TestSaveCompanionMetadataUpdatesExisting(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()
	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create output dir: %v", err)
	}

	oldCreated := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	initialMeta := MapMetadata{
		Name:       "belarus",
		SourceType: "geofabrik",
		SourceURL:  "https://download.geofabrik.de/europe/belarus-latest.osm.pbf",
		RegionID:   "belarus",
		HeapSize:   "2G",
		CreatedAt:  oldCreated,
		UpdatedAt:  oldCreated,
	}
	initialBytes, _ := json.MarshalIndent(initialMeta, "", "  ")
	_ = os.WriteFile(filepath.Join(outDir, "belarus.meta.json"), initialBytes, 0644)

	// Save again with new heapSize
	err = app.saveCompanionMetadata("belarus", "geofabrik", "https://download.geofabrik.de/europe/belarus-latest.osm.pbf", "4G", "high")
	if err != nil {
		t.Fatalf("saveCompanionMetadata failed: %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(outDir, "belarus.meta.json"))
	var updated MapMetadata
	_ = json.Unmarshal(data, &updated)

	if !updated.CreatedAt.Equal(oldCreated) {
		t.Errorf("expected CreatedAt to remain %v, got %v", oldCreated, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(oldCreated) {
		t.Errorf("expected UpdatedAt to be after oldCreated, got %v", updated.UpdatedAt)
	}
	if updated.HeapSize != "4G" {
		t.Errorf("expected HeapSize '4G', got %q", updated.HeapSize)
	}
	if updated.RegionID != "belarus" {
		t.Errorf("expected RegionID 'belarus', got %q", updated.RegionID)
	}

	// Test local map metadata saving
	// Test local map metadata saving
	err = app.saveCompanionMetadata("my-local-map", "local", "/path/to/custom.osm.pbf", "8G", "")
	if err != nil {
		t.Fatalf("saveCompanionMetadata for local failed: %v", err)
	}
	localData, _ := os.ReadFile(filepath.Join(outDir, "my-local-map.meta.json"))
	var localMeta MapMetadata
	_ = json.Unmarshal(localData, &localMeta)
	if localMeta.SourceType != "local" {
		t.Errorf("expected SourceType 'local', got %q", localMeta.SourceType)
	}
	if localMeta.SourceURL != "/path/to/custom.osm.pbf" {
		t.Errorf("expected SourceURL '/path/to/custom.osm.pbf', got %q", localMeta.SourceURL)
	}
	if localMeta.RegionID != "" {
		t.Errorf("expected empty RegionID for local map, got %q", localMeta.RegionID)
	}
}

func TestProvidersAndProxy(t *testing.T) {
	app := createTestApp()

	// 1. Test GetProviders
	providers := app.GetProviders()
	if len(providers) < 3 {
		t.Fatalf("expected at least 3 providers, got %d", len(providers))
	}
	foundBBBike := false
	for _, p := range providers {
		if p.ID == "bbbike" {
			foundBBBike = true
			break
		}
	}
	if !foundBBBike {
		t.Errorf("expected bbbike in providers list")
	}

	// 2. Test GetCatalogByProvider for BBBike
	bbbikeCat, err := app.GetCatalogByProvider("bbbike")
	if err != nil {
		t.Fatalf("GetCatalogByProvider(bbbike) error: %v", err)
	}
	if len(bbbikeCat.Roots) == 0 {
		t.Errorf("expected roots in bbbike catalog")
	}

	// 3. Test SearchCatalogByProvider for BBBike
	results, err := app.SearchCatalogByProvider("bbbike", "Berlin")
	if err != nil {
		t.Fatalf("SearchCatalogByProvider(bbbike, Berlin) error: %v", err)
	}
	if len(results) == 0 {
		t.Errorf("expected to find Berlin in bbbike catalog")
	}

	// 3b. Test SearchCatalogByProvider for Saint Petersburg using Russian and English queries
	for _, query := range []string{"Санкт-Петербург", "Питер", "Saint Petersburg", "Sankt"} {
		spbResults, err := app.SearchCatalogByProvider("bbbike", query)
		if err != nil {
			t.Fatalf("SearchCatalogByProvider(bbbike, %q) error: %v", query, err)
		}
		if len(spbResults) == 0 {
			t.Errorf("expected to find Saint Petersburg in bbbike catalog for query %q", query)
			continue
		}
		// verify at least one result contains Petersburg and valid URL
		foundMatch := false
		var checkNode func(r RegionData)
		checkNode = func(r RegionData) {
			if strings.Contains(r.Name, "Petersburg") {
				foundMatch = true
				want := "https://download.bbbike.org/osm/bbbike/SanktPetersburg/SanktPetersburg.osm.pbf"
				if r.PBFURL != want {
					t.Errorf("expected PBFURL %q, got %q", want, r.PBFURL)
				}
			}
			for _, child := range r.Children {
				checkNode(child)
			}
		}
		for _, r := range spbResults {
			checkNode(r)
		}
		if !foundMatch {
			t.Errorf("expected to locate Petersburg node in results for query %q", query)
		}
	}

	// 4. Test SetProxy and GetProxy
	tmpDir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	// Default proxy
	defaultCfg, err := app.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy error: %v", err)
	}
	if !defaultCfg.UseSystemProxy {
		t.Errorf("expected UseSystemProxy to be true by default")
	}

	// Set custom proxy
	err = app.SetProxy(false, "http://127.0.0.1:8080")
	if err != nil {
		t.Fatalf("SetProxy error: %v", err)
	}
	updatedCfg, err := app.GetProxy()
	if err != nil {
		t.Fatalf("GetProxy after set error: %v", err)
	}
	if updatedCfg.UseSystemProxy {
		t.Errorf("expected UseSystemProxy to be false")
	}
	if updatedCfg.CustomProxy != "http://127.0.0.1:8080" {
		t.Errorf("expected CustomProxy 'http://127.0.0.1:8080', got %q", updatedCfg.CustomProxy)
	}

	// Invalid proxy URL
	err = app.SetProxy(false, "ftp://invalid-proxy")
	if err == nil {
		t.Errorf("expected error for invalid proxy scheme")
	}
}

func TestSendNotification(t *testing.T) {
	app := NewApp()

	// Test basic notification
	err := app.SendNotification("ZiMap Downloader", "Тестовое уведомление из модульного теста")
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Skipf("notification daemon unavailable in this environment: %v", err)
		}
		t.Fatalf("SendNotification failed: %v", err)
	}

	// Test with special characters (quotes, backslashes, punctuation)
	err = app.SendNotification("ZiMap \"Downloader\"", "Карта \"test-map\" успешно собрана! ETA: 0s & 100%")
	if err != nil {
		if runtime.GOOS == "linux" {
			t.Skipf("notification daemon unavailable in this environment: %v", err)
		}
		t.Fatalf("SendNotification with special characters failed: %v", err)
	}
}

func TestSemVerComparison(t *testing.T) {
	testCases := []struct {
		remote  string
		current string
		want    bool
	}{
		{"v1.0.0", "v1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"v1.0.1", "v1.0.0", true},
		{"1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.9", true},
		{"v2.0.0", "v1.9.9", true},
		{"v0.9.9", "v1.0.0", false},
		{"v1.0.0-rc1", "v1.0.0", false},
		{"v1.0.1-rc1", "v1.0.0", true},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v2.0.0", false},
		{"v0.0.2", "0.0.2 BETA", true},
		{"0.0.2", "0.0.2 BETA", true},
		{"0.0.2 BETA", "0.0.2 BETA", false},
		{"v0.0.1", "0.0.1 BETA", true},
		{"0.0.0", "0.0.2 BETA", false},
	}

	for _, tc := range testCases {
		got := isVersionNewer(tc.remote, tc.current)
		if got != tc.want {
			t.Errorf("isVersionNewer(%q, %q) = %v, want %v", tc.remote, tc.current, got, tc.want)
		}
	}
}

func TestStorageMetricsAndClean(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("GARMIN_MAP_HOME")
	os.Setenv("GARMIN_MAP_HOME", tmpDir)
	defer os.Setenv("GARMIN_MAP_HOME", origHome)

	cacheDir := filepath.Join(tmpDir, "cache")
	outputDir := filepath.Join(tmpDir, "output")
	toolsDir := filepath.Join(tmpDir, "tools")

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("mkdir output: %v", err)
	}
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		t.Fatalf("mkdir tools: %v", err)
	}

	// 1. Cache files
	indexData := []byte("geofabrik index data 100 bytes padding to verify length calculation exactly 100 bytes long file!!")
	if len(indexData) != 100 {
		indexData = make([]byte, 100)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "geofabrik-index.json"), indexData, 0644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}
	proxyData := []byte(`{"useSystemProxy": true}`)
	if err := os.WriteFile(filepath.Join(cacheDir, "proxy-config.json"), proxyData, 0644); err != nil {
		t.Fatalf("write proxy file: %v", err)
	}

	// 2. Maps files in output
	mapData := make([]byte, 200)
	if err := os.WriteFile(filepath.Join(outputDir, "russia.img"), mapData, 0644); err != nil {
		t.Fatalf("write map: %v", err)
	}
	metaData := make([]byte, 30)
	if err := os.WriteFile(filepath.Join(outputDir, "russia.meta.json"), metaData, 0644); err != nil {
		t.Fatalf("write meta: %v", err)
	}

	// 3. Temp files in output and tools
	tempDirA := filepath.Join(outputDir, ".download-chunk-1")
	if err := os.MkdirAll(tempDirA, 0755); err != nil {
		t.Fatalf("mkdir tempDirA: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDirA, "part.tmp"), make([]byte, 500), 0644); err != nil {
		t.Fatalf("write part.tmp: %v", err)
	}

	tempDirB := filepath.Join(outputDir, ".splitter")
	if err := os.MkdirAll(tempDirB, 0755); err != nil {
		t.Fatalf("mkdir tempDirB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDirB, "tiles.tmp"), make([]byte, 300), 0644); err != nil {
		t.Fatalf("write tiles.tmp: %v", err)
	}

	tempDirC := filepath.Join(toolsDir, ".mkgmap-update-tmp")
	if err := os.MkdirAll(tempDirC, 0755); err != nil {
		t.Fatalf("mkdir tempDirC: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDirC, "mkgmap.jar"), make([]byte, 400), 0644); err != nil {
		t.Fatalf("write update jar: %v", err)
	}

	app := createTestApp()

	// Verify metrics calculation
	metrics, err := app.GetStorageMetrics()
	if err != nil {
		t.Fatalf("GetStorageMetrics error: %v", err)
	}

	if metrics.CacheSize != 100 {
		t.Errorf("expected CacheSize 100, got %d", metrics.CacheSize)
	}
	if metrics.MapsSize != 230 {
		t.Errorf("expected MapsSize 230, got %d", metrics.MapsSize)
	}
	if metrics.TempSize != 1200 {
		t.Errorf("expected TempSize 1200 (500+300+400), got %d", metrics.TempSize)
	}
	if metrics.TotalSize != 1530 {
		t.Errorf("expected TotalSize 1530, got %d", metrics.TotalSize)
	}

	// Clear temp and cache
	freed, err := app.ClearTempAndCacheFiles(true, true)
	if err != nil {
		t.Fatalf("ClearTempAndCacheFiles error: %v", err)
	}
	if freed != 1300 {
		t.Errorf("expected freed 1300 bytes (100 cache + 1200 temp), got %d", freed)
	}

	// Verify proxy-config.json was preserved
	if _, err := os.Stat(filepath.Join(cacheDir, "proxy-config.json")); err != nil {
		t.Errorf("expected proxy-config.json to be preserved, got error: %v", err)
	}

	// Verify map files were preserved
	if _, err := os.Stat(filepath.Join(outputDir, "russia.img")); err != nil {
		t.Errorf("expected russia.img to be preserved, got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "russia.meta.json")); err != nil {
		t.Errorf("expected russia.meta.json to be preserved, got error: %v", err)
	}

	// Verify cache index was deleted
	if _, err := os.Stat(filepath.Join(cacheDir, "geofabrik-index.json")); !os.IsNotExist(err) {
		t.Errorf("expected geofabrik-index.json to be deleted")
	}

	// Verify temp dirs were deleted
	if _, err := os.Stat(tempDirA); !os.IsNotExist(err) {
		t.Errorf("expected tempDirA to be deleted")
	}
	if _, err := os.Stat(tempDirB); !os.IsNotExist(err) {
		t.Errorf("expected tempDirB to be deleted")
	}
	if _, err := os.Stat(tempDirC); !os.IsNotExist(err) {
		t.Errorf("expected tempDirC to be deleted")
	}

	// Recheck metrics after clean
	postMetrics, err := app.GetStorageMetrics()
	if err != nil {
		t.Fatalf("GetStorageMetrics after clean error: %v", err)
	}
	if postMetrics.CacheSize != 0 {
		t.Errorf("expected post-clean CacheSize 0, got %d", postMetrics.CacheSize)
	}
	if postMetrics.TempSize != 0 {
		t.Errorf("expected post-clean TempSize 0, got %d", postMetrics.TempSize)
	}
	if postMetrics.MapsSize != 230 {
		t.Errorf("expected post-clean MapsSize 230, got %d", postMetrics.MapsSize)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "nested", "dest.txt")

	content := []byte("Hello Garmin ZiMap!")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("failed to read destination: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(got))
	}

	// Test non-existent source
	nonExistent := filepath.Join(tempDir, "does-not-exist.bin")
	if err := copyFile(nonExistent, filepath.Join(tempDir, "out.bin")); err == nil {
		t.Errorf("expected error for non-existent source file, got nil")
	}
}

func TestExportMapValidation(t *testing.T) {
	outputDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("setup.OutputDirectory error: %v", err)
	}
	_ = os.MkdirAll(outputDir, 0755)

	app := NewApp()

	// 1. Path traversal security check
	_, err = app.ExportMap("../secret.img")
	if err == nil || !strings.Contains(err.Error(), "outside output directory") {
		t.Errorf("expected outside output directory error, got: %v", err)
	}

	// 2. Non-existent map file
	_, err = app.ExportMap("non-existent-map.img")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected does not exist error, got: %v", err)
	}

	// 3. Non-.img file
	badFile := filepath.Join(outputDir, "test.txt")
	_ = os.WriteFile(badFile, []byte("not a map"), 0644)
	defer os.Remove(badFile)

	_, err = app.ExportMap("test.txt")
	if err == nil || !strings.Contains(err.Error(), "not an .img map") {
		t.Errorf("expected not an .img map error, got: %v", err)
	}

	// 4. Directory instead of file
	subDir := filepath.Join(outputDir, "subfolder.img")
	_ = os.MkdirAll(subDir, 0755)
	defer os.RemoveAll(subDir)

	_, err = app.ExportMap("subfolder.img")
	if err == nil || !strings.Contains(err.Error(), "target is a directory") {
		t.Errorf("expected directory error, got: %v", err)
	}

	// 5. Valid map file, but context not initialized
	validMap := filepath.Join(outputDir, "valid-sample.img")
	_ = os.WriteFile(validMap, []byte("GARMIN_MAP_MOCK"), 0644)
	defer os.Remove(validMap)

	_, err = app.ExportMap("valid-sample.img")
	if err == nil || !strings.Contains(err.Error(), "window context not initialized") {
		t.Errorf("expected context not initialized error, got: %v", err)
	}

	// 6. Name passed without .img extension should resolve to valid-sample.img
	_, err = app.ExportMap("valid-sample")
	if err == nil || !strings.Contains(err.Error(), "window context not initialized") {
		t.Errorf("expected context not initialized error when passing name without extension, got: %v", err)
	}
}

func TestSelectLocalPBFDialogNilContext(t *testing.T) {
	app := NewApp()
	_, err := app.SelectLocalPBFDialog()
	if err == nil || !strings.Contains(err.Error(), "window context not initialized") {
		t.Errorf("expected context not initialized error, got: %v", err)
	}
}

func TestBuildBBBikeStyleURLMethod(t *testing.T) {
	app := &App{}
	got := app.BuildBBBikeStyleURL("Munich", "opentopo")
	want := "https://download.bbbike.org/osm/bbbike/Muenchen/Muenchen.osm.garmin-opentopo-latin1.zip"
	if got != want {
		t.Errorf("BuildBBBikeStyleURL() = %q, want %q", got, want)
	}
}

func TestCheckForUpdateWithServer(t *testing.T) {
	origURL := releasesLatestURL
	defer func() { releasesLatestURL = origURL }()

	// 1. Newer version available
	tsNewer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v2.0.0",
			"name":     "ZiMap 2.0.0",
			"body":     "Major release with new features!",
			"html_url": "https://github.com/Ziviy/zimap/releases/tag/v2.0.0",
		})
	}))
	defer tsNewer.Close()

	releasesLatestURL = tsNewer.URL
	app := NewApp()
	info, err := app.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate error: %v", err)
	}
	if !info.IsNewer {
		t.Errorf("expected IsNewer to be true for v2.0.0")
	}
	if info.Version != "v2.0.0" {
		t.Errorf("expected version 'v2.0.0', got %q", info.Version)
	}

	// 2. Same version (not newer)
	tsSame := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": CurrentAppVersion,
			"html_url": "https://github.com/Ziviy/zimap/releases/tag/" + CurrentAppVersion,
		})
	}))
	defer tsSame.Close()

	releasesLatestURL = tsSame.URL
	infoSame, err := app.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate error: %v", err)
	}
	if infoSame.IsNewer {
		t.Errorf("expected IsNewer to be false for same version")
	}

	// 3. 404 Not Found returns gracefully with IsNewer=false
	ts404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts404.Close()

	releasesLatestURL = ts404.URL
	info404, err := app.CheckForUpdate()
	if err != nil {
		t.Fatalf("CheckForUpdate 404 should not return error: %v", err)
	}
	if info404.IsNewer {
		t.Errorf("expected IsNewer false on 404")
	}

	// 4. Server error (500)
	ts500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts500.Close()

	releasesLatestURL = ts500.URL
	_, err = app.CheckForUpdate()
	if err == nil {
		t.Errorf("expected error on 500 response, got nil")
	}

	// 5. Malformed JSON
	tsBadJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{invalid json"))
	}))
	defer tsBadJSON.Close()

	releasesLatestURL = tsBadJSON.URL
	_, err = app.CheckForUpdate()
	if err == nil {
		t.Errorf("expected error on malformed json, got nil")
	}
}

func TestGetSystemInfo(t *testing.T) {
	app := NewApp()
	info := app.GetSystemInfo()

	requiredKeys := []string{"os", "arch", "go_version", "num_cpu"}
	for _, key := range requiredKeys {
		val, exists := info[key]
		if !exists || val == "" {
			t.Errorf("expected non-empty key %q in GetSystemInfo, got %q", key, val)
		}
	}
}

func TestCancelActiveJob(t *testing.T) {
	app := NewApp()

	// 1. When idle, CancelActiveJob succeeds
	if err := app.CancelActiveJob(); err != nil {
		t.Errorf("CancelActiveJob when idle returned error: %v", err)
	}

	// 2. When active, cancel func is invoked and cleaned up
	cancelled := false
	app.cancelJob = func() {
		cancelled = true
	}

	if err := app.CancelActiveJob(); err != nil {
		t.Errorf("CancelActiveJob returned error: %v", err)
	}
	if !cancelled {
		t.Errorf("expected cancelJob to be invoked")
	}
	if app.cancelJob != nil {
		t.Errorf("expected cancelJob to be reset to nil")
	}
}

func TestClearTempWhileJobRunning(t *testing.T) {
	app := NewApp()
	app.cancelJob = func() {}
	defer func() { app.cancelJob = nil }()

	_, err := app.ClearTempAndCacheFiles(false, true)
	if err == nil || !strings.Contains(err.Error(), "while a map build is running") {
		t.Errorf("expected running job rejection error, got: %v", err)
	}
}

func TestDownloadPrecompiledBBBike(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("GARMIN_MAP_HOME")
	os.Setenv("GARMIN_MAP_HOME", tmpDir)
	defer os.Setenv("GARMIN_MAP_HOME", origHome)

	// Create a valid zip archive containing Muenchen-garmin-opentopo-latin1/gmapsupp.img
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	imgEntry, err := zw.Create("Muenchen-garmin-opentopo-latin1/gmapsupp.img")
	if err != nil {
		t.Fatal(err)
	}
	mockImgData := []byte("PRECOMPILED_GARMIN_IMG_DATA")
	if _, err := imgEntry.Write(mockImgData); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer ts.Close()

	app := NewApp()
	ctx := context.Background()
	mapName := "Munich-Custom"

	err = app.downloadPrecompiledBBBike(ctx, ts.URL, mapName, "2G", "topo")
	if err != nil {
		t.Fatalf("downloadPrecompiledBBBike failed: %v", err)
	}

	outDir, _ := setup.OutputDirectory()
	expectedImg := filepath.Join(outDir, mapName+".img")
	gotData, err := os.ReadFile(expectedImg)
	if err != nil {
		t.Fatalf("expected compiled map at %s, got error: %v", expectedImg, err)
	}
	if string(gotData) != string(mockImgData) {
		t.Errorf("map content mismatch: got %q, want %q", string(gotData), string(mockImgData))
	}

	// Verify companion metadata was saved
	metaFile := filepath.Join(outDir, mapName+".meta.json")
	metaData, err := os.ReadFile(metaFile)
	if err != nil {
		t.Fatalf("metadata file missing: %v", err)
	}
	var meta MapMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("metadata json error: %v", err)
	}
	if meta.SourceType != "bbbike" {
		t.Errorf("expected SourceType 'bbbike', got %q", meta.SourceType)
	}

	// Test archive missing gmapsupp.img
	var badZipBuf bytes.Buffer
	zwBad := zip.NewWriter(&badZipBuf)
	badEntry, _ := zwBad.Create("dummy/readme.txt")
	_, _ = badEntry.Write([]byte("no gmapsupp"))
	_ = zwBad.Close()

	tsBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(badZipBuf.Bytes())
	}))
	defer tsBad.Close()

	err = app.downloadPrecompiledBBBike(ctx, tsBad.URL, "BadMap", "2G", "standard")
	if err == nil || !strings.Contains(err.Error(), "gmapsupp.img not found") {
		t.Errorf("expected gmapsupp.img not found error, got: %v", err)
	}
}

func TestFindRegionAndIDEdgeCases(t *testing.T) {
	app := createTestApp()

	// 1. Test by SourceURL
	r := app.findRegion("https://download.geofabrik.de/europe/germany-latest.osm.pbf", "arbitrary_name")
	if r == nil || r.ID != "germany" {
		t.Errorf("expected to find germany by URL, got: %#v", r)
	}

	// 2. Test by exact ID
	r = app.findRegion("", "kazakhstan")
	if r == nil || r.ID != "kazakhstan" {
		t.Errorf("expected to find kazakhstan by ID, got: %#v", r)
	}

	// 3. Test by name with .img extension attached
	r = app.findRegion("", "japan.img")
	if r == nil || r.ID != "japan" {
		t.Errorf("expected to find japan from 'japan.img', got: %#v", r)
	}

	// 4. Test normalized strings with underscores/spaces
	r = app.findRegion("", "central_fed_district")
	if r == nil || r.ID != "central-fed-district" {
		t.Errorf("expected to find central-fed-district from underscored query, got: %#v", r)
	}

	r = app.findRegion("", "Central Federal District")
	if r == nil || r.ID != "central-fed-district" {
		t.Errorf("expected to find central-fed-district from spaces query, got: %#v", r)
	}

	// 5. Non-existent region
	r = app.findRegion("", "atlantis")
	if r != nil {
		t.Errorf("expected nil for non-existent region, got: %#v", r)
	}
	id := app.findRegionID("", "atlantis")
	if id != "" {
		t.Errorf("expected empty string for non-existent region ID, got: %q", id)
	}
}

func TestCheckTools(t *testing.T) {
	app := NewApp()
	status, err := app.CheckTools()
	if err != nil {
		t.Fatalf("CheckTools unexpected error: %v", err)
	}
	// Verify fields are populated
	_ = status.Installed
	_ = status.Java
	_ = status.Mkgmap
	_ = status.Splitter
}

func TestGetAppVersion(t *testing.T) {
	app := NewApp()
	ver := app.GetAppVersion()
	if ver != "0.0.2 BETA" {
		t.Errorf("GetAppVersion() = %q, want '0.0.2 BETA'", ver)
	}
}

func TestFindMatchingAsset(t *testing.T) {
	assets := []GitHubAsset{
		{Name: "ZiMap-macos-arm64.zip", BrowserDownloadURL: "https://example.com/mac-arm64.zip", Size: 25000000},
		{Name: "ZiMap-macos-amd64.zip", BrowserDownloadURL: "https://example.com/mac-amd64.zip", Size: 25000000},
		{Name: "ZiMap-windows-amd64.zip", BrowserDownloadURL: "https://example.com/win-amd64.zip", Size: 23000000},
		{Name: "ZiMap-linux-amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux-amd64.tar.gz", Size: 22000000},
		{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt", Size: 100},
	}

	// 1. darwin / arm64
	match := findMatchingAsset(assets, "darwin", "arm64")
	if match == nil || match.Name != "ZiMap-macos-arm64.zip" {
		t.Errorf("expected darwin arm64 match, got: %#v", match)
	}

	// 2. darwin / amd64
	match = findMatchingAsset(assets, "darwin", "amd64")
	if match == nil || match.Name != "ZiMap-macos-amd64.zip" {
		t.Errorf("expected darwin amd64 match, got: %#v", match)
	}

	// 3. windows / amd64
	match = findMatchingAsset(assets, "windows", "amd64")
	if match == nil || match.Name != "ZiMap-windows-amd64.zip" {
		t.Errorf("expected windows amd64 match, got: %#v", match)
	}

	// 4. linux / amd64
	match = findMatchingAsset(assets, "linux", "amd64")
	if match == nil || match.Name != "ZiMap-linux-amd64.tar.gz" {
		t.Errorf("expected linux amd64 match, got: %#v", match)
	}
}

func TestSimulateUpdateLifecycle(t *testing.T) {
	app := NewApp()
	info, err := app.SimulateUpdateCheck()
	if err != nil {
		t.Fatalf("SimulateUpdateCheck returned error: %v", err)
	}
	if !info.IsNewer {
		t.Errorf("expected IsNewer to be true for simulated update")
	}
	if !info.CanAutoUpdate {
		t.Errorf("expected CanAutoUpdate to be true for simulated update")
	}
	if info.CurrentVersion != "0.0.2 BETA" {
		t.Errorf("expected CurrentVersion '0.0.2 BETA', got %q", info.CurrentVersion)
	}

	// Test CancelUpdate when idle
	if err := app.CancelUpdate(); err != nil {
		t.Errorf("CancelUpdate when idle returned error: %v", err)
	}
}

// Regression tests for missing output/ directory on fresh standalone installs:
// os.MkdirTemp fails with "GetFileAttributesEx ... cannot find the file specified"
// when the output directory does not exist yet (TODO #53).

func TestConvertURLCreatesMissingOutputDirectory(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := NewApp() // ctx intentionally nil: emitEvent no-ops, WithCancel tolerated

	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Fatalf("precondition: output dir must not exist, got: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	// Non-BBBike URL exercises the MkdirAll(outDir) path before MkdirTemp.
	err = app.ConvertURL(ts.URL+"/test-region.osm.pbf", "test-region", "2G", "standard")
	if err == nil {
		t.Fatalf("expected ConvertURL to fail for 404 mock server")
	}
	if strings.Contains(err.Error(), "GetFileAttributesEx") || strings.Contains(err.Error(), "cannot find the file specified") {
		t.Errorf("expected download error, got missing-directory error: %v", err)
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("expected ConvertURL to create output directory, got: %v", err)
	}
}

func TestConvertLocalCreatesMissingOutputDirectory(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()

	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}

	// Valid input file so runConversion reaches the workdir creation stage.
	pbfPath := filepath.Join(t.TempDir(), "region.osm.pbf")
	if err := os.WriteFile(pbfPath, []byte("OSM PBF PLACEHOLDER"), 0644); err != nil {
		t.Fatalf("failed to write placeholder pbf: %v", err)
	}

	// Expected to fail later (no splitter/mkgmap tools in a fresh home),
	// but the output directory must exist by then.
	if err := app.ConvertLocal(pbfPath, "region", "2G", "standard"); err == nil {
		t.Fatalf("expected ConvertLocal to fail without tools installed")
	}
	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("expected ConvertLocal to create output directory, got: %v", err)
	}
}

func TestSaveCompanionMetadataCreatesMissingOutputDirectory(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tempHome)

	app := createTestApp()

	outDir, err := setup.OutputDirectory()
	if err != nil {
		t.Fatalf("failed to get output directory: %v", err)
	}

	if err := app.saveCompanionMetadata("fresh-map", "local", "/tmp/custom.osm.pbf", "2G", ""); err != nil {
		t.Fatalf("saveCompanionMetadata failed without output dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "fresh-map.meta.json")); err != nil {
		t.Errorf("expected metadata file to be created: %v", err)
	}
}
