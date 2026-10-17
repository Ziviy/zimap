package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Custom error for download failures.
var ErrDownloadFailed = errors.New("download failed")

// DownloadError holds details about a specific download failure.
type DownloadError struct {
	Status string // HTTP Status code received (e.g., "404 Not Found")
	URL    string
}

func (e *DownloadError) Error() string {
	return fmt.Sprintf("%s: URL %s returned status %s", ErrDownloadFailed.Error(), e.URL, e.Status)
}

func (e *DownloadError) Unwrap() error {
	return ErrDownloadFailed
}

// ProxyConfig holds proxy configuration settings.
type ProxyConfig struct {
	UseSystemProxy bool   `json:"useSystemProxy"`
	CustomProxy    string `json:"customProxy,omitempty"`
}

var (
	currentProxyMu     sync.RWMutex
	currentProxyConfig = ProxyConfig{UseSystemProxy: true}
)

// SetGlobalProxy updates global proxy configuration.
func SetGlobalProxy(config ProxyConfig) {
	currentProxyMu.Lock()
	defer currentProxyMu.Unlock()
	currentProxyConfig = config
}

// GetGlobalProxy returns the active proxy configuration.
func GetGlobalProxy() ProxyConfig {
	currentProxyMu.RLock()
	defer currentProxyMu.RUnlock()
	return currentProxyConfig
}

func makeTransport(headerTimeout time.Duration) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = headerTimeout

	currentProxyMu.RLock()
	cfg := currentProxyConfig
	currentProxyMu.RUnlock()

	if !cfg.UseSystemProxy && cfg.CustomProxy != "" {
		if parsedURL, err := url.Parse(cfg.CustomProxy); err == nil {
			transport.Proxy = http.ProxyURL(parsedURL)
		}
	} else if cfg.UseSystemProxy {
		transport.Proxy = http.ProxyFromEnvironment
	} else {
		transport.Proxy = nil
	}

	return transport
}

// Client manages HTTP download operations.
type Client struct {
	Client *http.Client
}

// New creates a new download client.
func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout:   90 * time.Second,
			Transport: makeTransport(90 * time.Second),
		}
	}
	return &Client{
		Client: httpClient,
	}
}

// NewStreaming creates a client for downloads whose duration is determined by
// file size and connection speed. Connection establishment and response headers
// still have bounded timeouts (90s), but reading the response body has no blanket deadline.
func NewStreaming() *Client {
	return New(&http.Client{Transport: makeTransport(90 * time.Second)})
}

// DownloadOptions configures progress callbacks, logging, and retry parameters.
type DownloadOptions struct {
	ProgressCallback func(downloaded int64, total int64, speedBps float64)
	LogCallback      func(msg string)
	MaxRetries       int
	InitialBackoff   time.Duration
}

// Download performs an HTTP GET request and saves the content to a specified file path.
func (client *Client) Download(fileUrl string, targetPath string) error {
	return client.DownloadWithContext(context.Background(), fileUrl, targetPath, nil)
}

// DownloadWithContext performs a download with context support, retry loop, and progress reporting.
func (client *Client) DownloadWithContext(ctx context.Context, fileUrl string, targetPath string, progressCallback func(downloaded int64, total int64, speedBps float64)) error {
	return client.DownloadWithOptions(ctx, fileUrl, targetPath, DownloadOptions{
		ProgressCallback: progressCallback,
		MaxRetries:       3,
		InitialBackoff:   2 * time.Second,
	})
}

// DownloadWithOptions performs a download with custom retry parameters and optional log callback.
func (client *Client) DownloadWithOptions(ctx context.Context, fileUrl string, targetPath string, opts DownloadOptions) error {
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}
	backoff := opts.InitialBackoff
	if backoff <= 0 {
		backoff = 2 * time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := client.downloadAttempt(ctx, fileUrl, targetPath, opts.ProgressCallback)
		if err == nil {
			return nil
		}
		lastErr = err

		// If user cancelled, do not retry
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			return err
		}

		// Non-retryable HTTP errors (e.g. 404 Not Found, 403 Forbidden)
		var dlErr *DownloadError
		if errors.As(err, &dlErr) {
			if dlErr.Status == "404 Not Found" || dlErr.Status == "403 Forbidden" {
				return err
			}
		}

		if attempt < maxRetries {
			if opts.LogCallback != nil {
				opts.LogCallback(fmt.Sprintf("[WARN] Download interrupted (%v). Retrying (attempt %d of %d) in %v...", err, attempt+1, maxRetries, backoff))
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}
	}
	return fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

