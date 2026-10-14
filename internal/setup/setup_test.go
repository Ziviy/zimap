package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckToolsDirectoryMissingTools(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpDir)
	// Clear PATH so system tools aren't found if checking fallback
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	err := CheckToolsDirectory()
	// Since tmpDir has no tools and PATH is empty, CheckToolsDirectory should return errors
	if err == nil {
		t.Errorf("expected error when tools are missing, got nil")
	}
}

func TestCheckToolsDirectoryWithMocks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpDir)

	toolsDir := filepath.Join(tmpDir, "tools")
	mkgmapDir := filepath.Join(toolsDir, "mkgmap")
	splitterDir := filepath.Join(toolsDir, "splitter")

	_ = os.MkdirAll(mkgmapDir, 0755)
	_ = os.MkdirAll(splitterDir, 0755)

	createJAR(t, filepath.Join(mkgmapDir, "mkgmap.jar"), "Manifest-Version: 1.0\n")
	createJAR(t, filepath.Join(splitterDir, "splitter.jar"), "Manifest-Version: 1.0\n")

	mkgmapPath, err := MkgmapBinPath()
	if err != nil {
		t.Fatalf("MkgmapBinPath error: %v", err)
	}
	if filepath.Base(mkgmapPath) != "mkgmap.jar" {
		t.Errorf("unexpected mkgmap path: %s", mkgmapPath)
	}

	splitterPath, err := SplitterBinPath()
	if err != nil {
		t.Fatalf("SplitterBinPath error: %v", err)
	}
	if filepath.Base(splitterPath) != "splitter.jar" {
		t.Errorf("unexpected splitter path: %s", splitterPath)
	}
}
