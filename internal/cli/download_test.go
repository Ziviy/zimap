package cli

import (
	"strings"
	"testing"
)

func TestDownloadCommandRejectsMoreThanOneArgument(t *testing.T) {
	err := DownloadCommand().Run([]string{"one", "two"})
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("DownloadCommand().Run() error = %v, want usage error", err)
	}
}
