package device

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CheckGarminUSB detects if a Garmin device is connected via USB even if its filesystem is not mounted.
// mountedCount is the number of currently mounted Garmin devices found by ScanDevices.
func CheckGarminUSB(mountedCount int) USBStatus {
	// Support testing mock environment variable
	if os.Getenv("ZIMAP_MOCK_GARMIN_MTP") == "1" {
		return USBStatus{
			GarminUSBAttached: true,
			DeviceDescription: "Garmin Fenix 7 (Mock USB)",
			NeedsMTPNotice:    mountedCount == 0,
		}
	}

	attached, isGarminMode, desc := detectGarminUSBHardware()

	needsNotice := false
	if attached && mountedCount == 0 {
		// Device is attached by USB, but no storage volume was mounted
		needsNotice = true
	}

	conflictMsg := ""
	if needsNotice {
		mtpSvc := NewMTPService()
		if c, err := mtpSvc.CheckConflicts(context.Background()); err == nil && len(c) > 0 {
			conflictMsg = strings.Join(c, ", ")
		}
	}

	return USBStatus{
		GarminUSBAttached: attached,
		DeviceDescription: desc,
		NeedsMTPNotice:    needsNotice,
		IsGarminMode:      isGarminMode,
		MTPConflict:       conflictMsg,
	}
}

func detectGarminUSBHardware() (bool, bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "darwin":
		// Fast check via ioreg
		out, err := exec.CommandContext(ctx, "ioreg", "-p", "IOUSB", "-l").Output()
		if err == nil {
			str := string(out)
			if strings.Contains(str, "2334") || strings.Contains(strings.ToLower(str), "0x091e") || strings.Contains(str, "Garmin") {
				// Check for PID 3 (0x0003), which is Garmin's proprietary protocol mode ("Режим Garmin")
				isGarminMode := strings.Contains(str, "\"idProduct\" = 3") || strings.Contains(str, "idProduct = 3")
				desc := "Garmin Device (MTP Mode)"
				if isGarminMode {
					desc = "Garmin Device (Garmin Protocol Mode)"
				}
				return true, isGarminMode, desc
			}
		}

		// Fallback to system_profiler
		outSP, errSP := exec.CommandContext(ctx, "system_profiler", "SPUSBDataType").Output()
		if errSP == nil {
			strSP := string(outSP)
			if strings.Contains(strSP, "0x091e") || strings.Contains(strSP, "0x091E") || strings.Contains(strSP, "Garmin") {
				isGarminMode := strings.Contains(strSP, "Product ID: 0x0003") || strings.Contains(strSP, "Product ID: 3")
				desc := "Garmin Device (MTP Mode)"
				if isGarminMode {
					desc = "Garmin Device (Garmin Protocol Mode)"
				}
				return true, isGarminMode, desc
			}
		}

	case "windows":
		psCmd := `Get-PnpDevice -PresentOnly | Where-Object { $_.InstanceId -like "*VID_091E*" } | Select-Object -ExpandProperty FriendlyName`
		out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd).Output()
		if err == nil {
			trimmed := strings.TrimSpace(string(out))
			if trimmed != "" {
				return true, false, trimmed
			}
		}

	case "linux":
		if p, err := exec.LookPath("lsusb"); err == nil {
			out, err := exec.CommandContext(ctx, p).Output()
			if err == nil && strings.Contains(strings.ToLower(string(out)), "091e:") {
				isGarminMode := strings.Contains(strings.ToLower(string(out)), "091e:0003")
				return true, isGarminMode, "Garmin Device (USB)"
			}
		}
	}

	return false, false, ""
}
