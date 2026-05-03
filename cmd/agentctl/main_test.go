package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sebastiaankok/agents/internal/jobspec"
)

func TestRunCommand_JobImageFlag(t *testing.T) {
	var capturedCfg jobspec.Config
	cmd := newRunCmd(func(cfg jobspec.Config) {
		capturedCfg = cfg
	})
	cmd.SetArgs([]string{"--job-image", "custom:v2", "--dry-run"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedCfg.JobImage != "custom:v2" {
		t.Errorf("JobImage = %q, want %q", capturedCfg.JobImage, "custom:v2")
	}
}

func TestRunCommand_Flags(t *testing.T) {
	var capturedCfg jobspec.Config
	cmd := newRunCmd(func(cfg jobspec.Config) {
		capturedCfg = cfg
	})
	cmd.SetArgs([]string{
		"--job-image", "custom:v2",
		"--issue", "42",
		"--namespace", "test-ns",
		"--max-parallel", "5",
		"--dry-run",
	})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedCfg.JobImage != "custom:v2" {
		t.Errorf("JobImage = %q, want %q", capturedCfg.JobImage, "custom:v2")
	}
	if capturedCfg.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", capturedCfg.Namespace, "test-ns")
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
