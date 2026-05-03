package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandHelp(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "agentctl") {
		t.Errorf("help output does not contain %q: %s", "agentctl", out)
	}
}
