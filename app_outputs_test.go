package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zimap/internal/setup"
)

func TestOutputMapRejectsSymlinkEscapes(t *testing.T) {
	for _, operation := range []string{"delete", "export"} {
		for _, linkType := range []string{"file", "directory"} {
			t.Run(operation+"/"+linkType, func(t *testing.T) {
				t.Setenv("ZIMAP_HOME", t.TempDir())
				outputDir, err := setup.OutputDirectory()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					t.Fatal(err)
				}

				outsideDir := t.TempDir()
				outsideMap := filepath.Join(outsideDir, "private.img")
				if err := os.WriteFile(outsideMap, []byte("private map"), 0644); err != nil {
					t.Fatal(err)
				}

				linkPath := filepath.Join(outputDir, "linked.img")
				linkTarget := outsideMap
				mapPath := linkPath
				if linkType == "directory" {
					linkPath = filepath.Join(outputDir, "linked")
					linkTarget = outsideDir
					mapPath = filepath.Join(linkPath, "private.img")
				}
				if err := os.Symlink(linkTarget, linkPath); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}

				app := &App{}
				if operation == "delete" {
					err = app.DeleteOutput(mapPath)
				} else {
					_, err = app.ExportMap(mapPath)
				}
				if err == nil || strings.Contains(err.Error(), "window context not initialized") {
					t.Errorf("expected path rejection before accessing the map, got %v", err)
				}
				content, err := os.ReadFile(outsideMap)
				if err != nil || string(content) != "private map" {
					t.Errorf("outside map was modified: content=%q, error=%v", content, err)
				}
			})
		}
	}
}

func TestDeleteOutputThroughSymlinkedOutputDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZIMAP_HOME", home)
	actualOutput := t.TempDir()
	if err := os.Symlink(actualOutput, filepath.Join(home, "output")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	mapPath := filepath.Join(actualOutput, "region.IMG")
	if err := os.WriteFile(mapPath, []byte("map"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := (&App{}).DeleteOutput("region.IMG"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mapPath); !os.IsNotExist(err) {
		t.Fatalf("expected map deletion, got %v", err)
	}
}

func TestCopyFilePreservesSourceAliases(t *testing.T) {
	for _, alias := range []string{"same path", "hard link"} {
		t.Run(alias, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "source.img")
			if err := os.WriteFile(source, []byte("original map"), 0644); err != nil {
				t.Fatal(err)
			}
			destination := source
			if alias == "hard link" {
				destination = filepath.Join(filepath.Dir(source), "alias.img")
				if err := os.Link(source, destination); err != nil {
					t.Skipf("hard links unavailable: %v", err)
				}
			}
			if err := copyFile(source, destination); err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(source)
			if err != nil || string(content) != "original map" {
				t.Fatalf("copying onto an alias changed the source: content=%q, error=%v", content, err)
			}
		})
	}
}
