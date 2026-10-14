//go:build windows

package device

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func scanCandidateMounts() []string {
	var list []string
	for r := 'C'; r <= 'Z'; r++ {
		drive := fmt.Sprintf("%c:\\", r)
		if info, err := os.Stat(drive); err == nil && info.IsDir() {
			if _, err2 := os.Stat(filepath.Join(drive, "GARMIN")); err2 == nil {
				list = append(list, drive)
			} else if _, err3 := os.Stat(filepath.Join(drive, "GarminDevice.xml")); err3 == nil {
				list = append(list, drive)
			}
		}
	}
	return list
}

// EjectDevice invokes Windows Shell Application COM object to eject the drive.
func EjectDevice(mountPath string) error {
	vol := filepath.VolumeName(mountPath)
	if vol == "" {
		vol = strings.TrimSuffix(mountPath, "\\")
	}
	psScript := fmt.Sprintf("(New-Object -comObject Shell.Application).Namespace(17).ParseName('%s').InvokeVerb('Eject')", vol)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	return cmd.Run()
}
