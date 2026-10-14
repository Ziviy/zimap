package device

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrSyncInProgress = errors.New("a map sync is already in progress")
	ErrGarminDirNotFound = errors.New("GARMIN directory not found on device volume")
)

// ResolveDestinationPath safely locates where a map file should be written on a Garmin device.
// If the map already exists on the device, its existing path is preserved.
// If it is a new map, it targets GARMIN/Map/ if that directory exists, or GARMIN/ otherwise.
func ResolveDestinationPath(mountPath, fileName string) (string, error) {
	if mountPath == "" || fileName == "" {
		return "", errors.New("mount path and file name must not be empty")
	}

	absMount, err := filepath.Abs(filepath.Clean(mountPath))
	if err != nil {
		return "", fmt.Errorf("invalid mount path: %w", err)
	}

	baseName := filepath.Base(fileName)
	if !strings.EqualFold(filepath.Ext(baseName), ".img") {
		return "", ErrNotAnImgFile
	}

	if IsSystemMap(baseName) {
		return "", ErrSystemMapProtected
	}

	garminDir := filepath.Join(absMount, "GARMIN")
	mapDir := filepath.Join(garminDir, "Map")

	// 1. If file already exists on device, keep its exact current location
	candidateMap := filepath.Join(mapDir, baseName)
	if _, err := os.Stat(candidateMap); err == nil {
		return candidateMap, nil
	}

	candidateGarmin := filepath.Join(garminDir, baseName)
	if _, err := os.Stat(candidateGarmin); err == nil {
		return candidateGarmin, nil
	}

	// 2. New map: prefer GARMIN/Map if present, else GARMIN
	if info, err := os.Stat(mapDir); err == nil && info.IsDir() {
		return candidateMap, nil
	}

	if info, err := os.Stat(garminDir); err == nil && info.IsDir() {
		return candidateGarmin, nil
	}

	return "", ErrGarminDirNotFound
}

// SyncFile safely streams a local map file to a Garmin device with live progress,
// pre-flight space check, atomic rename, hardware cache sync, and timestamp preservation.
func SyncFile(ctx context.Context, srcPath, destPath string, progressFn func(SyncProgressEvent)) error {
	srcStat, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("cannot read local map file: %w", err)
	}
	if srcStat.IsDir() {
		return fmt.Errorf("source is a directory, expected .img file: %s", srcPath)
	}

	totalSize := srcStat.Size()
	srcModTime := srcStat.ModTime()
	baseName := filepath.Base(destPath)

	// Pre-flight check: ensure target directory exists
	targetDir := filepath.Dir(destPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Pre-flight space check
	if _, free, err := getDiskSpace(targetDir); err == nil && free > 0 {
		needed := uint64(totalSize + DeviceSafetyMarginBytes)
		if free < needed {
			return fmt.Errorf("insufficient storage on device: required %s, available %s",
				FormatStorageBytes(needed), FormatStorageBytes(free))
		}
	}

	tempDest := destPath + ".tmp-zimap"
	_ = os.Remove(tempDest)

	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source map: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(tempDest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary file on device: %w", err)
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = destFile.Close()
			_ = os.Remove(tempDest)
		}
	}()

	// 128 KB transfer chunk buffer
	buf := make([]byte, 128*1024)
	var bytesCopied int64
	lastEmitTime := time.Now()
	lastCopied := int64(0)
	var currentSpeed int64

	if progressFn != nil {
		progressFn(SyncProgressEvent{
			MapID:          baseName,
			FileName:       baseName,
			BytesCopied:    0,
			TotalBytes:     totalSize,
			Percentage:     0,
			SpeedBytesSec:  0,
			SpeedFormatted: "0 B/s",
			ETASeconds:     0,
		})
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := srcFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write error to device: %w", writeErr)
			}
			bytesCopied += int64(n)

			now := time.Now()
			timeDiff := now.Sub(lastEmitTime)
			if progressFn != nil && (timeDiff >= 150*time.Millisecond || bytesCopied == totalSize) {
				pct := 0
				if totalSize > 0 {
					pct = int((float64(bytesCopied) / float64(totalSize)) * 100)
					if pct > 100 {
						pct = 100
					}
				}

				if timeDiff > 0 {
					deltaBytes := bytesCopied - lastCopied
					instantSpeed := int64(float64(deltaBytes) / timeDiff.Seconds())
					if currentSpeed == 0 {
						currentSpeed = instantSpeed
					} else {
						// Exponential moving average for smooth display
						currentSpeed = int64(0.6*float64(currentSpeed) + 0.4*float64(instantSpeed))
					}
				}

				eta := 0
				if currentSpeed > 0 && totalSize > bytesCopied {
					eta = int((totalSize - bytesCopied) / currentSpeed)
				}

				progressFn(SyncProgressEvent{
					MapID:          baseName,
					FileName:       baseName,
					BytesCopied:    bytesCopied,
					TotalBytes:     totalSize,
					Percentage:     pct,
					SpeedBytesSec:  currentSpeed,
					SpeedFormatted: FormatStorageBytes(uint64(currentSpeed)) + "/s",
					ETASeconds:     eta,
				})

				lastEmitTime = now
				lastCopied = bytesCopied
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("read error from source map: %w", readErr)
		}
	}

	// Flush OS and storage controller hardware buffers
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync device storage buffer: %w", err)
	}

	if err := destFile.Close(); err != nil {
		return fmt.Errorf("failed to finalize device file: %w", err)
	}

	// Atomic replace
	if err := os.Rename(tempDest, destPath); err != nil {
		// On Windows or FAT32, replace by unlinking first if Rename returns error
		_ = os.Remove(destPath)
		if err2 := os.Rename(tempDest, destPath); err2 != nil {
			return fmt.Errorf("failed to replace destination map file: %w", err2)
		}
	}

	// Synchronize modification timestamp to match computer's build time
	_ = os.Chtimes(destPath, srcModTime, srcModTime)

	cleanup = false

	if progressFn != nil {
		progressFn(SyncProgressEvent{
			MapID:          baseName,
			FileName:       baseName,
			BytesCopied:    totalSize,
			TotalBytes:     totalSize,
			Percentage:     100,
			SpeedBytesSec:  0,
			SpeedFormatted: "0 B/s",
			ETASeconds:     0,
		})
	}

	return nil
}

