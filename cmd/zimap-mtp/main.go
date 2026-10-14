package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ganeshrvel/go-mtpx"

	"zimap/internal/device"
)

// Known processes that lock the USB MTP interface on macOS/Windows/Linux
var conflictingProcessKeywords = []string{
	"Garmin Express",
	"Garmin Express Service",
	"Android File Transfer",
	"Android File Transfer Agent",
	"OpenMTP",
}

type DeviceStorageJSON struct {
	StorageID     uint32 `json:"storageId"`
	Description   string `json:"description"`
	TotalBytes    uint64 `json:"totalBytes"`
	FreeBytes     uint64 `json:"freeBytes"`
	MaxObjectSize uint64 `json:"maxObjectSize,omitempty"`
}

type DeviceInfoJSON struct {
	Connected     bool              `json:"connected"`
	VendorID      string            `json:"vendorId,omitempty"`
	ProductID     string            `json:"productId,omitempty"`
	FriendlyName  string            `json:"friendlyName,omitempty"`
	Manufacturer  string            `json:"manufacturer,omitempty"`
	Model         string            `json:"model,omitempty"`
	SerialNumber  string            `json:"serialNumber,omitempty"`
	BatteryLevel  int               `json:"batteryLevel,omitempty"`
	DeviceVersion string            `json:"deviceVersion,omitempty"`
	Storage       DeviceStorageJSON `json:"storage"`
}

