package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sebastiaankok/agents/internal/jobspec"
)

func TestRunCommand_JobImageFlag(t *testing.T) {
	var capturedImage string
	cmd := newRunCmd(func(cfg jobspec.Config) {
		capturedImage = cfg.JobImage
	})
	cmd.SetArgs([]string{"--job-image", "custom:v2"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedImage != "custom:v2" {
		t.Errorf("Image = %q, want %q", capturedImage, "custom:v2")
	}
}

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
