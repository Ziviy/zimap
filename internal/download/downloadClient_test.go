package download

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadSuccess(t *testing.T) {
	client := clientWithResponse(http.StatusOK, io.NopCloser(strings.NewReader("new content")))

	targetPath := filepath.Join(t.TempDir(), "nested", "map.pbf")
	if err := New(client).Download("https://example.test/map.pbf", targetPath); err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != "new content" {
		t.Fatalf("downloaded content = %q, want %q", content, "new content")
	}
}

func TestDownloadHTTPErrorPreservesExistingFile(t *testing.T) {
	client := clientWithResponse(http.StatusNotFound, io.NopCloser(strings.NewReader("not found")))

	targetPath := filepath.Join(t.TempDir(), "map.pbf")
	if err := os.WriteFile(targetPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := New(client).Download("https://example.test/missing.pbf", targetPath)
	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf("Download() error = %v, want ErrDownloadFailed", err)
	}

	content, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "old content" {
		t.Fatalf("existing content = %q, want it preserved", content)
	}
}

func TestDownloadWriteFailurePreservesExistingFile(t *testing.T) {
	body := io.NopCloser(io.MultiReader(strings.NewReader("partial"), errorReader{}))
	client := clientWithResponse(http.StatusOK, body)

	targetPath := filepath.Join(t.TempDir(), "map.pbf")
	if err := os.WriteFile(targetPath, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := New(client).Download("https://example.test/map.pbf", targetPath); err == nil {
		t.Fatal("Download() error = nil, want truncated response error")
	}

	content, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != "old content" {
		t.Fatalf("existing content = %q, want it preserved", content)
	}
}

func TestStreamingClientHasNoOverallBodyDeadline(t *testing.T) {
	client := NewStreaming()
	if client.Client.Timeout != 0 {
		t.Fatalf("streaming client timeout = %v, want no overall body deadline", client.Client.Timeout)
	}

	transport, ok := client.Client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("streaming transport type = %T, want *http.Transport", client.Client.Transport)
	}
	if transport.ResponseHeaderTimeout == 0 {
		t.Fatal("streaming client must retain a response-header timeout")
	}
}

func TestContentLengthUsesHEAD(t *testing.T) {
	var method string
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		method = request.Method
		return &http.Response{
			StatusCode:    http.StatusOK,
			Status:        "200 OK",
			Body:          http.NoBody,
			ContentLength: 12345,
			Header:        make(http.Header),
			Request:       request,
		}, nil
	})}

	size, err := New(httpClient).ContentLength("https://example.test/map.pbf")
	if err != nil {
		t.Fatalf("ContentLength() error = %v", err)
	}
	if method != http.MethodHead {
		t.Fatalf("request method = %q, want HEAD", method)
	}
	if size != 12345 {
		t.Fatalf("ContentLength() = %d, want 12345", size)
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func clientWithResponse(statusCode int, body io.ReadCloser) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: statusCode,
			Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
			Body:       body,
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})}
}

type errorReader struct{}

func (errorReader) Read(buffer []byte) (int, error) {
	return 0, errors.New("simulated response interruption")
}

func TestDownloadRetrySuccess(t *testing.T) {
	attempts := 0
	httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		attempts++
		if attempts == 1 {
			return nil, errors.New("temporary timeout error")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("retry success data")),
			Header:     make(http.Header),
			Request:    request,
		}, nil
	})}

	targetPath := filepath.Join(t.TempDir(), "retry.pbf")
	var loggedWarnings []string
	client := New(httpClient)
	err := client.DownloadWithOptions(t.Context(), "https://example.test/retry.pbf", targetPath, DownloadOptions{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		LogCallback: func(msg string) {
			loggedWarnings = append(loggedWarnings, msg)
		},
	})
	if err != nil {
		t.Fatalf("DownloadWithOptions() error = %v, want success", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
	if len(loggedWarnings) != 1 {
		t.Fatalf("loggedWarnings count = %d, want 1", len(loggedWarnings))
	}
	data, err := os.ReadFile(targetPath)
	if err != nil || string(data) != "retry success data" {
		t.Fatalf("downloaded data = %q, want %q", string(data), "retry success data")
	}
}

func TestProxyConfig(t *testing.T) {
	orig := GetGlobalProxy()
	defer SetGlobalProxy(orig)

	SetGlobalProxy(ProxyConfig{
		UseSystemProxy: false,
		CustomProxy:    "http://127.0.0.1:8888",
	})
	cfg := GetGlobalProxy()
	if cfg.UseSystemProxy || cfg.CustomProxy != "http://127.0.0.1:8888" {
		t.Fatalf("unexpected proxy config: %+v", cfg)
	}
}

