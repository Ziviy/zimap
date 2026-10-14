//go:build !darwin && !windows

package device

import (
	"os"
	"os/exec"
	"path/filepath"
)

func scanCandidateMounts() []string {
	var list []string
	user := os.Getenv("USER")
	baseDirs := []string{
		"/media",
		filepath.Join("/media", user),
		filepath.Join("/run/media", user),
		"/mnt",
	}

	for _, base := range baseDirs {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			full := filepath.Join(base, entry.Name())
			if info, err := os.Stat(full); err == nil && info.IsDir() {
				list = append(list, full)
			}
		}
	}
	return list
}

// EjectDevice invokes umount for Linux systems.
func EjectDevice(mountPath string) error {
	cmd := exec.Command("umount", mountPath)
	return cmd.Run()
}
