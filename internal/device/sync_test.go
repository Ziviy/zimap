package device

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveDestinationPath(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Errors on invalid inputs
	if _, err := ResolveDestinationPath("", "test.img"); err == nil {
		t.Errorf("expected error on empty mount")
	}
	if _, err := ResolveDestinationPath(tmpDir, ""); err == nil {
		t.Errorf("expected error on empty filename")
	}
	if _, err := ResolveDestinationPath(tmpDir, "test.txt"); !errors.Is(err, ErrNotAnImgFile) {
		t.Errorf("expected ErrNotAnImgFile, got %v", err)
	}
	if _, err := ResolveDestinationPath(tmpDir, "gmapbmap.img"); !errors.Is(err, ErrSystemMapProtected) {
		t.Errorf("expected ErrSystemMapProtected, got %v", err)
	}

	// 2. Error if no GARMIN folder exists
	if _, err := ResolveDestinationPath(tmpDir, "map.img"); !errors.Is(err, ErrGarminDirNotFound) {
		t.Errorf("expected ErrGarminDirNotFound, got %v", err)
	}

	garminDir := filepath.Join(tmpDir, "GARMIN")
	if err := os.MkdirAll(garminDir, 0755); err != nil {
		t.Fatalf("failed to create GARMIN dir: %v", err)
	}

	// 3. New file when only GARMIN/ exists
	dest, err := ResolveDestinationPath(tmpDir, "russia.img")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := filepath.Join(garminDir, "russia.img")
	if dest != expected {
		t.Errorf("expected %s, got %s", expected, dest)
	}

	// 4. New file when GARMIN/Map/ exists
	mapDir := filepath.Join(garminDir, "Map")
	if err := os.MkdirAll(mapDir, 0755); err != nil {
		t.Fatalf("failed to create GARMIN/Map: %v", err)
	}

	destMap, err := ResolveDestinationPath(tmpDir, "spb.img")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedMap := filepath.Join(mapDir, "spb.img")
	if destMap != expectedMap {
		t.Errorf("expected %s, got %s", expectedMap, destMap)
	}

	// 5. Existing file in GARMIN/ (even if Map/ exists) preserves its existing location
	existingGarmin := filepath.Join(garminDir, "moscow.img")
	if err := os.WriteFile(existingGarmin, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to write dummy moscow: %v", err)
	}

	destExisting, err := ResolveDestinationPath(tmpDir, "moscow.img")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if destExisting != existingGarmin {
		t.Errorf("expected existing path %s, got %s", existingGarmin, destExisting)
	}
}

func TestSyncFile_SuccessAndTimestamp(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "output")
	destDir := filepath.Join(tmpDir, "device", "GARMIN", "Map")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.MkdirAll(destDir, 0755)

	srcPath := filepath.Join(srcDir, "testmap.img")
	destPath := filepath.Join(destDir, "testmap.img")

	// Generate 500 KB test payload
	content := bytes.Repeat([]byte("ABCDEF0123456789"), 32*1024)
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to write test source: %v", err)
	}

	// Set back source modtime by 3 days
	srcModTime := time.Now().Add(-72 * time.Hour).Truncate(2 * time.Second)
	_ = os.Chtimes(srcPath, srcModTime, srcModTime)

	var progressEvents []SyncProgressEvent
	progressFn := func(e SyncProgressEvent) {
		progressEvents = append(progressEvents, e)
	}

	ctx := context.Background()
	err := SyncFile(ctx, srcPath, destPath, progressFn)
	if err != nil {
		t.Fatalf("SyncFile failed: %v", err)
	}

	// 1. Verify content
	destContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read dest map: %v", err)
	}
	if !bytes.Equal(destContent, content) {
		t.Errorf("destination content does not match source")
	}

	// 2. Verify timestamp preservation
	destStat, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("failed to stat dest map: %v", err)
	}
	diff := destStat.ModTime().Sub(srcModTime)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expected mod time close to %v, got %v (diff %v)", srcModTime, destStat.ModTime(), diff)
	}

	// 3. Verify .tmp-zimap cleaned up
	tempFile := destPath + ".tmp-zimap"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Errorf("expected temp file to be removed, but it exists: %s", tempFile)
	}

	// 4. Verify progress events
	if len(progressEvents) == 0 {
		t.Errorf("expected progress events, got 0")
	} else {
		lastEvent := progressEvents[len(progressEvents)-1]
		if lastEvent.Percentage != 100 {
			t.Errorf("expected last event percentage 100, got %d", lastEvent.Percentage)
		}
	}
}

func TestSyncFile_CancelCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	srcDir := filepath.Join(tmpDir, "output")
	destDir := filepath.Join(tmpDir, "device", "GARMIN")
	_ = os.MkdirAll(srcDir, 0755)
	_ = os.MkdirAll(destDir, 0755)

	srcPath := filepath.Join(srcDir, "large.img")
	destPath := filepath.Join(destDir, "large.img")

	// 2 MB file
	content := bytes.Repeat([]byte("X"), 2*1024*1024)
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatalf("failed to write test source: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := SyncFile(ctx, srcPath, destPath, nil)
	if err == nil {
		t.Fatalf("expected error on cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	// Verify temp file cleaned up
	tempFile := destPath + ".tmp-zimap"
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Errorf("expected temp file to be removed after cancel, but it exists")
	}
}

func TestSyncManager_ConcurrentGuard(t *testing.T) {
	sm := NewSyncManager()
	if sm.IsSyncing() {
		t.Errorf("expected IsSyncing to be false initially")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.img")
	dest := filepath.Join(tmpDir, "dest.img")
	if err := os.WriteFile(src, []byte("map data"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})
	releaseCopy := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- sm.Sync(ctx, src, dest, func(e SyncProgressEvent) {
			if e.BytesCopied == 0 {
				close(started)
				// Hold the busy slot until the second call has checked the guard.
				select {
				case <-releaseCopy:
				case <-ctx.Done():
				}
			}
		})
	}()

	select {
	case <-started:
	case err := <-done:
		t.Fatalf("first sync ended before reporting progress: %v", err)
	}

	err2 := sm.Sync(context.Background(), src, dest, nil)
	if !errors.Is(err2, ErrSyncInProgress) {
		t.Errorf("expected ErrSyncInProgress, got %v", err2)
	}

	sm.Cancel()
	close(releaseCopy)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Errorf("expected cancellation of first sync, got %v", err)
	}

	if sm.IsSyncing() {
		t.Errorf("expected IsSyncing to be false after completion")
	}
}
