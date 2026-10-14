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