func (client *Client) downloadAttempt(ctx context.Context, fileUrl string, targetPath string, progressCallback func(downloaded int64, total int64, speedBps float64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", fileUrl, err)
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to initiate HTTP request for %s: %w", fileUrl, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &DownloadError{Status: resp.Status, URL: fileUrl}
	}

	dir := filepath.Dir(targetPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	out, err := os.CreateTemp(dir, "."+filepath.Base(targetPath)+".download-*")
	if err != nil {
		return fmt.Errorf("create temporary download for %s: %w", targetPath, err)
	}
	temporaryPath := out.Name()
	defer os.Remove(temporaryPath)

	var reader io.Reader = resp.Body
	if progressCallback != nil {
		total := resp.ContentLength
		reader = &progressReader{
			reader:   resp.Body,
			total:    total,
			progress: progressCallback,
		}
	}

	if _, err = io.Copy(out, reader); err != nil {
		out.Close()
		return fmt.Errorf("failed to write downloaded content to %s: %w", targetPath, err)
	}
	if err = out.Close(); err != nil {
		return fmt.Errorf("close downloaded content for %s: %w", targetPath, err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	if err = replaceFile(temporaryPath, targetPath); err != nil {
		return fmt.Errorf("install downloaded file at %s: %w", targetPath, err)
	}
	return nil
}

type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	startTime time.Time
	lastTime  time.Time
	lastBytes int64
	speedBps  float64
	progress  func(downloaded int64, total int64, speedBps float64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		now := time.Now()
		if pr.startTime.IsZero() {
			pr.startTime = now
			pr.lastTime = now
			pr.lastBytes = 0
		}

		pr.current += int64(n)

		timeDelta := now.Sub(pr.lastTime)
		if timeDelta.Seconds() >= 0.5 || pr.speedBps == 0 {
			if timeDelta.Seconds() > 0 {
				bytesDelta := pr.current - pr.lastBytes
				pr.speedBps = float64(bytesDelta) / timeDelta.Seconds()
			}
			pr.lastTime = now
			pr.lastBytes = pr.current
		}

		if pr.progress != nil {
			pr.progress(pr.current, pr.total, pr.speedBps)
		}
	}
	return n, err
}

func (client *Client) ContentLength(fileURL string) (int64, error) {
	request, err := http.NewRequest(http.MethodHead, fileURL, nil)
	if err != nil {
		return -1, err
	}
	response, err := client.Client.Do(request)
	if err != nil {
		return -1, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return -1, &DownloadError{Status: response.Status, URL: fileURL}
	}
	if response.ContentLength < 0 {
		return -1, fmt.Errorf("content length is unavailable for %s", fileURL)
	}
	return response.ContentLength, nil
}

func replaceFile(sourcePath string, targetPath string) error {
	info, statErr := os.Lstat(targetPath)
	hasExistingTarget := statErr == nil
	if statErr != nil && !os.IsNotExist(statErr) {
		return statErr
	}
	if hasExistingTarget && info.IsDir() {
		return fmt.Errorf("cannot replace directory %s with a downloaded file", targetPath)
	}

	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	} else if !hasExistingTarget {
		return err
	}

	// Windows cannot rename over an existing file; retain a backup for rollback.
	backupPath, err := os.CreateTemp(filepath.Dir(targetPath), ".previous-*")
	if err != nil {
		return err
	}
	backupName := backupPath.Name()
	if err := backupPath.Close(); err != nil {
		_ = os.Remove(backupName)
		return err
	}
	if err := os.Remove(backupName); err != nil {
		return err
	}
	if err := os.Rename(targetPath, backupName); err != nil {
		return err
	}

	if err := os.Rename(sourcePath, targetPath); err != nil {
		if restoreErr := os.Rename(backupName, targetPath); restoreErr != nil {
			return fmt.Errorf("restore previous download (preserved at %s): %w", backupName, errors.Join(err, restoreErr))
		}
		return err
	}
	_ = os.Remove(backupName)
	return nil
}
