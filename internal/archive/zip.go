package archive

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
)

type ZipHandler struct{}

func (handler *ZipHandler) CanProcess(extension string) bool {
	return extension == ".zip"
}

func (handler *ZipHandler) Process(archivePath string, destinationPath string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return err
	}

	for _, entry := range reader.File {
		targetPath, err := secureArchivePath(destinationPath, entry.Name)
		if err != nil {
			return err
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsupported zip symlink %q", entry.Name)
		}
		if entry.FileInfo().IsDir() {
			if err := ensureNoSymlinkComponents(destinationPath, targetPath); err != nil {
				return err
			}
			if err := os.MkdirAll(targetPath, entry.Mode().Perm()); err != nil {
				return err
			}
			continue
		}

		entryReader, err := entry.Open()
		if err != nil {
			return err
		}
		err = writeArchiveFile(entryReader, destinationPath, targetPath, entry.Mode().Perm())
		closeErr := entryReader.Close()
		if err != nil {
			return fmt.Errorf("extract %q: %w", entry.Name, err)
		}
		if closeErr != nil {
			return fmt.Errorf("close %q: %w", filepath.Base(entry.Name), closeErr)
		}
	}
	return nil
}
