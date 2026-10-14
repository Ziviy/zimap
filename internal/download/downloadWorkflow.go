package download

import (
	"fmt"
	"net/url"
	"path/filepath"
)

func ProcessMapDownload(rawURL string, outputDirectory string) (string, error) {
	return ProcessMapDownloadTo(rawURL, filepath.Join(outputDirectory, "map.osm.pbf"))
}

func ProcessMapDownloadTo(rawURL string, mapFilePath string) (string, error) {
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid map URL %q", rawURL)
	}

	client := NewStreaming()
	err = client.Download(rawURL, mapFilePath)
	if err != nil {
		return "", err
	}

	return mapFilePath, nil
}

func DownloadTool(url string, filePath string) error {
	client := New(nil)

	err := client.Download(url, filePath)
	if err != nil {
		return err
	}

	return nil
}
