package archive

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TarHandler struct{}

func (handler *TarHandler) CanProcess(extension string) bool {
	return extension == ".tar"
}

func (handler *TarHandler) Process(archivePath string, destinationPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return err
	}
	return extractTar(tar.NewReader(file), destinationPath)
}

func extractTar(reader *tar.Reader, destinationPath string) error {
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}

		target, err := secureArchivePath(destinationPath, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := ensureNoSymlinkComponents(destinationPath, target); err != nil {
				return err
			}
			if err := os.MkdirAll(target, header.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := writeArchiveFile(reader, destinationPath, target, header.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		case tar.TypeXGlobalHeader:
			continue
		default:
			return fmt.Errorf("unsupported tar entry type %d for %q", header.Typeflag, header.Name)
		}
	}
}

func writeArchiveFile(reader io.Reader, rootPath string, targetPath string, mode os.FileMode) error {
	if err := ensureNoSymlinkComponents(rootPath, filepath.Dir(targetPath)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}
	if info, err := os.Lstat(targetPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to overwrite symlink %q", targetPath)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, reader); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func secureArchivePath(rootPath string, entryName string) (string, error) {
	cleanName := filepath.Clean(filepath.FromSlash(entryName))
	if filepath.IsAbs(cleanName) {
		return "", fmt.Errorf("unsafe archive path %q", entryName)
	}

	targetPath := filepath.Join(rootPath, cleanName)
	relativePath, err := filepath.Rel(rootPath, targetPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive path escapes destination: %q", entryName)
	}
	return targetPath, nil
}

func ensureNoSymlinkComponents(rootPath string, targetPath string) error {
	relativePath, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return err
	}
	currentPath := rootPath
	for _, component := range strings.Split(relativePath, string(filepath.Separator)) {
		if component == "." || component == "" {
			continue
		}
		currentPath = filepath.Join(currentPath, component)
		info, statErr := os.Lstat(currentPath)
		if os.IsNotExist(statErr) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive target crosses symlink %q", currentPath)
		}
	}
	return nil
}
