package git

import (
	"strings"
	"testing"
)

func TestParseRemoteOutput_HTTPS(t *testing.T) {
	output := "origin\thttps://github.com/sebastiaankok/agents.git (fetch)\n" +
		"origin\thttps://github.com/sebastiaankok/agents.git (push)"
	got := parseRemoteOutput(output)
	if got != "sebastiaankok/agents" {
		t.Fatalf("parseRemoteOutput() = %q, want %q", got, "sebastiaankok/agents")
	}
}

func TestParseRemoteOutput_SSH(t *testing.T) {
	output := "origin\tgit@github.com:sebastiaankok/agents.git (fetch)\n" +
		"origin\tgit@github.com:sebastiaankok/agents.git (push)"
	got := parseRemoteOutput(output)
	if got != "sebastiaankok/agents" {
		t.Fatalf("parseRemoteOutput() = %q, want %q", got, "sebastiaankok/agents")
	}
}

func TestParseRemoteOutput_NoGitSuffix(t *testing.T) {
	output := "origin\thttps://github.com/owner/repo (fetch)"
	got := parseRemoteOutput(output)
	if got != "owner/repo" {
		t.Fatalf("parseRemoteOutput() = %q, want %q", got, "owner/repo")
	}
}

func TestParseRemoteOutput_Empty(t *testing.T) {
	got := parseRemoteOutput("")
	if got != "" {
		t.Fatalf("parseRemoteOutput() = %q, want empty", got)
	}
}

func TestParseRemoteOutput_NonGitHub(t *testing.T) {
	output := "origin\thttps://gitlab.com/owner/repo.git (fetch)"
	got := parseRemoteOutput(output)
	if got != "" {
		t.Fatalf("parseRemoteOutput() = %q, want empty for non-GitHub URL", got)
	}
}

func TestExtractRepo_HTTPS(t *testing.T) {
	line := "origin\thttps://github.com/owner/repo.git (fetch)"
	got := extractRepo(line)
	if got != "owner/repo" {
		t.Fatalf("extractRepo() = %q, want %q", got, "owner/repo")
	}
}

func TestExtractRepo_SSH(t *testing.T) {
	line := "origin\tgit@github.com:owner/repo.git (fetch)"
	got := extractRepo(line)
	if got != "owner/repo" {
		t.Fatalf("extractRepo() = %q, want %q", got, "owner/repo")
	}
}

func TestExtractRepo_Empty(t *testing.T) {
	got := extractRepo("")
	if got != "" {
		t.Fatalf("extractRepo() = %q, want empty", got)
	}
}

func TestInferRepo_Integration(t *testing.T) {
	repo, err := InferRepo()
	if err != nil {
		t.Skipf("skipping: not in a git repo with GitHub remote (%v)", err)
	}
	if !strings.Contains(repo, "/") {
		t.Errorf("InferRepo() = %q, want owner/repo format", repo)
	}
}
