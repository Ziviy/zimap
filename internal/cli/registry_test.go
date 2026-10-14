package cli

import (
	"strings"
	"testing"
)

func TestGetCommands(t *testing.T) {
	cmds := GetCommands()

	expected := []string{"download", "convert", "update"}
	for _, name := range expected {
		cmd, exists := cmds[name]
		if !exists {
			t.Errorf("expected command %q to be registered", name)
		}
		if cmd.Name != name {
			t.Errorf("cmd.Name = %q, want %q", cmd.Name, name)
		}
		if cmd.Run == nil {
			t.Errorf("cmd.Run for %q is nil", name)
		}
	}
}

func TestProcessCommandEmptyArgs(t *testing.T) {
	err := ProcessCommand([]string{})
	if err == nil || !strings.Contains(err.Error(), "no command provided") {
		t.Errorf("expected no command provided error, got: %v", err)
	}
}

func TestProcessCommandUnsupported(t *testing.T) {
	err := ProcessCommand([]string{"non-existent-cmd"})
	if err == nil || !strings.Contains(err.Error(), "Command not supported") {
		t.Errorf("expected Command not supported error, got: %v", err)
	}
}
