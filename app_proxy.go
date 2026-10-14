package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"zimap/internal/download"
)

func (a *App) GetProxy() (download.ProxyConfig, error) {
	cacheDir := getCacheDir()
	cfgFile := filepath.Join(cacheDir, "proxy-config.json")
	if data, err := os.ReadFile(cfgFile); err == nil {
		var cfg download.ProxyConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			return cfg, nil
		}
	}
	// Default configuration: use system proxy
	return download.ProxyConfig{
		UseSystemProxy: true,
		CustomProxy:    "",
	}, nil
}

func (a *App) SetProxy(useSystem bool, customProxy string) error {
	customProxy = strings.TrimSpace(customProxy)
	if customProxy != "" {
		u, err := url.Parse(customProxy)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5") {
			return fmt.Errorf("invalid proxy URL: must start with http://, https://, or socks5://")
		}
	}

	cfg := download.ProxyConfig{
		UseSystemProxy: useSystem,
		CustomProxy:    customProxy,
	}

	download.SetGlobalProxy(cfg)

	cacheDir := getCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, "proxy-config.json"), data, 0644)
}
