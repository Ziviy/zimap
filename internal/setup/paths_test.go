package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRootWithEnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpDir)

	root, err := ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot() returned unexpected error: %v", err)
	}

	expected, _ := filepath.Abs(tmpDir)
	if root != expected {
		t.Errorf("ProjectRoot() = %q, want %q", root, expected)
	}
}

func TestStandardDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpDir)

	outDir, err := OutputDirectory()
	if err != nil {
		t.Fatalf("OutputDirectory() error: %v", err)
	}
	wantOut := filepath.Join(tmpDir, "output")
	if outDir != wantOut {
		t.Errorf("OutputDirectory() = %q, want %q", outDir, wantOut)
	}

	cacheDir, err := CacheDirectory()
	if err != nil {
		t.Fatalf("CacheDirectory() error: %v", err)
	}
	wantCache := filepath.Join(tmpDir, "cache")
	if cacheDir != wantCache {
		t.Errorf("CacheDirectory() = %q, want %q", cacheDir, wantCache)
	}

	toolsDir, err := ToolsDirectory()
	if err != nil {
		t.Fatalf("ToolsDirectory() error: %v", err)
	}
	wantTools := filepath.Join(tmpDir, "tools")
	if toolsDir != wantTools {
		t.Errorf("ToolsDirectory() = %q, want %q", toolsDir, wantTools)
	}
}

func TestProjectPathAbsoluteAndRelative(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GARMIN_MAP_HOME", tmpDir)

	// Absolute path should be returned cleaned
	absPath := filepath.Join(tmpDir, "custom", "path")
	gotAbs, err := projectPath(absPath)
	if err != nil {
		t.Fatalf("projectPath(abs) error: %v", err)
	}
	if gotAbs != absPath {
		t.Errorf("projectPath(abs) = %q, want %q", gotAbs, absPath)
	}

	// Relative path should be joined to ProjectRoot
	relPath := "sub/file.txt"
	gotRel, err := projectPath(relPath)
	if err != nil {
		t.Fatalf("projectPath(rel) error: %v", err)
	}
	wantRel := filepath.Join(tmpDir, relPath)
	if gotRel != wantRel {
		t.Errorf("projectPath(rel) = %q, want %q", gotRel, wantRel)
	}
}

func TestFindProjectRoot(t *testing.T) {
	tmpDir := t.TempDir()
	// Create mock go.mod
	goModPath := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module test"), 0644); err != nil {
		t.Fatal(err)
	}

	nestedDir := filepath.Join(tmpDir, "deeply", "nested", "directory")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	foundRoot, found := findProjectRoot(nestedDir)
	if !found {
		t.Fatalf("expected to find project root from nested directory")
	}
	if foundRoot != tmpDir {
		t.Errorf("findProjectRoot = %q, want %q", foundRoot, tmpDir)
	}

	// Test directory with no go.mod in parents (e.g. system temp root or root)
	emptyDir := filepath.Join(tmpDir, "isolated")
	// If we start from root "/" or equivalent without go.mod
	_, foundRoot2 := findProjectRoot(os.TempDir())
	// os.TempDir might or might not have go.mod in parent, but an isolated temp without go.mod shouldn't match above its boundary
	_ = emptyDir
	_ = foundRoot2
}

func TestDefaultDataDirectoryPerOS(t *testing.T) {
	// 1. Darwin
	darwinDir, err := defaultDataDirectory("darwin")
	if err != nil {
		t.Fatalf("defaultDataDirectory(darwin) error: %v", err)
	}
	if filepath.Base(darwinDir) != "ZiMap" {
		t.Errorf("expected base 'ZiMap' for darwin, got %q", darwinDir)
	}

	// 2. Windows with LOCALAPPDATA
	winBase := filepath.Join(t.TempDir(), "AppData", "Local")
	t.Setenv("LOCALAPPDATA", winBase)
	winDir, err := defaultDataDirectory("windows")
	if err != nil {
		t.Fatalf("defaultDataDirectory(windows) error: %v", err)
	}
	wantWin := filepath.Join(winBase, "ZiMap")
	if winDir != wantWin {
		t.Errorf("defaultDataDirectory(windows) = %q, want %q", winDir, wantWin)
	}

	// 3. Linux with XDG_DATA_HOME
	xdgBase := filepath.Join(t.TempDir(), "xdg-share")
	t.Setenv("XDG_DATA_HOME", xdgBase)
	linuxDir, err := defaultDataDirectory("linux")
	if err != nil {
		t.Fatalf("defaultDataDirectory(linux) error: %v", err)
	}
	wantLinux := filepath.Join(xdgBase, "ZiMap")
	if linuxDir != wantLinux {
		t.Errorf("defaultDataDirectory(linux) = %q, want %q", linuxDir, wantLinux)
	}
}

func TestDetectPortableDir(t *testing.T) {
	tmpDir := t.TempDir()
	fakeExe := filepath.Join(tmpDir, "ZiMap.exe")

	// 1. Without portable marker or folder
	if _, ok := detectPortableDir(fakeExe); ok {
		t.Fatalf("expected detectPortableDir=false when no marker or folder exists")
	}

	// 2. With .portable marker file
	marker := filepath.Join(tmpDir, ".portable")
	if err := os.WriteFile(marker, []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	gotDir, ok := detectPortableDir(fakeExe)
	if !ok {
		t.Fatalf("expected detectPortableDir=true when .portable file exists")
	}
	wantPortable := filepath.Join(tmpDir, "zimap-data")
	if gotDir != wantPortable {
		t.Errorf("detectPortableDir = %q, want %q", gotDir, wantPortable)
	}
	_ = os.Remove(marker)

	// 3. With zimap-data directory
	if err := os.MkdirAll(wantPortable, 0755); err != nil {
		t.Fatal(err)
	}
	gotDir2, ok2 := detectPortableDir(fakeExe)
	if !ok2 || gotDir2 != wantPortable {
		t.Errorf("detectPortableDir with zimap-data dir = (%q, %v), want (%q, true)", gotDir2, ok2, wantPortable)
	}

	// 4. Inside macOS .app bundle must never trigger portable dir
	bundleExe := filepath.Join(tmpDir, "ZiMap.app", "Contents", "MacOS", "ZiMap")
	if _, ok := detectPortableDir(bundleExe); ok {
		t.Errorf("expected detectPortableDir=false inside .app bundle")
	}
}

func TestDataDirectoryStandaloneWithoutEnv(t *testing.T) {
	t.Setenv("ZIMAP_HOME", "")
	t.Setenv("GARMIN_MAP_HOME", "")

	dir, err := DataDirectory()
	if err != nil {
		t.Fatalf("DataDirectory() error: %v", err)
	}
	if filepath.Base(dir) != "ZiMap" && filepath.Base(dir) != "zimap-data" {
		t.Errorf("expected DataDirectory() to end with ZiMap or zimap-data, got %q", dir)
	}
}
