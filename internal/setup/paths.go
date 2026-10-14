package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const homeEnvironmentVariable = "ZIMAP_HOME"
const legacyHomeEnvironmentVariable = "GARMIN_MAP_HOME"

// ProjectRoot returns the directory used for mutable application data. During
// development it finds the nearest go.mod; installed binaries can override the
// location with ZIMAP_HOME (or legacy GARMIN_MAP_HOME).
func ProjectRoot() (string, error) {
	if configuredRoot := os.Getenv(homeEnvironmentVariable); configuredRoot != "" {
		return filepath.Abs(configuredRoot)
	}
	if configuredRoot := os.Getenv(legacyHomeEnvironmentVariable); configuredRoot != "" {
		return filepath.Abs(configuredRoot)
	}

	workingDirectory, err := os.Getwd()
	if err == nil {
		if root, found := findProjectRoot(workingDirectory); found {
			return root, nil
		}
	}

	executable, executableErr := os.Executable()
	if executableErr == nil {
		if root, found := findProjectRoot(filepath.Dir(executable)); found {
			return root, nil
		}
	}

	if err != nil {
		return "", fmt.Errorf("determine working directory: %w", err)
	}
	return workingDirectory, nil
}

func findProjectRoot(start string) (string, bool) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}

	for {
		if info, statErr := os.Stat(filepath.Join(directory, "go.mod")); statErr == nil && !info.IsDir() {
			return directory, true
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", false
		}
		directory = parent
	}
}

func projectPath(relativePath string) (string, error) {
	if filepath.IsAbs(relativePath) {
		return filepath.Clean(relativePath), nil
	}
	root, err := ProjectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, relativePath), nil
}

func OutputDirectory() (string, error) {
	return projectPath("output")
}

func CacheDirectory() (string, error) {
	return projectPath("cache")
}

func ToolsDirectory() (string, error) {
	return projectPath("tools")
}
