package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// InferRepo runs `git remote -v` and returns the GitHub owner/repo.
func InferRepo() (string, error) {
	out, err := exec.Command("git", "remote", "-v").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}

	repo := parseRemoteOutput(string(out))
	if repo == "" {
		return "", fmt.Errorf("no GitHub remote found in `git remote -v`")
	}
	return repo, nil
}

// parseRemoteOutput extracts owner/repo from git remote -v output.
func parseRemoteOutput(output string) string {
	for _, line := range strings.Split(output, "\n") {
		repo := extractRepo(line)
		if repo != "" {
			return repo
		}
	}
	return ""
}

// extractRepo parses a single git remote line and returns owner/repo if it's a GitHub URL.
func extractRepo(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	parts := strings.SplitN(line, "\t", 2)
	if len(parts) < 2 {
		return ""
	}
	url := strings.Fields(parts[1])[0]

	return parseURL(url)
}

// githubHTTPRe matches HTTPS GitHub URLs like https://github.com/owner/repo.git
var githubHTTPRe = regexp.MustCompile(`^https?://github\.com/([^/]+/[^/]+?)(?:\.git)?$`)

// githubSSHRe matches SSH GitHub URLs like git@github.com:owner/repo.git
var githubSSHRe = regexp.MustCompile(`^git@github\.com:([^/]+/[^/]+?)(?:\.git)?$`)

func parseURL(url string) string {
	if m := githubHTTPRe.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	if m := githubSSHRe.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	return ""
}
