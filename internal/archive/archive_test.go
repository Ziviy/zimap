package archive

import (
	"archive/tar"
	archivezip "archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTarExtractsRegularFile(t *testing.T) {
	archivePath := createTar(t, "folder/file.txt", []byte("content"))
	destinationPath := t.TempDir()

	if err := ExtractArchive(archivePath, destinationPath); err != nil {
		t.Fatalf("ExtractArchive() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(destinationPath, "folder", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "content" {
		t.Fatalf("content = %q, want %q", content, "content")
	}
}

func TestTarRejectsPathTraversal(t *testing.T) {
	archivePath := createTar(t, "../../escaped.txt", []byte("unsafe"))
	destinationParent := t.TempDir()
	destinationPath := filepath.Join(destinationParent, "destination")

	err := ExtractArchive(archivePath, destinationPath)
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("ExtractArchive() error = %v, want path traversal error", err)
	}
	if _, statErr := os.Stat(filepath.Join(destinationParent, "escaped.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("archive wrote outside destination, stat error = %v", statErr)
	}
}

func TestTarReturnsTruncationError(t *testing.T) {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	if err := writer.WriteHeader(&tar.Header{Name: "file.txt", Mode: 0644, Size: 1024}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(make([]byte, 1024)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "truncated.tar")
	if err := os.WriteFile(archivePath, buffer.Bytes()[:700], 0644); err != nil {
		t.Fatal(err)
	}
	if err := ExtractArchive(archivePath, t.TempDir()); err == nil {
		t.Fatal("ExtractArchive() error = nil, want truncated archive error")
	}
}

func TestExtractArchiveRejectsUnknownFormat(t *testing.T) {
	err := ExtractArchive("archive.rar", t.TempDir())
	if err == nil {
		t.Fatal("ExtractArchive() error = nil, want unsupported format error")
	}
}

func TestZipRejectsPathTraversal(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "archive.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := archivezip.NewWriter(file)
	entry, err := writer.Create("../escaped.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("unsafe")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	destinationParent := t.TempDir()
	err = ExtractArchive(archivePath, filepath.Join(destinationParent, "destination"))
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("ExtractArchive() error = %v, want path traversal error", err)
	}
	if _, statErr := os.Stat(filepath.Join(destinationParent, "escaped.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("archive wrote outside destination, stat error = %v", statErr)
	}
}

func createTar(t *testing.T, name string, content []byte) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "archive.tar")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(file)
	if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}
