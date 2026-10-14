package cli

import (
	"strings"
	"testing"
)

func TestConvertCommandRequiresOnePath(t *testing.T) {
	err := ConvertCommand().Run(nil)
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("ConvertCommand().Run() error = %v, want usage error", err)
	}
}
