package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"zimap/internal/setup"
)

// SendNotification displays a system notification across macOS, Windows, and Linux.
func (a *App) SendNotification(title, message string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf("display notification %q with title %q sound name \"default\"", message, title)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()
	case "windows":
		escapedTitle := strings.ReplaceAll(title, "'", "''")
		escapedMsg := strings.ReplaceAll(message, "'", "''")
		psScript := fmt.Sprintf(
			`[void] [System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms'); $notify = New-Object System.Windows.Forms.NotifyIcon; $notify.Icon = [System.Drawing.SystemIcons]::Information; $notify.BalloonTipTitle = '%s'; $notify.BalloonTipText = '%s'; $notify.Visible = $True; $notify.ShowBalloonTip(5000); Start-Sleep -Milliseconds 500; $notify.Dispose()`,
			escapedTitle,
			escapedMsg,
		)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
		return cmd.Run()
	case "linux":
		if p, err := exec.LookPath("notify-send"); err == nil {
			args := []string{title, message}
			if root, err := setup.ProjectRoot(); err == nil {
				iconCandidate := filepath.Join(root, "build", "appicon.png")
				if _, err := os.Stat(iconCandidate); err == nil {
					args = []string{"-i", iconCandidate, title, message}
				}
			}
			cmd := exec.Command(p, args...)
			return cmd.Run()
		}
		return nil
	default:
		return nil
	}
}
