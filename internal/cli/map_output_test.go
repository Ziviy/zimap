package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFinalMapPublishesOnlyFinalImageAndPreservesPreviousMaps(t *testing.T) {
	outputDirectory := t.TempDir()
	previousMapPath := filepath.Join(outputDirectory, "previous-region.img")
	if err := os.WriteFile(previousMapPath, []byte("previous"), 0644); err != nil {
		t.Fatal(err)
	}

	finalMapPath, err := buildFinalMap(outputDirectory, "new-region", func(workDirectory string) (string, error) {
		if err := os.WriteFile(filepath.Join(workDirectory, "new-region.osm.pbf"), []byte("pbf"), 0644); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(workDirectory, "63240001.img"), []byte("tile"), 0644); err != nil {
			return "", err
		}
		convertedPath := filepath.Join(workDirectory, "gmapsupp.img")
		return convertedPath, os.WriteFile(convertedPath, []byte("final"), 0644)
	})
	if err != nil {
		t.Fatalf("buildFinalMap() error = %v", err)
	}
	if finalMapPath != filepath.Join(outputDirectory, "new-region.img") {
		t.Fatalf("final path = %q", finalMapPath)
	}
	if content, err := os.ReadFile(finalMapPath); err != nil || string(content) != "final" {
		t.Fatalf("final map content = %q, error = %v", content, err)
	}
	if content, err := os.ReadFile(previousMapPath); err != nil || string(content) != "previous" {
		t.Fatalf("previous map content = %q, error = %v", content, err)
	}

	entries, err := os.ReadDir(outputDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("output entries = %v, want only two final maps", entries)
	}
}

func TestBuildFinalMapReplacesOnlyMapWithSameName(t *testing.T) {
	outputDirectory := t.TempDir()
	targetPath := filepath.Join(outputDirectory, "region.img")
	if err := os.WriteFile(targetPath, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := buildFinalMap(outputDirectory, "region", func(workDirectory string) (string, error) {
		convertedPath := filepath.Join(workDirectory, "gmapsupp.img")
		return convertedPath, os.WriteFile(convertedPath, []byte("new"), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("target content = %q, want new", content)
	}
}

func TestBuildFinalMapCleansWorkspaceAfterFailure(t *testing.T) {
	outputDirectory := t.TempDir()
	wantErr := errors.New("conversion failed")
	_, err := buildFinalMap(outputDirectory, "region", func(workDirectory string) (string, error) {
		if writeErr := os.WriteFile(filepath.Join(workDirectory, "partial.img"), []byte("partial"), 0644); writeErr != nil {
			return "", writeErr
		}
		return "", wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("buildFinalMap() error = %v, want conversion error", err)
	}
	entries, readErr := os.ReadDir(outputDirectory)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("output entries after failure = %v, want empty", entries)
	}
}

func TestMapFileNameFromURL(t *testing.T) {
	got := mapFileNameFromURL("https://download.geofabrik.de/russia/kaliningrad-latest.osm.pbf")
	if got != "kaliningrad" {
		t.Fatalf("mapFileNameFromURL() = %q, want kaliningrad", got)
	}
	if got := mapFileNameFromURL("https://example.test/"); got != "region" {
		t.Fatalf("mapFileNameFromURL(root URL) = %q, want region", got)
	}
}
