package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"zimap/internal/device"
	"zimap/internal/download"
	"zimap/internal/setup"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const CurrentAppVersion = "0.0.5"

func getCacheDir() string {
	if dir, err := setup.CacheDirectory(); err == nil && dir != "" {
		return dir
	}
	return "cache"
}

type App struct {
	ctx              context.Context
	cancelJob        context.CancelFunc
	jobMutex         sync.Mutex
	catalogCache     *CatalogData
	catalogMutex     sync.RWMutex
	catalogTime      time.Time
	providerCatalogs map[string]*CatalogData
	providerTimes    map[string]time.Time
	syncManager      *device.SyncManager
	mtpService       *device.MTPService
	updateCancel     context.CancelFunc
	updateRunning    bool
	updateMutex      sync.Mutex
}

func NewApp() *App {
	a := &App{
		providerCatalogs: make(map[string]*CatalogData),
		providerTimes:    make(map[string]time.Time),
		syncManager:      device.NewSyncManager(),
		mtpService:       device.NewMTPService(),
	}
	if cfg, err := a.GetProxy(); err == nil {
		download.SetGlobalProxy(cfg)
	}
	return a
}

func (a *App) getSyncManager() *device.SyncManager {
	if a.syncManager == nil {
		a.syncManager = device.NewSyncManager()
	}
	return a.syncManager
}

func (a *App) getMTPService() *device.MTPService {
	if a.mtpService == nil {
		a.mtpService = device.NewMTPService()
	}
	return a.mtpService
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) domReady(ctx context.Context) {
}

func (a *App) shutdown(ctx context.Context) {
}

func (a *App) emitEvent(name string, optionalData ...interface{}) {
	if a == nil || a.ctx == nil {
		return
	}
	defer func() {
		_ = recover()
	}()
	wailsruntime.EventsEmit(a.ctx, name, optionalData...)
}

type ProgressDetail struct {
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    int     `json:"percent"`
	SpeedBps   float64 `json:"speedBps"`
	SpeedText  string  `json:"speedText"`
	EtaSeconds int     `json:"etaSeconds"`
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSec/(1024*1024))
	} else if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.0f B/s", bytesPerSec)
}
