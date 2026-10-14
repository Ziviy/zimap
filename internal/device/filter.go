package device

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// topoRegex matches Garmin OEM part-number maps (e.g. D1399570A.img, D9218040B.img, DA599010A.img)
	topoRegex = regexp.MustCompile(`(?i)^d[a-z0-9]{7,8}\.img$`)

	systemPrefixes = []string{
		"gmapbmap",
		"gmaptz",
		"gmapdem",
		"gmapprom",
		"gmapgc",
		"gmap3d",
	}

	ErrSystemMapProtected = errors.New("cannot delete system Garmin map: file is protected")
	ErrPathOutsideDevice  = errors.New("access denied: path is outside Garmin device mount")
	ErrNotAnImgFile       = errors.New("only .img files can be deleted")

	// knownGarminMapTitles provides authentic human-readable names for standard Garmin system maps.
	knownGarminMapTitles = map[string]string{
		"gmapbmap.img":  "Garmin Worldwide Basemap",
		"gmaptz.img":    "Garmin Time Zone Map",
		"gmapdem.img":   "Garmin Worldwide DEM",
		"gmapdem1.img":  "Garmin DEM",
		"gmapprom.img":  "Garmin TopoActive",
		"gmapprom1.img": "Garmin TopoActive Secondary",
		"gmapgc.img":    "Garmin Geocoding Database",
		"gmap3d.img":    "Garmin 3D Landmarks",
		"d9218040b.img": "Garmin SkiView (Ski Resorts)",
		"d9225010a.img": "Garmin CourseView (Golf)",
		"d1399570a.img": "Garmin TopoActive Europe",
		"d2419520a.img": "Garmin TopoActive Europe (East)",
		"d2471560a.img": "Garmin TopoActive Europe (Central)",
		"da599010a.img": "Garmin TopoActive DEM / Region",
		"da597010a.img": "Garmin TopoActive Region",
	}
)

// HasGarminSignatureFiles checks if official companion authentication files (.gma, .unl)
// exist alongside the map file on disk.
func HasGarminSignatureFiles(filePath string) (bool, string) {
	if filePath == "" {
		return false, ""
	}
	base := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	if _, err := os.Stat(base + ".gma"); err == nil {
		return true, "gma_signature"
	}
	if _, err := os.Stat(base + ".unl"); err == nil {
		return true, "unl_license"
	}
	return false, ""
}

// HasGarminSignatureHeader checks if the internal title contains official Garmin brand signatures.
func HasGarminSignatureHeader(internalTitle string) (bool, string) {
	lower := strings.ToLower(internalTitle)
	keywords := []string{
		"garmin",
		"topoactive",
		"skiview",
		"courseview",
		"basemap",
		"cycle map",
		"city navigator",
		"dem basemap",
		"navteq",
		"here",
	}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true, "header_signature"
		}
	}
	return false, ""
}

// ResolveGarminMapTitle returns a clean user-friendly title for Garmin factory maps.
func ResolveGarminMapTitle(fileName, internalTitle string) string {
	base := strings.ToLower(filepath.Base(fileName))
	if known, exists := knownGarminMapTitles[base]; exists {
		return known
	}
	if internalTitle != "" && !strings.EqualFold(internalTitle, strings.TrimSuffix(base, ".img")) {
		return internalTitle
	}
	if topoRegex.MatchString(base) {
		return "Garmin OEM (" + strings.ToUpper(strings.TrimSuffix(base, ".img")) + ")"
	}
	return strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
}

// DetectPreinstalledMap inspects a map's filename, full path, and internal title to determine
// if it is an official factory/preinstalled Garmin map with digital signature.
func DetectPreinstalledMap(fileName, fullPath, internalTitle string) (isPreinstalled bool, signatureType string) {
	// 1. Check for companion signature/license files (.gma, .unl, .sum)
	if fullPath != "" {
		if hasSig, sigType := HasGarminSignatureFiles(fullPath); hasSig {
			return true, sigType
		}
	}

	// 2. Check internal title/header signature
	if internalTitle != "" {
		if hasSig, sigType := HasGarminSignatureHeader(internalTitle); hasSig {
			return true, sigType
		}
	}

	// 3. Check filename pattern & system prefixes
	if isOEMFilename(fileName) {
		return true, "oem_system"
	}

	return false, ""
}

// isOEMFilename checks if the filename matches Garmin system prefixes or OEM part numbers.
func isOEMFilename(fileName string) bool {
	base := strings.ToLower(filepath.Base(fileName))
	if !strings.HasSuffix(base, ".img") {
		return true // non-.img files are strictly protected
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(base, prefix) {
			return true
		}
	}

	if topoRegex.MatchString(base) {
		digitCount := 0
		for _, r := range base {
			if r >= '0' && r <= '9' {
				digitCount++
			}
		}
		if digitCount >= 4 {
			return true
		}
	}

	return false
}

// IsSystemMap returns true if the filename represents a protected factory/system Garmin map.
func IsSystemMap(fileName string) bool {
	return isOEMFilename(fileName)
}

// ValidateDeletePath checks that the target file is safely located inside
// <mountPath>/GARMIN/ or <mountPath>/GARMIN/Map/, is an .img file, is not a system map,
// and does not contain path traversal. Returns the resolved clean absolute path.
func ValidateDeletePath(mountPath, filePath string) (string, error) {
	if mountPath == "" || filePath == "" {
		return "", errors.New("mount path and file path must not be empty")
	}

	absMount, err := filepath.Abs(filepath.Clean(mountPath))
	if err != nil {
		return "", fmt.Errorf("invalid mount path: %w", err)
	}

	var targetPath string
	if filepath.IsAbs(filePath) {
		targetPath = filepath.Clean(filePath)
	} else {
		targetPath = filepath.Clean(filepath.Join(absMount, filePath))
	}

	// Verify targetPath is within absMount
	rel, err := filepath.Rel(absMount, targetPath)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrPathOutsideDevice
	}

	// Ensure target is inside GARMIN or GARMIN/Map
	relLower := strings.ToLower(filepath.ToSlash(rel))
	if !strings.HasPrefix(relLower, "garmin/") && relLower != "garmin" {
		return "", errors.New("file must be located inside GARMIN directory on device")
	}

	// Check extension
	if !strings.EqualFold(filepath.Ext(targetPath), ".img") {
		return "", ErrNotAnImgFile
	}

	// Guardrail: must not be system map or have Garmin signature files
	if IsSystemMap(filepath.Base(targetPath)) {
		return "", ErrSystemMapProtected
	}
	if hasSig, _ := HasGarminSignatureFiles(targetPath); hasSig {
		return "", ErrSystemMapProtected
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		return "", fmt.Errorf("file not found on device: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("target is a directory, not a file")
	}

	return targetPath, nil
}
