package download

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDownloadCancelledBeforePublishPreservesExistingFile(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := New(clientWithResponse(http.StatusOK, io.NopCloser(strings.NewReader("replacement"))))
	target := filepath.Join(t.TempDir(), "map.pbf")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	err := client.DownloadWithOptions(ctx, "https://example.test/map.pbf", target, DownloadOptions{
		MaxRetries: 1,
		ProgressCallback: func(int64, int64, float64) {
			cancel()
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected cancellation, got %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "original" {
		t.Errorf("cancelled download changed the existing file: content=%q, error=%v", content, err)
	}
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil || len(entries) != 1 {
		t.Errorf("temporary download files were not cleaned up: entries=%v, error=%v", entries, err)
	}
}

func TestDownloadDoesNotReplaceDirectory(t *testing.T) {
	target := filepath.Join(t.TempDir(), "map.pbf")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(original, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}
	client := New(clientWithResponse(http.StatusOK, io.NopCloser(strings.NewReader("replacement"))))
	if err := client.DownloadWithOptions(t.Context(), "https://example.test/map.pbf", target, DownloadOptions{MaxRetries: 1}); err == nil {
		t.Error("expected an error when the target is a directory")
	}
	content, err := os.ReadFile(original)
	if err != nil || string(content) != "keep" {
		t.Fatalf("download moved or changed the existing directory: content=%q, error=%v", content, err)
	}
}

func TestReplaceFilePreservesTargetOnFailure(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "map.pbf")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(filepath.Join(directory, "missing.pbf"), target); err == nil {
		t.Error("expected an error for a missing replacement")
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "original" {
		t.Fatalf("failed replacement changed the target: content=%q, error=%v", content, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected backup files: entries=%v, error=%v", entries, err)
	}
}

func TestReplaceFileReplacesExistingFile(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "map.pbf")
	source := filepath.Join(directory, "download.tmp")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := replaceFile(source, target); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(target)
	if err != nil || string(content) != "replacement" {
		t.Fatalf("replacement failed: content=%q, error=%v", content, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("unexpected temporary files: entries=%v, error=%v", entries, err)
	}
}