// SyncManager coordinates concurrent sync operations and cancellation.
type SyncManager struct {
	mu        sync.Mutex
	isSyncing bool
	cancel    context.CancelFunc
}

// NewSyncManager creates a new SyncManager.
func NewSyncManager() *SyncManager {
	return &SyncManager{}
}

// IsSyncing returns whether a sync operation is currently active.
func (sm *SyncManager) IsSyncing() bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return sm.isSyncing
}

// Cancel cancels the active sync operation if running.
func (sm *SyncManager) Cancel() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.cancel != nil {
		sm.cancel()
	}
}

// Sync executes a file sync operation, guarding against concurrent executions.
func (sm *SyncManager) Sync(
	parentCtx context.Context,
	srcPath, destPath string,
	progressFn func(SyncProgressEvent),
) error {
	sm.mu.Lock()
	if sm.isSyncing {
		sm.mu.Unlock()
		return ErrSyncInProgress
	}
	ctx, cancel := context.WithCancel(parentCtx)
	sm.cancel = cancel
	sm.isSyncing = true
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		sm.isSyncing = false
		sm.cancel = nil
		sm.mu.Unlock()
	}()

	return SyncFile(ctx, srcPath, destPath, progressFn)
}

// StartSync initiates a file sync in a background goroutine, synchronously setting IsSyncing
// to true so callers immediately see the manager as busy.
func (sm *SyncManager) StartSync(
	parentCtx context.Context,
	srcPath, destPath string,
	progressFn func(SyncProgressEvent),
	onComplete func(err error),
) error {
	sm.mu.Lock()
	if sm.isSyncing {
		sm.mu.Unlock()
		return ErrSyncInProgress
	}
	ctx, cancel := context.WithCancel(parentCtx)
	sm.cancel = cancel
	sm.isSyncing = true
	sm.mu.Unlock()

	go func() {
		err := SyncFile(ctx, srcPath, destPath, progressFn)
		sm.mu.Lock()
		sm.isSyncing = false
		sm.cancel = nil
		sm.mu.Unlock()
		if onComplete != nil {
			onComplete(err)
		}
	}()

	return nil
}

// SyncMTP executes an MTP map upload operation, guarding against concurrent executions.
func (sm *SyncManager) SyncMTP(
	parentCtx context.Context,
	mtpSvc *MTPService,
	storageID uint32,
	srcPath, targetDir string,
	progressFn func(SyncProgressEvent),
) error {
	sm.mu.Lock()
	if sm.isSyncing {
		sm.mu.Unlock()
		return ErrSyncInProgress
	}
	ctx, cancel := context.WithCancel(parentCtx)
	sm.cancel = func() {
		cancel()
		if mtpSvc != nil {
			mtpSvc.CancelTransfer()
		}
	}
	sm.isSyncing = true
	sm.mu.Unlock()

	defer func() {
		sm.mu.Lock()
		sm.isSyncing = false
		sm.cancel = nil
		sm.mu.Unlock()
	}()

	return mtpSvc.UploadMap(ctx, storageID, srcPath, targetDir, progressFn)
}

// StartSyncMTP initiates an MTP map upload in a background goroutine.
func (sm *SyncManager) StartSyncMTP(
	parentCtx context.Context,
	mtpSvc *MTPService,
	storageID uint32,
	srcPath, targetDir string,
	progressFn func(SyncProgressEvent),
	onComplete func(err error),
) error {
	sm.mu.Lock()
	if sm.isSyncing {
		sm.mu.Unlock()
		return ErrSyncInProgress
	}
	ctx, cancel := context.WithCancel(parentCtx)
	sm.cancel = func() {
		cancel()
		if mtpSvc != nil {
			mtpSvc.CancelTransfer()
		}
	}
	sm.isSyncing = true
	sm.mu.Unlock()

	go func() {
		err := mtpSvc.UploadMap(ctx, storageID, srcPath, targetDir, progressFn)
		sm.mu.Lock()
		sm.isSyncing = false
		sm.cancel = nil
		sm.mu.Unlock()
		if onComplete != nil {
			onComplete(err)
		}
	}()

	return nil
}

// ParseMTPStorageID extracts the storage ID from an MTP mount path (e.g. mtp://storage-65537).
func ParseMTPStorageID(mountPath string) uint32 {
	var sid uint32
	if n, _ := fmt.Sscanf(mountPath, "mtp://storage-%d", &sid); n == 1 && sid > 0 {
		return sid
	}
	return 65537
}


