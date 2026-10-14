package archive

import (
	"archive/tar"
	"compress/gzip"
	"os"
)

type TargzHandler struct{}

func (handler *TargzHandler) CanProcess(extension string) bool {
	return extension == ".gz" || extension == ".tgz"
}

func (handler *TargzHandler) Process(archivePath string, destinationPath string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	if err := os.MkdirAll(destinationPath, 0755); err != nil {
		return err
	}
	return extractTar(tar.NewReader(gzipReader), destinationPath)
}