type DetectResponse struct {
	Success   bool            `json:"success"`
	Device    *DeviceInfoJSON `json:"device,omitempty"`
	Conflicts []string        `json:"conflicts,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type MapItemJSON struct {
	ObjectID       uint32    `json:"objectId"`
	FileName       string    `json:"fileName"`
	InternalTitle  string    `json:"internalTitle"`
	Size           int64     `json:"size"`
	Modified       time.Time `json:"modified"`
	IsCustom       bool      `json:"isCustom"`
	IsPreinstalled bool      `json:"isPreinstalled"`
	SignatureType  string    `json:"signatureType,omitempty"`
	RelativeDir    string    `json:"relativeDir"`
}

type ListMapsResponse struct {
	Success bool          `json:"success"`
	Maps    []MapItemJSON `json:"maps"`
	Error   string        `json:"error,omitempty"`
}

type ConflictResponse struct {
	Success   bool     `json:"success"`
	Conflicts []string `json:"conflicts"`
}

type SimpleResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "detect":
		handleDetect(args)
	case "list-maps":
		handleListMaps(args)
	case "upload":
		handleUpload(args)
	case "delete":
		handleDelete(args)
	case "check-conflicts":
		handleCheckConflicts(args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `zimap-mtp — Standalone Garmin MTP sidecar utility

Usage:
  zimap-mtp detect [--mock]
  zimap-mtp list-maps [--storage-id=N] [--mock]
  zimap-mtp upload --source=<path> [--storage-id=N] [--target-dir=GARMIN] [--mock]
  zimap-mtp delete --storage-id=N [--object-id=N] [--file-name=NAME] [--mock]
  zimap-mtp check-conflicts
`)
}

func isMockMode(flags *flag.FlagSet, mockFlag *bool) bool {
	if mockFlag != nil && *mockFlag {
		return true
	}
	return os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1"
}

// ==========================================
// 1. DETECT
// ==========================================

func handleDetect(args []string) {
	fs := flag.NewFlagSet("detect", flag.ExitOnError)
	mock := fs.Bool("mock", false, "Simulate detected Garmin MTP watch")
	_ = fs.Parse(args)

	conflicts := detectConflicts()

	if isMockMode(fs, mock) {
		resp := DetectResponse{
			Success: true,
			Device: &DeviceInfoJSON{
				Connected:     true,
				VendorID:      "091e",
				ProductID:     "4c43",
				FriendlyName:  "fenix 7 Sapphire Solar",
				Manufacturer:  "Garmin",
				Model:         "fenix 7",
				SerialNumber:  "3412345678",
				BatteryLevel:  88,
				DeviceVersion: "18.16",
				Storage: DeviceStorageJSON{
					StorageID:     65537,
					Description:   "Primary",
					TotalBytes:    32212254720,
					FreeBytes:     14285714285,
					MaxObjectSize: 4294967295,
				},
			},
			Conflicts: conflicts,
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	dev, err := mtpx.Initialize(mtpx.Init{DebugMode: false})
	if err != nil {
		// Device not found or port locked
		resp := DetectResponse{
			Success:   true,
			Conflicts: conflicts,
		}
		if len(conflicts) > 0 {
			resp.Success = false
			resp.Error = fmt.Sprintf("USB port is locked by conflicting application (%s)", strings.Join(conflicts, ", "))
		} else {
			resp.Device = &DeviceInfoJSON{
				Connected: false,
			}
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}
	defer mtpx.Dispose(dev)

	devInfo, err := mtpx.FetchDeviceInfo(dev)
	if err != nil {
		resp := DetectResponse{
			Success:   false,
			Error:     fmt.Sprintf("failed to fetch device info: %v", err),
			Conflicts: conflicts,
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	storages, err := mtpx.FetchStorages(dev)
	if err != nil || len(storages) == 0 {
		resp := DetectResponse{
			Success:   false,
			Error:     "no storage found on MTP device",
			Conflicts: conflicts,
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	// Pick primary or first storage with largest capacity
	var primaryStorage mtpx.StorageData = storages[0]
	for _, s := range storages {
		if s.Info.MaxCapability > primaryStorage.Info.MaxCapability {
			primaryStorage = s
		}
	}

	usbInfo, _ := dev.GetUsbInfo()
	vendorID := ""
	productID := ""
	if usbInfo != nil {
		vendorID = fmt.Sprintf("%04x", usbInfo.IdVendor)
		productID = fmt.Sprintf("%04x", usbInfo.IdProduct)
	}

	modelName := strings.TrimSpace(devInfo.Model)
	if modelName == "" {
		modelName = "Garmin Watch"
	}

	friendlyName := modelName
	if !strings.HasPrefix(strings.ToLower(friendlyName), "garmin") && strings.EqualFold(devInfo.Manufacturer, "garmin") {
		friendlyName = "Garmin " + friendlyName
	}

	resp := DetectResponse{
		Success: true,
		Device: &DeviceInfoJSON{
			Connected:     true,
			VendorID:      vendorID,
			ProductID:     productID,
			FriendlyName:  friendlyName,
			Manufacturer:  devInfo.Manufacturer,
			Model:         modelName,
			SerialNumber:  devInfo.SerialNumber,
			DeviceVersion: devInfo.DeviceVersion,
			Storage: DeviceStorageJSON{
				StorageID:     primaryStorage.Sid,
				Description:   primaryStorage.Info.StorageDescription,
				TotalBytes:    primaryStorage.Info.MaxCapability,
				FreeBytes:     primaryStorage.Info.FreeSpaceInBytes,
				MaxObjectSize: 4294967295,
			},
		},
		Conflicts: conflicts,
	}

	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

// ==========================================
// 2. LIST MAPS
// ==========================================

func handleListMaps(args []string) {
	fs := flag.NewFlagSet("list-maps", flag.ExitOnError)
	storageID := fs.Uint("storage-id", 65537, "MTP storage ID")
	mock := fs.Bool("mock", false, "Simulate map listing")
	_ = fs.Parse(args)

	if isMockMode(fs, mock) {
		resp := ListMapsResponse{
			Success: true,
			Maps: []MapItemJSON{
				{
					ObjectID:      1042,
					FileName:      "russia-northwestern.img",
					InternalTitle: "OSM Russia Northwestern",
					Size:          734003200,
					Modified:      time.Now().Add(-48 * time.Hour),
					IsCustom:      true,
					RelativeDir:   "GARMIN",
				},
				{
					ObjectID:      1043,
					FileName:      "finland.img",
					InternalTitle: "OSM Finland Topo",
					Size:          456123400,
					Modified:      time.Now().Add(-120 * time.Hour),
					IsCustom:      true,
					RelativeDir:   "GARMIN",
				},
			},
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}

	dev, err := mtpx.Initialize(mtpx.Init{DebugMode: false})
	if err != nil {
		resp := ListMapsResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to initialize MTP: %v", err),
		}
		_ = json.NewEncoder(os.Stdout).Encode(resp)
		return
	}
	defer mtpx.Dispose(dev)

	var maps []MapItemJSON
	seenNames := make(map[string]bool)

	// Scan /GARMIN directory
	scanDir := func(dirPath string, relDir string) {
		sigFiles := make(map[string]string)
		var imgFiles []*mtpx.FileInfo
		imgObjectIDs := make(map[string]uint32)

		_, _, _, _ = mtpx.Walk(dev, uint32(*storageID), dirPath, false, false, true, func(objectId uint32, fi *mtpx.FileInfo, err error) error {
			if err != nil || fi == nil || fi.IsDir {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(fi.Name))
			baseNoExt := strings.ToLower(strings.TrimSuffix(fi.Name, filepath.Ext(fi.Name)))
			if ext == ".gma" {
				sigFiles[baseNoExt] = "gma_signature"
			} else if ext == ".unl" {
				sigFiles[baseNoExt] = "unl_license"
			} else if ext == ".img" {
				imgFiles = append(imgFiles, fi)
				imgObjectIDs[fi.Name] = objectId
			}
			return nil
		})

		for _, fi := range imgFiles {
			fileName := fi.Name
			objectId := imgObjectIDs[fileName]
			if seenNames[fileName] {
				continue
			}
			seenNames[fileName] = true

			baseNoExt := strings.ToLower(strings.TrimSuffix(fileName, filepath.Ext(fileName)))
			sigType := sigFiles[baseNoExt]
			isPreinstalled := sigType != ""

			var internalTitle string
			// Try reading internal title
			buf := new(bytes.Buffer)
			if err := dev.GetPartialObject(objectId, buf, 0, 150); err == nil {
				internalTitle, _ = device.ParseMapInternalNameFromReader(buf)
			}

			if !isPreinstalled {
				isPreinstalled, sigType = device.DetectPreinstalledMap(fileName, "", internalTitle)
			}

			if internalTitle == "" || isPreinstalled {
				internalTitle = device.ResolveGarminMapTitle(fileName, internalTitle)
			}

			maps = append(maps, MapItemJSON{
				ObjectID:       objectId,
				FileName:       fileName,
				InternalTitle:  internalTitle,
				Size:           fi.Size,
				Modified:       fi.ModTime,
				IsCustom:       !isPreinstalled,
				IsPreinstalled: isPreinstalled,
				SignatureType:  sigType,
				RelativeDir:    relDir,
			})
		}
	}

	scanDir("/GARMIN", "GARMIN")
	scanDir("/GARMIN/Map", "GARMIN/Map")

	resp := ListMapsResponse{
		Success: true,
		Maps:    maps,
	}
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

// ==========================================
// 3. UPLOAD
// ==========================================

func handleUpload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	source := fs.String("source", "", "Source map file path (.img)")
	storageID := fs.Uint("storage-id", 65537, "MTP storage ID")
	targetDir := fs.String("target-dir", "GARMIN", "Target directory on Garmin device")
	mock := fs.Bool("mock", false, "Simulate file upload")
	_ = fs.Parse(args)

	if *source == "" {
		emitUploadError("source path is required")
		os.Exit(1)
	}

	absSource, err := filepath.Abs(*source)
	if err == nil {
		*source = absSource
	}

	fileInfo, err := os.Stat(*source)
	if err != nil {
		emitUploadError(fmt.Sprintf("source file not found: %v", err))
		os.Exit(1)
	}

	totalBytes := fileInfo.Size()

	// Handle cancellation gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var canceled atomic.Bool

	go func() {
		<-sigChan
		canceled.Store(true)
	}()

	emitUploadEvent(map[string]interface{}{
		"event":      "start",
		"totalBytes": totalBytes,
	})

	startTime := time.Now()

	if isMockMode(fs, mock) {
		// Mock streaming simulation
		steps := 10
		chunk := totalBytes / int64(steps)
		if chunk == 0 {
			chunk = 1
		}

		for i := 1; i <= steps; i++ {
			if canceled.Load() {
				emitUploadError("transfer canceled by user")
				os.Exit(1)
			}
			time.Sleep(100 * time.Millisecond)

			sent := int64(i) * chunk
			if sent > totalBytes || i == steps {
				sent = totalBytes
			}
			pct := int((sent * 100) / totalBytes)
			speedBps := int64(30 * 1024 * 1024) // 30 MB/s simulated
			var eta int
			if sent < totalBytes {
				eta = int((totalBytes - sent) / speedBps)
			}

			emitUploadEvent(map[string]interface{}{
				"event":      "progress",
				"sentBytes":  sent,
				"totalBytes": totalBytes,
				"percent":    pct,
				"speedBps":   speedBps,
				"etaSeconds": eta,
			})
		}

		elapsed := time.Since(startTime).Seconds()
		emitUploadEvent(map[string]interface{}{
			"event":          "done",
			"objectId":       9999,
			"totalBytes":     totalBytes,
			"elapsedSeconds": elapsed,
		})
		return
	}

	// Real MTP Upload
	dev, err := mtpx.Initialize(mtpx.Init{DebugMode: false})
	if err != nil {
		emitUploadError(fmt.Sprintf("failed to initialize MTP: %v", err))
		os.Exit(1)
	}
	defer mtpx.Dispose(dev)

	cleanTarget := *targetDir
	if !strings.HasPrefix(cleanTarget, "/") {
		cleanTarget = "/" + cleanTarget
	}

	progressCb := func(pInfo *mtpx.ProgressInfo, cbErr error) error {
		if cbErr != nil {
			return cbErr
		}
		if canceled.Load() {
			return fmt.Errorf("transfer canceled by user")
		}
		if pInfo == nil || pInfo.ActiveFileSize == nil {
			return nil
		}

		sent := pInfo.ActiveFileSize.Sent
		total := pInfo.ActiveFileSize.Total
		if total == 0 {
			total = totalBytes
		}

		pct := int(pInfo.ActiveFileSize.Progress)
		speedBps := int64(pInfo.Speed * 1024 * 1024)
		var eta int
		if speedBps > 0 && sent < total {
			eta = int((total - sent) / speedBps)
		}

		emitUploadEvent(map[string]interface{}{
			"event":      "progress",
			"sentBytes":  sent,
			"totalBytes": total,
			"percent":    pct,
			"speedBps":   speedBps,
			"etaSeconds": eta,
		})

		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			emitUploadError(fmt.Sprintf("internal MTP panic: %v", r))
			os.Exit(1)
		}
	}()

	preprocessCb := func(fi *os.FileInfo, fullPath string, err error) error {
		return nil
	}

	destObjId, _, _, err := mtpx.UploadFiles(
		dev,
		uint32(*storageID),
		[]string{*source},
		cleanTarget,
		true,
		preprocessCb,
		progressCb,
	)

	if err != nil {
		emitUploadError(err.Error())
		os.Exit(1)
	}

	elapsed := time.Since(startTime).Seconds()
	emitUploadEvent(map[string]interface{}{
		"event":          "done",
		"objectId":       destObjId,
		"totalBytes":     totalBytes,
		"elapsedSeconds": elapsed,
	})
}

func emitUploadEvent(evt map[string]interface{}) {
	_ = json.NewEncoder(os.Stdout).Encode(evt)
	_ = os.Stdout.Sync()
}

func emitUploadError(msg string) {
	_ = json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
		"event": "error",
		"error": msg,
	})
	_ = os.Stdout.Sync()
}

// ==========================================
// 4. DELETE
// ==========================================

func handleDelete(args []string) {
	fs := flag.NewFlagSet("delete", flag.ExitOnError)
	storageID := fs.Uint("storage-id", 65537, "MTP storage ID")
	objectID := fs.Uint("object-id", 0, "MTP Object ID to delete")
	fileName := fs.String("file-name", "", "File name to delete (e.g. map.img)")
	mock := fs.Bool("mock", false, "Simulate file deletion")
	_ = fs.Parse(args)

	if *fileName != "" && device.IsSystemMap(*fileName) {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{
			Success: false,
			Error:   "cannot delete system map: file is protected",
		})
		os.Exit(1)
	}

	if isMockMode(fs, mock) {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{Success: true})
		return
	}

	dev, err := mtpx.Initialize(mtpx.Init{DebugMode: false})
	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to initialize MTP: %v", err),
		})
		os.Exit(1)
	}
	defer mtpx.Dispose(dev)

	objToDelete := uint32(*objectID)

	// If no objectId provided, look up by filename in /GARMIN
	if objToDelete == 0 && *fileName != "" {
		targetFile := *fileName
		_, _, _, _ = mtpx.Walk(dev, uint32(*storageID), "/GARMIN", true, false, true, func(objId uint32, fi *mtpx.FileInfo, err error) error {
			if err == nil && fi != nil && strings.EqualFold(fi.Name, targetFile) {
				objToDelete = objId
			}
			return nil
		})
	}

	if objToDelete == 0 {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{
			Success: false,
			Error:   "object not found on device",
		})
		os.Exit(1)
	}

	// Verify target object is not a protected system map
	var isProtected bool
	_, _, _, _ = mtpx.Walk(dev, uint32(*storageID), "/GARMIN", true, false, true, func(objId uint32, fi *mtpx.FileInfo, err error) error {
		if err == nil && fi != nil && objId == objToDelete {
			if device.IsSystemMap(fi.Name) {
				isProtected = true
			}
		}
		return nil
	})
	if isProtected {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{
			Success: false,
			Error:   "cannot delete system map: file is protected",
		})
		os.Exit(1)
	}

	// Delete file
	err = dev.DeleteObject(objToDelete)
	if err != nil {
		// Fallback to DeleteFile helper
		err = mtpx.DeleteFile(dev, uint32(*storageID), []mtpx.FileProp{{ObjectId: objToDelete}})
	}

	if err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to delete MTP object: %v", err),
		})
		os.Exit(1)
	}

	_ = json.NewEncoder(os.Stdout).Encode(SimpleResponse{Success: true})
}

// ==========================================
// 5. CHECK CONFLICTS
// ==========================================

func handleCheckConflicts(args []string) {
	conflicts := detectConflicts()
	_ = json.NewEncoder(os.Stdout).Encode(ConflictResponse{
		Success:   true,
		Conflicts: conflicts,
	})
}

func detectConflicts() []string {
	var found []string
	switch runtime.GOOS {
	case "darwin", "linux":
		out, err := exec.Command("ps", "-ax", "-c", "-o", "comm=").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, kw := range conflictingProcessKeywords {
				kwLower := strings.ToLower(kw)
				for _, line := range lines {
					if strings.Contains(strings.ToLower(strings.TrimSpace(line)), kwLower) {
						found = appendIfNotPresent(found, kw)
						break
					}
				}
			}
		}
	case "windows":
		out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
		if err == nil {
			str := strings.ToLower(string(out))
			for _, kw := range conflictingProcessKeywords {
				if strings.Contains(str, strings.ToLower(kw)) {
					found = appendIfNotPresent(found, kw)
				}
			}
		}
	}
	return found
}

func appendIfNotPresent(list []string, item string) []string {
	for _, it := range list {
		if strings.EqualFold(it, item) {
			return list
		}
	}
	return append(list, item)
}
