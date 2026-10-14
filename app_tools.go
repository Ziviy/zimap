package main

import (
	"fmt"
	"os"
	"runtime"

	"zimap/internal/setup"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type ToolDetail struct {
	Installed bool   `json:"installed"`
	Embedded  bool   `json:"embedded"`
	Path      string `json:"path"`
}

type ToolsStatus struct {
	Installed bool       `json:"installed"`
	Java      ToolDetail `json:"java"`
	Mkgmap    ToolDetail `json:"mkgmap"`
	Splitter  ToolDetail `json:"splitter"`
	Error     string     `json:"error,omitempty"`
}

func (a *App) CheckTools() (ToolsStatus, error) {
	status := ToolsStatus{Installed: true}

	if p, emb, err := setup.JavaDetails(); err == nil {
		status.Java = ToolDetail{Installed: true, Embedded: emb, Path: p}
	} else {
		status.Installed = false
	}

	if p, emb, err := setup.ToolDetails("mkgmap"); err == nil {
		status.Mkgmap = ToolDetail{Installed: true, Embedded: emb, Path: p}
	} else {
		status.Installed = false
	}

	if p, emb, err := setup.ToolDetails("splitter"); err == nil {
		status.Splitter = ToolDetail{Installed: true, Embedded: emb, Path: p}
	} else {
		status.Installed = false
	}

	if !status.Installed {
		if err := setup.CheckToolsDirectory(); err != nil {
			status.Error = err.Error()
		}
	}

	return status, nil
}

func (a *App) OpenURL(targetURL string) error {
	if a.ctx != nil {
		wailsruntime.BrowserOpenURL(a.ctx, targetURL)
	}
	return nil
}

func (a *App) UpdateTools() error {
	// temporarily capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := r.Read(buf)
			if n > 0 {
				a.emitEvent("tools:log", string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
	}()

	err := setup.UpdateTools()
	w.Close()
	os.Stdout = oldStdout
	return err
}

func (a *App) GetSystemInfo() map[string]string {
	return map[string]string{
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"go_version":  runtime.Version(),
		"num_cpu":     fmt.Sprintf("%d", runtime.NumCPU()),
		"app_version": CurrentAppVersion,
	}
}

func (a *App) GetAppVersion() string {
	return CurrentAppVersion
}
