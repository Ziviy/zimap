package archive

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Handler interface {
	CanProcess(extension string) bool
	Process(archivePath string, destinationPath string) error
}

func ExtractArchive(archivePath string, destinationPath string) error {
	fileExt := strings.ToLower(filepath.Ext(archivePath))

	handlers := []Handler{
		&TarHandler{},
		&TargzHandler{},
		&ZipHandler{},
	}

	for _, handler := range handlers {
		if handler.CanProcess(fileExt) {
			return handler.Process(archivePath, destinationPath)
		}
	}

	return fmt.Errorf("unsupported archive format %q", fileExt)
}
