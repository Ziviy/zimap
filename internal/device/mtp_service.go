package device

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// MTPStorageInfo represents MTP storage partition metrics.
type MTPStorageInfo struct {
	StorageID     uint32 `json:"storageId"`
	Description   string `json:"description"`
	TotalBytes    uint64 `json:"totalBytes"`
	FreeBytes     uint64 `json:"freeBytes"`
	MaxObjectSize uint64 `json:"maxObjectSize,omitempty"`
}

// MTPDeviceInfo represents detected MTP hardware attributes.
type MTPDeviceInfo struct {
	Connected     bool           `json:"connected"`
	VendorID      string         `json:"vendorId,omitempty"`
	ProductID     string         `json:"productId,omitempty"`
	FriendlyName  string         `json:"friendlyName,omitempty"`
	Manufacturer  string         `json:"manufacturer,omitempty"`
	Model         string         `json:"model,omitempty"`
	SerialNumber  string         `json:"serialNumber,omitempty"`
	BatteryLevel  int            `json:"batteryLevel,omitempty"`
	DeviceVersion string         `json:"deviceVersion,omitempty"`
	Storage       MTPStorageInfo `json:"storage"`
}

// MTPDetectResult represents the JSON output from `zimap-mtp detect`.
type MTPDetectResult struct {
	Success   bool           `json:"success"`
	Device    *MTPDeviceInfo `json:"device,omitempty"`
	Conflicts []string       `json:"conflicts,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// MTPMapItem represents a map file entry returned by `zimap-mtp list-maps`.
type MTPMapItem struct {
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

// MTPListMapsResult represents the JSON output from `zimap-mtp list-maps`.
type MTPListMapsResult struct {
	Success bool         `json:"success"`
	Maps    []MTPMapItem `json:"maps"`
	Error   string       `json:"error,omitempty"`
}

// MTPConflictResult represents the JSON output from `zimap-mtp check-conflicts`.
type MTPConflictResult struct {
	Success   bool     `json:"success"`
	Conflicts []string `json:"conflicts"`
}

// MTPSimpleResult represents generic success/error output from sidecar commands.
type MTPSimpleResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// MTPService coordinates execution and event streaming with the `zimap-mtp` helper process.
type MTPService struct {
	customBin string
	mu        sync.Mutex
	activeCmd *exec.Cmd
}

// NewMTPService instantiates an MTPService.
func NewMTPService() *MTPService {
	return &MTPService{}
}

// SetBinaryPath sets a custom location for the zimap-mtp binary (used in tests or overrides).
func (s *MTPService) SetBinaryPath(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.customBin = path
}

// FindBinaryPath locates the zimap-mtp binary on the system.
func (s *MTPService) FindBinaryPath() (string, error) {
	s.mu.Lock()
	custom := s.customBin
	s.mu.Unlock()

	if custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
	}

	// 1. Check environment variable override
	if envPath := os.Getenv("ZIMAP_MTP_BIN"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 2. Check next to running executable
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		binName := "zimap-mtp"
		if runtime.GOOS == "windows" {
			binName = "zimap-mtp.exe"
		}

		// Next to executable
		c1 := filepath.Join(exeDir, binName)
		if _, err := os.Stat(c1); err == nil {
			return c1, nil
		}

		// In macOS .app bundle: Contents/MacOS/zimap-mtp or Contents/Helpers/zimap-mtp
		c2 := filepath.Join(exeDir, "..", "MacOS", binName)
		if _, err := os.Stat(c2); err == nil {
			return c2, nil
		}
		c3 := filepath.Join(exeDir, "..", "Helpers", binName)
		if _, err := os.Stat(c3); err == nil {
			return c3, nil
		}
	}

	// 3. Check development / workspace tools/mtp directory
	devCandidates := []string{
		filepath.Join("tools", "mtp", "zimap-mtp"),
		filepath.Join("..", "tools", "mtp", "zimap-mtp"),
		filepath.Join("..", "..", "tools", "mtp", "zimap-mtp"),
	}
	if runtime.GOOS == "windows" {
		for i, cand := range devCandidates {
			devCandidates[i] = cand + ".exe"
		}
	}
	for _, cand := range devCandidates {
		if abs, err := filepath.Abs(cand); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}

	// 4. Check PATH
	if p, err := exec.LookPath("zimap-mtp"); err == nil {
		return p, nil
	}

	return "", errors.New("zimap-mtp binary not found")
}

func (s *MTPService) resolveBinary() (string, error) {
	return s.FindBinaryPath()
}

// IsAvailable checks if the zimap-mtp binary is available to run.
func (s *MTPService) IsAvailable() bool {
	_, err := s.resolveBinary()
	return err == nil
}

// DetectDevice calls `zimap-mtp detect` and returns hardware and storage details.
func (s *MTPService) DetectDevice(ctx context.Context) (*MTPDetectResult, error) {
	bin, err := s.resolveBinary()
	if err != nil {
		return nil, err
	}

	args := []string{"detect"}
	if os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1" {
		args = append(args, "--mock")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			var res MTPDetectResult
			if jsonErr := json.Unmarshal(out, &res); jsonErr == nil {
				return &res, nil
			}
		}
		return nil, fmt.Errorf("zimap-mtp detect failed: %w", err)
	}

	var res MTPDetectResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("failed to parse detect output: %w", err)
	}

	return &res, nil
}

// ListMaps queries custom and factory maps inside /GARMIN via `zimap-mtp list-maps`.
func (s *MTPService) ListMaps(ctx context.Context, storageID uint32) ([]DeviceMap, error) {
	bin, err := s.resolveBinary()
	if err != nil {
		return nil, err
	}

	args := []string{"list-maps", fmt.Sprintf("--storage-id=%d", storageID)}
	if os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1" {
		args = append(args, "--mock")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("zimap-mtp list-maps failed: %w", err)
	}

	var res MTPListMapsResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("failed to parse list-maps output: %w", err)
	}

	if !res.Success {
		return nil, fmt.Errorf("list-maps failed: %s", res.Error)
	}

	var result []DeviceMap
	for _, m := range res.Maps {
		result = append(result, DeviceMap{
			FileName:       m.FileName,
			FilePath:       filepath.Join(m.RelativeDir, m.FileName),
			InternalName:   m.InternalTitle,
			SizeBytes:      m.Size,
			SizeFormatted:  FormatStorageBytes(uint64(m.Size)),
			ModTime:        m.Modified,
			IsCustom:       m.IsCustom,
			IsPreinstalled: m.IsPreinstalled,
			SignatureType:  m.SignatureType,
			RelativeDir:    m.RelativeDir,
			ObjectID:       m.ObjectID,
		})
	}

	return result, nil
}

// UploadMap streams a map file to the Garmin MTP device, emitting real-time progress events.
func (s *MTPService) UploadMap(ctx context.Context, storageID uint32, srcPath string, targetDir string, progressCb func(SyncProgressEvent)) error {
	bin, err := s.resolveBinary()
	if err != nil {
		return err
	}

	if targetDir == "" {
		targetDir = "GARMIN"
	}

	args := []string{
		"upload",
		"--source=" + srcPath,
		fmt.Sprintf("--storage-id=%d", storageID),
		"--target-dir=" + targetDir,
	}
	if os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1" {
		args = append(args, "--mock")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	s.mu.Lock()
	s.activeCmd = cmd
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeCmd = nil
		s.mu.Unlock()
	}()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start zimap-mtp: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	baseName := filepath.Base(srcPath)
	var uploadErr error

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		switch evt["event"] {
		case "progress":
			sentBytes := int64(evt["sentBytes"].(float64))
			totalBytes := int64(evt["totalBytes"].(float64))
			percent := int(evt["percent"].(float64))
			speedBps := int64(evt["speedBps"].(float64))
			etaSeconds := int(evt["etaSeconds"].(float64))

			if progressCb != nil {
				progressCb(SyncProgressEvent{
					MapID:          baseName,
					FileName:       baseName,
					BytesCopied:    sentBytes,
					TotalBytes:     totalBytes,
					Percentage:     percent,
					SpeedBytesSec:  speedBps,
					SpeedFormatted: FormatStorageBytes(uint64(speedBps)) + "/s",
					ETASeconds:     etaSeconds,
				})
			}
		case "done":
			// Finished successfully
		case "error":
			uploadErr = fmt.Errorf("%v", evt["error"])
		}
	}

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if uploadErr != nil {
		return uploadErr
	}
	if waitErr != nil {
		errMsg := strings.TrimSpace(stderrBuf.String())
		if errMsg != "" {
			return fmt.Errorf("zimap-mtp upload failed: %s", errMsg)
		}
		return fmt.Errorf("zimap-mtp exited with error: %w", waitErr)
	}

	return nil
}

// DeleteMap deletes a map file on the MTP device by objectId or filename.
func (s *MTPService) DeleteMap(ctx context.Context, storageID uint32, objectID uint32, fileName string) error {
	bin, err := s.resolveBinary()
	if err != nil {
		return err
	}

	args := []string{"delete", fmt.Sprintf("--storage-id=%d", storageID)}
	if objectID > 0 {
		args = append(args, fmt.Sprintf("--object-id=%d", objectID))
	}
	if fileName != "" {
		args = append(args, "--file-name="+filepath.Base(fileName))
	}
	if os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1" {
		args = append(args, "--mock")
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.Output()
	if err != nil {
		if len(out) > 0 {
			var res MTPSimpleResult
			if jsonErr := json.Unmarshal(out, &res); jsonErr == nil && res.Error != "" {
				return errors.New(res.Error)
			}
		}
		return fmt.Errorf("zimap-mtp delete failed: %w", err)
	}

	var res MTPSimpleResult
	if err := json.Unmarshal(out, &res); err != nil {
		return fmt.Errorf("failed to parse delete response: %w", err)
	}
	if !res.Success {
		return fmt.Errorf("delete failed: %s", res.Error)
	}

	return nil
}

// CheckConflicts checks if processes such as Garmin Express or Android File Transfer are running.
func (s *MTPService) CheckConflicts(ctx context.Context) ([]string, error) {
	bin, err := s.resolveBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, bin, "check-conflicts")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var res MTPConflictResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, err
	}

	return res.Conflicts, nil
}

// CancelTransfer aborts any ongoing file transfer.
func (s *MTPService) CancelTransfer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeCmd != nil && s.activeCmd.Process != nil {
		_ = s.activeCmd.Process.Signal(os.Interrupt)
	}
}
