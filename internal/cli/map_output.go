package cli

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func buildFinalMap(outputDirectory string, fileName string, build func(workDirectory string) (string, error)) (string, error) {
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	workDirectory, err := os.MkdirTemp(outputDirectory, ".zimap-"+fileName+"-*")
	if err != nil {
		return "", fmt.Errorf("create map workspace: %w", err)
	}

	convertedMapPath, buildErr := build(workDirectory)
	if buildErr != nil {
		cleanupErr := os.RemoveAll(workDirectory)
		if cleanupErr != nil {
			return "", errors.Join(buildErr, fmt.Errorf("clean map workspace: %w", cleanupErr))
		}
		return "", buildErr
	}

	finalMapPath := filepath.Join(outputDirectory, fileName+".img")
	if err := publishFinalMap(convertedMapPath, finalMapPath); err != nil {
		cleanupErr := os.RemoveAll(workDirectory)
		if cleanupErr != nil {
			return "", errors.Join(err, fmt.Errorf("clean map workspace: %w", cleanupErr))
		}
		return "", err
	}
	if err := os.RemoveAll(workDirectory); err != nil {
		return finalMapPath, fmt.Errorf("map created at %s, but cleanup failed: %w", finalMapPath, err)
	}
	return finalMapPath, nil
}

func publishFinalMap(sourcePath string, targetPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("inspect converted map: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("converted map %s is a directory", sourcePath)
	}
	if !strings.EqualFold(filepath.Ext(targetPath), ".img") {
		return fmt.Errorf("final map must have an .img extension: %s", targetPath)
	}

	backupFile, err := os.CreateTemp(filepath.Dir(targetPath), ".previous-map-*.img")
	if err != nil {
		return fmt.Errorf("reserve map backup path: %w", err)
	}
	backupPath := backupFile.Name()
	if err := backupFile.Close(); err != nil {
		return fmt.Errorf("close map backup placeholder: %w", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf("prepare map backup path: %w", err)
	}
	defer os.Remove(backupPath)

	_, statErr := os.Lstat(targetPath)
	hasExistingTarget := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return fmt.Errorf("inspect existing map: %w", statErr)
	}
	if hasExistingTarget {
		if err := os.Rename(targetPath, backupPath); err != nil {
			return fmt.Errorf("back up existing map: %w", err)
		}
	}
	if err := os.Rename(sourcePath, targetPath); err != nil {
		if hasExistingTarget {
			_ = os.Rename(backupPath, targetPath)
		}
		return fmt.Errorf("publish final map: %w", err)
	}
	return nil
}

func mapFileNameFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "map"
	}
	name := path.Base(parsedURL.Path)
	name = strings.TrimSuffix(name, ".osm.pbf")
	name = strings.TrimSuffix(name, "-latest")
	return regionFileName(name)
}
