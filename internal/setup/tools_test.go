package setup

import (
	"archive/zip"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestToolsIncludeJavaUpdater(t *testing.T) {
	var javaTool *Tool
	for index := range Tools {
		if Tools[index].Name == "java" {
			javaTool = &Tools[index]
			break
		}
	}
	if javaTool == nil {
		t.Fatal("Tools does not contain Java")
	}
	if javaTool.Path != "tools/jdk" {
		t.Fatalf("Java path = %q, want tools/jdk", javaTool.Path)
	}
	if !strings.Contains(javaTool.URL, "/latest/21/") || !strings.Contains(javaTool.URL, "{os}") || !strings.Contains(javaTool.URL, "{arch}") {
		t.Fatalf("Java URL = %q, want latest Java 21 URL with platform placeholders", javaTool.URL)
	}
	wantUpdater := reflect.ValueOf(UpdateJava).Pointer()
	if reflect.ValueOf(javaTool.RunUpdate).Pointer() != wantUpdater {
		t.Fatal("Java tool does not use UpdateJava")
	}
}

func TestValidateJavaDistribution(t *testing.T) {
	rootPath := t.TempDir()
	javaPath := javaExecutablePath(rootPath, runtime.GOOS)
	if err := os.MkdirAll(filepath.Dir(javaPath), 0755); err != nil {
		t.Fatal(err)
	}
	mode := os.FileMode(0755)
	if runtime.GOOS == "windows" {
		mode = 0644
	}
	if err := os.WriteFile(javaPath, []byte("java"), mode); err != nil {
		t.Fatal(err)
	}
	if err := validateJavaDistribution(rootPath, runtime.GOOS); err != nil {
		t.Fatalf("validateJavaDistribution() error = %v", err)
	}
}

func TestValidateJavaDistributionRejectsMissingExecutable(t *testing.T) {
	if err := validateJavaDistribution(t.TempDir(), runtime.GOOS); err == nil {
		t.Fatal("validateJavaDistribution() error = nil, want missing executable error")
	}
}

func TestAdoptiumPlatformNames(t *testing.T) {
	if got := adoptiumOS("darwin"); got != "mac" {
		t.Fatalf("adoptiumOS(darwin) = %q, want mac", got)
	}
	if got := adoptiumOS("linux"); got != "linux" {
		t.Fatalf("adoptiumOS(linux) = %q, want linux", got)
	}
	if got := adoptiumArch("arm64"); got != "aarch64" {
		t.Fatalf("adoptiumArch(arm64) = %q, want aarch64", got)
	}
	if got := adoptiumArch("amd64"); got != "x64" {
		t.Fatalf("adoptiumArch(amd64) = %q, want x64", got)
	}
}

func TestReplacePathRollsBackWhenInstallationFails(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "tool.jar")
	if err := os.WriteFile(targetPath, []byte("working version"), 0644); err != nil {
		t.Fatal(err)
	}

	err := replacePath(filepath.Join(directory, "missing.jar"), targetPath)
	if err == nil {
		t.Fatal("replacePath() error = nil, want installation failure")
	}

	content, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("old tool was not restored: %v", readErr)
	}
	if string(content) != "working version" {
		t.Fatalf("restored content = %q, want working version", content)
	}
}

func TestValidateJAR(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "tool.jar")
	file, err := os.Create(jarPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	manifest, err := writer.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifest.Write([]byte("Manifest-Version: 1.0\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if err := validateJAR(jarPath); err != nil {
		t.Fatalf("validateJAR() error = %v", err)
	}
}

func TestValidateJARRejectsTruncatedFile(t *testing.T) {
	jarPath := filepath.Join(t.TempDir(), "tool.jar")
	if err := os.WriteFile(jarPath, []byte("PK\x03\x04truncated"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := validateJAR(jarPath); err == nil {
		t.Fatal("validateJAR() error = nil, want invalid archive error")
	}
}

func TestValidateMkgmapDistributionRequiresManifestDependencies(t *testing.T) {
	directory := t.TempDir()
	jarPath := filepath.Join(directory, "mkgmap.jar")
	createJAR(t, jarPath, "Manifest-Version: 1.0\nClass-Path: lib/osmpbf.jar lib/protobuf.jar\n")

	err := validateMkgmapDistribution(jarPath)
	if err == nil || !strings.Contains(err.Error(), "lib/osmpbf.jar") {
		t.Fatalf("validateMkgmapDistribution() error = %v, want missing dependency", err)
	}

	for _, dependency := range []string{"osmpbf.jar", "protobuf.jar"} {
		createJAR(t, filepath.Join(directory, "lib", dependency), "Manifest-Version: 1.0\n")
	}
	if err := validateMkgmapDistribution(jarPath); err != nil {
		t.Fatalf("validateMkgmapDistribution() error = %v", err)
	}
}

func TestProjectRootUsesEnvironmentOverride(t *testing.T) {
	expectedRoot := t.TempDir()
	t.Setenv(homeEnvironmentVariable, expectedRoot)

	root, err := ProjectRoot()
	if err != nil {
		t.Fatalf("ProjectRoot() error = %v", err)
	}
	if root != expectedRoot {
		t.Fatalf("ProjectRoot() = %q, want %q", root, expectedRoot)
	}
}

func createJAR(t *testing.T, path string, manifestContent string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	manifest, err := writer.Create("META-INF/MANIFEST.MF")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifest.Write([]byte(manifestContent)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}
