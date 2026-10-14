//go:build darwin

package device

import (
	"os"
	"os/exec"
	"path/filepath"
)

func scanCandidateMounts() []string {
	var list []string
	entries, err := os.ReadDir("/Volumes")
	if err != nil {
		return list
	}
	for _, entry := range entries {
		full := filepath.Join("/Volumes", entry.Name())
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			list = append(list, full)
		}
	}
	return list
}

// EjectDevice cleanly unmounts and ejects the specified Garmin volume on macOS.
func EjectDevice(mountPath string) error {
	cmd := exec.Command("diskutil", "eject", mountPath)
	if err := cmd.Run(); err == nil {
		return nil
	}
	cmdUnmount := exec.Command("diskutil", "unmountDisk", mountPath)
	return cmdUnmount.Run()
}
