package setup

import (
	"archive/zip"
	"bufio"
	"fmt"
	"zimap/internal/archive"
	"zimap/internal/download"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var downloader = download.NewStreaming()

type Tool struct {
	Name      string
	URL       string
	Path      string
	RunUpdate func(tool Tool) error
}

var Tools = []Tool{
	{
		Name:      "java",
		URL:       "https://api.adoptium.net/v3/binary/latest/21/ga/{os}/{arch}/jre/hotspot/normal/eclipse?project=jdk",
		Path:      "tools/jdk",
		RunUpdate: UpdateJava,
	},
	{
		Name:      "mkgmap",
		URL:       "https://www.mkgmap.org.uk/download/mkgmap-r4924.zip",
		Path:      "tools/mkgmap/mkgmap.jar",
		RunUpdate: UpdateTool,
	},
	{
		Name:      "splitter",
		URL:       "https://www.mkgmap.org.uk/download/splitter-r654.zip",
		Path:      "tools/splitter/splitter.jar",
		RunUpdate: UpdateTool,
	},
}

func JavaDetails() (path string, isEmbedded bool, err error) {
	var relativeCandidates []string
	if runtime.GOOS == "darwin" {
		relativeCandidates = append(relativeCandidates, "tools/jdk/Contents/Home/bin/java")
	} else {
		javaName := "java"
		if runtime.GOOS == "windows" {
			javaName = "java.exe"
		}
		relativeCandidates = append(relativeCandidates, filepath.Join("tools", "jdk", "bin", javaName))
	}

	for _, candidate := range relativeCandidates {
		p, err := projectPath(candidate)
		if err == nil {
			if info, statErr := os.Stat(p); statErr == nil && !info.IsDir() {
				return p, true, nil
			}
		}
	}

	if p, err := exec.LookPath("java"); err == nil {
		return p, false, nil
	}
	return "", false, fmt.Errorf("java runtime not found; install Java 8 or newer or place it under tools/jdk")
}

func JavaBinPath() (string, error) {
	p, _, err := JavaDetails()
	return p, err
}

func ToolDetails(toolName string) (path string, isEmbedded bool, err error) {
	p, err := toolBinPath(toolName)
	if err != nil {
		return "", false, err
	}
	return p, true, nil
}

func MkgmapBinPath() (string, error) {
	p, _, err := ToolDetails("mkgmap")
	return p, err
}

func SplitterBinPath() (string, error) {
	p, _, err := ToolDetails("splitter")
	return p, err
}

func toolBinPath(toolName string) (string, error) {
	for _, tool := range Tools {
		if tool.Name != toolName {
			continue
		}
		path, err := projectPath(tool.Path)
		if err != nil {
			return "", err
		}
		if err := validateMkgmapDistribution(path); err != nil {
			return "", fmt.Errorf("invalid %s installation: %w; run the update command", toolName, err)
		}
		return path, nil
	}
	return "", fmt.Errorf("%s tool is not configured", toolName)
}

// UpdateTool downloads and validates a release before replacing the currently
// installed version. The staging directory is next to the destination so the
// final rename stays on the same filesystem.
func UpdateTool(tool Tool) error {
	targetPath, err := projectPath(tool.Path)
	if err != nil {
		return err
	}
	targetDirectory := filepath.Dir(targetPath)
	stagingParent := filepath.Dir(targetDirectory)
	if err := os.MkdirAll(stagingParent, 0755); err != nil {
		return fmt.Errorf("create tool directory: %w", err)
	}

	stagingDirectory, err := os.MkdirTemp(stagingParent, ".mkgmap-update-*")
	if err != nil {
		return fmt.Errorf("create staging directory for %s: %w", tool.Name, err)
	}
	defer os.RemoveAll(stagingDirectory)

	downloadPath := filepath.Join(stagingDirectory, filepath.Base(tool.URL))
	fmt.Printf("Downloading %s...\n", tool.Name)
	if err := downloader.Download(tool.URL, downloadPath); err != nil {
		return fmt.Errorf("download %s: %w", tool.Name, err)
	}

	candidatePath := downloadPath
	installSourcePath := downloadPath
	installTargetPath := targetPath
	if strings.EqualFold(filepath.Ext(downloadPath), ".zip") {
		extractedPath := filepath.Join(stagingDirectory, "extracted")
		if err := archive.ExtractArchive(downloadPath, extractedPath); err != nil {
			return fmt.Errorf("extract %s: %w", tool.Name, err)
		}
		candidatePath, err = findFile(extractedPath, filepath.Base(tool.Path))
		if err != nil {
			return fmt.Errorf("locate %s in release: %w", filepath.Base(tool.Path), err)
		}
		installSourcePath = filepath.Dir(candidatePath)
		installTargetPath = targetDirectory
	} else if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		return fmt.Errorf("create tool directory: %w", err)
	}

	if err := validateMkgmapDistribution(candidatePath); err != nil {
		return fmt.Errorf("validate downloaded %s: %w", tool.Name, err)
	}
	if err := replacePath(installSourcePath, installTargetPath); err != nil {
		return fmt.Errorf("install %s: %w", tool.Name, err)
	}

	fmt.Printf("%s updated successfully.\n", tool.Name)
	return nil
}

