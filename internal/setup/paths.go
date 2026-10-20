package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const homeEnvironmentVariable = "ZIMAP_HOME"
const legacyHomeEnvironmentVariable = "GARMIN_MAP_HOME"
const appDirName = "ZiMap"
const portableDirName = "zimap-data"
const portableMarkerFile = ".portable"

// DataDirectory returns the directory used for mutable application data
// (compiled maps, catalog cache, proxy config, and managed tools).
func DataDirectory() (string, error) {
	return ProjectRoot()
}

// ProjectRoot returns the directory used for mutable application data.
// Resolution order:
//  1. ZIMAP_HOME environment variable (if set)
//  2. GARMIN_MAP_HOME environment variable (legacy override, if set)
//  3. Portable data directory (<exeDir>/zimap-data) when a portable marker
//     (.portable or zimap-data/) exists next to the executable
//  4. OS-standard private application data directory:
//     - macOS:   ~/Library/Application Support/ZiMap
//     - Windows: %LOCALAPPDATA%\ZiMap (or %APPDATA%\ZiMap)
//     - Linux:   $XDG_DATA_HOME/ZiMap or ~/.local/share/ZiMap
//  5. Working directory fallback if OS directories cannot be determined.
func ProjectRoot() (string, error) {
	if configuredRoot := os.Getenv(homeEnvironmentVariable); configuredRoot != "" {
		return filepath.Abs(configuredRoot)
	}
	if configuredRoot := os.Getenv(legacyHomeEnvironmentVariable); configuredRoot != "" {
		return filepath.Abs(configuredRoot)
	}

	if executable, err := os.Executable(); err == nil {
		if dir, ok := detectPortableDir(executable); ok {
			return dir, nil
		}
	}

	if dataDir, err := defaultDataDirectory(runtime.GOOS); err == nil && dataDir != "" {
		return dataDir, nil
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("determine working directory: %w", err)
	}
	return workingDirectory, nil
}

func detectPortableDir(executablePath string) (string, bool) {
	slashed := filepath.ToSlash(executablePath)
	if strings.Contains(slashed, ".app/Contents/MacOS/") {
		return "", false
	}
	exeDir := filepath.Dir(executablePath)
	portableDir := filepath.Join(exeDir, portableDirName)
	if info, err := os.Stat(portableDir); err == nil && info.IsDir() {
		if abs, absErr := filepath.Abs(portableDir); absErr == nil {
			return abs, true
		}
		return portableDir, true
	}
	markerPath := filepath.Join(exeDir, portableMarkerFile)
	if info, err := os.Stat(markerPath); err == nil && !info.IsDir() {
		if abs, absErr := filepath.Abs(portableDir); absErr == nil {
			return abs, true
		}
		return portableDir, true
	}
	return "", false
}

func defaultDataDirectory(goos string) (string, error) {
	switch goos {
	case "darwin":
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			return filepath.Join(cfgDir, appDirName), nil
		}
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			return filepath.Join(homeDir, "Library", "Application Support", appDirName), nil
		}
	case "windows":
		if localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA")); localAppData != "" {
			return filepath.Join(localAppData, appDirName), nil
		}
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			return filepath.Join(cfgDir, appDirName), nil
		}
		if exePath, err := os.Executable(); err == nil {
			return filepath.Join(filepath.Dir(exePath), portableDirName), nil
		}
	default:
		if xdgData := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); xdgData != "" {
			return filepath.Join(xdgData, appDirName), nil
		}
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			return filepath.Join(homeDir, ".local", "share", appDirName), nil
		}
		if cfgDir, err := os.UserConfigDir(); err == nil && cfgDir != "" {
			return filepath.Join(cfgDir, appDirName), nil
		}
	}
	return "", fmt.Errorf("unable to resolve default application data directory for %s", goos)
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
