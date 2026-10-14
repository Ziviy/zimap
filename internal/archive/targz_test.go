package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createTarGz(t *testing.T, name string, content []byte) string {
	t.Helper()
	archivePath := filepath.Join(t.TempDir(), "archive.tar.gz")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gzWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzWriter)

	if err := tarWriter.WriteHeader(&tar.Header{
		Name: name,
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return archivePath
}

func TestTargzHandlerCanProcess(t *testing.T) {
	handler := &TargzHandler{}
	if !handler.CanProcess(".gz") {
		t.Errorf("expected CanProcess(.gz) to be true")
	}
	if !handler.CanProcess(".tgz") {
		t.Errorf("expected CanProcess(.tgz) to be true")
	}
	if handler.CanProcess(".tar") {
		t.Errorf("expected CanProcess(.tar) to be false")
	}
	if handler.CanProcess(".zip") {
		t.Errorf("expected CanProcess(.zip) to be false")
	}
}

func TestTargzExtractsRegularFile(t *testing.T) {
	archivePath := createTarGz(t, "subfolder/hello.txt", []byte("tar.gz content"))
	destDir := t.TempDir()

	if err := ExtractArchive(archivePath, destDir); err != nil {
		t.Fatalf("ExtractArchive(.tar.gz) failed: %v", err)
	}

	extractedFile := filepath.Join(destDir, "subfolder", "hello.txt")
	got, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(got) != "tar.gz content" {
		t.Errorf("got %q, want %q", string(got), "tar.gz content")
	}
}

func TestTargzRejectsPathTraversal(t *testing.T) {
	archivePath := createTarGz(t, "../../escaped_targz.txt", []byte("danger"))
	destParent := t.TempDir()
	destDir := filepath.Join(destParent, "safe_dir")

	err := ExtractArchive(archivePath, destDir)
	if err == nil || !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("expected path traversal error, got: %v", err)
	}

	escaped := filepath.Join(destParent, "escaped_targz.txt")
	if _, err := os.Stat(escaped); !os.IsNotExist(err) {
		t.Fatalf("security violation: file was extracted outside target directory")
	}
}

func TestTargzInvalidGzipHeader(t *testing.T) {
	corruptPath := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	if err := os.WriteFile(corruptPath, []byte("not a valid gzip header"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := t.TempDir()
	err := ExtractArchive(corruptPath, destDir)
	if err == nil {
		t.Errorf("expected error extracting invalid gzip file, got nil")
	}
}

func TestTargzTruncatedArchive(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: "data.bin", Mode: 0644, Size: 1024})
	_, _ = tw.Write(make([]byte, 1024))
	_ = tw.Close()
	_ = gw.Close()

	// Truncate the gzip stream
	truncatedData := buf.Bytes()[:len(buf.Bytes())/2]
	truncPath := filepath.Join(t.TempDir(), "truncated.tgz")
	_ = os.WriteFile(truncPath, truncatedData, 0644)

	destDir := t.TempDir()
	err := ExtractArchive(truncPath, destDir)
	if err == nil {
		t.Errorf("expected error extracting truncated archive, got nil")
	}
}