func UpdateJava(tool Tool) error {
	targetPath, err := projectPath(tool.Path)
	if err != nil {
		return err
	}
	parentPath := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentPath, 0755); err != nil {
		return err
	}

	stagingDirectory, err := os.MkdirTemp(parentPath, ".jdk-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stagingDirectory)

	replacer := strings.NewReplacer(
		"{os}", adoptiumOS(runtime.GOOS),
		"{arch}", adoptiumArch(runtime.GOARCH),
	)
	archivePath := filepath.Join(stagingDirectory, SetJavaArchiveName(runtime.GOOS))
	fmt.Println("Downloading Java...")
	if err := downloader.Download(replacer.Replace(tool.URL), archivePath); err != nil {
		return fmt.Errorf("download Java archive: %w", err)
	}

	extractedPath := filepath.Join(stagingDirectory, "extracted")
	if err := archive.ExtractArchive(archivePath, extractedPath); err != nil {
		return fmt.Errorf("extract JDK archive: %w", err)
	}
	sourcePath, err := singleExtractedRoot(extractedPath)
	if err != nil {
		return err
	}
	if err := validateJavaDistribution(sourcePath, runtime.GOOS); err != nil {
		return fmt.Errorf("validate downloaded Java: %w", err)
	}
	if err := replacePath(sourcePath, targetPath); err != nil {
		return fmt.Errorf("install Java: %w", err)
	}
	fmt.Println("Java updated successfully.")
	return nil
}

func validateJavaDistribution(rootPath string, osName string) error {
	javaPath := javaExecutablePath(rootPath, osName)
	info, err := os.Stat(javaPath)
	if err != nil {
		return fmt.Errorf("Java executable not found at %s: %w", javaPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("Java executable is not a regular file: %s", javaPath)
	}
	if osName != "windows" && info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("Java executable is not executable: %s", javaPath)
	}
	return nil
}

func javaExecutablePath(rootPath string, osName string) string {
	if osName == "darwin" {
		return filepath.Join(rootPath, "Contents", "Home", "bin", "java")
	}
	if osName == "windows" {
		return filepath.Join(rootPath, "bin", "java.exe")
	}
	return filepath.Join(rootPath, "bin", "java")
}

func validateJAR(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if strings.EqualFold(entry.Name, "META-INF/MANIFEST.MF") {
			return nil
		}
	}
	return fmt.Errorf("manifest not found")
}

func validateMkgmapDistribution(jarPath string) error {
	if err := validateJAR(jarPath); err != nil {
		return err
	}

	classPath, err := manifestAttribute(jarPath, "Class-Path")
	if err != nil {
		return err
	}
	for _, dependency := range strings.Fields(classPath) {
		dependencyPath := filepath.Join(filepath.Dir(jarPath), filepath.FromSlash(dependency))
		relativePath, relErr := filepath.Rel(filepath.Dir(jarPath), dependencyPath)
		if relErr != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe manifest dependency %q", dependency)
		}
		dependencyArchive, openErr := zip.OpenReader(dependencyPath)
		if openErr != nil {
			return fmt.Errorf("missing or invalid dependency %q: %w", dependency, openErr)
		}
		dependencyArchive.Close()
	}
	return nil
}

func manifestAttribute(jarPath string, attributeName string) (string, error) {
	reader, err := zip.OpenReader(jarPath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var manifest io.ReadCloser
	for _, entry := range reader.File {
		if strings.EqualFold(entry.Name, "META-INF/MANIFEST.MF") {
			manifest, err = entry.Open()
			if err != nil {
				return "", err
			}
			break
		}
	}
	if manifest == nil {
		return "", fmt.Errorf("manifest not found")
	}
	defer manifest.Close()

	prefix := strings.ToLower(attributeName) + ":"
	var currentAttribute string
	var currentValue strings.Builder
	flush := func() (string, bool) {
		if strings.ToLower(currentAttribute) == prefix {
			return strings.TrimSpace(currentValue.String()), true
		}
		return "", false
	}

	scanner := bufio.NewScanner(manifest)
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if strings.HasPrefix(line, " ") {
			currentValue.WriteString(strings.TrimPrefix(line, " "))
			continue
		}
		if value, found := flush(); found {
			return value, nil
		}
		separator := strings.IndexByte(line, ':')
		if separator < 0 {
			currentAttribute = ""
			currentValue.Reset()
			continue
		}
		currentAttribute = line[:separator+1]
		currentValue.Reset()
		currentValue.WriteString(strings.TrimSpace(line[separator+1:]))
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if value, found := flush(); found {
		return value, nil
	}
	return "", nil
}

func findFile(rootPath string, name string) (string, error) {
	var match string
	err := filepath.WalkDir(rootPath, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && entry.Name() == name {
			match = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if match == "" {
		return "", os.ErrNotExist
	}
	return match, nil
}

func singleExtractedRoot(extractedPath string) (string, error) {
	entries, err := os.ReadDir(extractedPath)
	if err != nil {
		return "", err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		return filepath.Join(extractedPath, entries[0].Name()), nil
	}
	return extractedPath, nil
}

func replacePath(sourcePath string, targetPath string) error {
	backupPath, err := os.MkdirTemp(filepath.Dir(targetPath), ".previous-*")
	if err != nil {
		return err
	}
	if err := os.Remove(backupPath); err != nil {
		return err
	}
	defer os.RemoveAll(backupPath)

	_, statErr := os.Stat(targetPath)
	hasExistingTarget := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if hasExistingTarget {
		if err := os.Rename(targetPath, backupPath); err != nil {
			return err
		}
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		if hasExistingTarget {
			_ = os.Rename(backupPath, targetPath)
		}
		return err
	}
	return nil
}

func SetJavaArchiveName(osName string) string {
	if osName == "windows" {
		return "jdk.zip"
	}
	return "jdk.tar.gz"
}

func adoptiumOS(osName string) string {
	if osName == "darwin" {
		return "mac"
	}
	return osName
}

func adoptiumArch(arch string) string {
	switch arch {
	case "amd64":
		return "x64"
	case "arm64":
		return "aarch64"
	default:
		return arch
	}
}
