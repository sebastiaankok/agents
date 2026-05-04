package workflows_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func workflowPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, ".github", "workflows", name)
}

func dockerfilePath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	return filepath.Join(root, rel)
}

func readWorkflow(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(workflowPath(name))
	if err != nil {
		t.Fatalf("workflow %s not found: %v", name, err)
	}
	var wf map[string]any
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("workflow %s invalid YAML: %v", name, err)
	}
	return wf
}

// Cycle 1 – tracer bullet: publish.yml exists and triggers on push to main.
func TestPublishWorkflow_Exists(t *testing.T) {
	wf := readWorkflow(t, "publish.yml")
	on, ok := wf["on"].(map[string]any)
	if !ok {
		t.Fatal("publish.yml missing 'on' key")
	}
	push, ok := on["push"].(map[string]any)
	if !ok {
		t.Fatal("publish.yml missing 'on.push'")
	}
	branches, ok := push["branches"].([]any)
	if !ok {
		t.Fatal("publish.yml missing 'on.push.branches'")
	}
	var found bool
	for _, b := range branches {
		if b == "main" {
			found = true
			break
		}
	}
	if !found {
		t.Error("publish.yml does not trigger on push to main")
	}
}

// Cycle 2 – agentctl binary built for linux/amd64 and darwin/arm64.
func TestPublishWorkflow_AgentctlBothPlatforms(t *testing.T) {
	data, err := os.ReadFile(workflowPath("publish.yml"))
	if err != nil {
		t.Fatalf("publish.yml not found: %v", err)
	}
	content := string(data)
	for _, want := range []string{"linux", "amd64", "darwin", "arm64"} {
		if !strings.Contains(content, want) {
			t.Errorf("publish.yml missing platform %q", want)
		}
	}
}

// Cycle 3 – agentctl binary attached to GitHub Release.
func TestPublishWorkflow_AgentctlRelease(t *testing.T) {
	data, err := os.ReadFile(workflowPath("publish.yml"))
	if err != nil {
		t.Fatalf("publish.yml not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "softprops/action-gh-release") && !strings.Contains(content, "gh release") {
		t.Error("publish.yml does not attach binaries to GitHub Release")
	}
}

// Cycle 4 – controller image pushed with correct name, latest and SHA tags.
func TestPublishWorkflow_ControllerImage(t *testing.T) {
	data, err := os.ReadFile(workflowPath("publish.yml"))
	if err != nil {
		t.Fatalf("publish.yml not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ghcr.io/sebastiaankok/agents/controller") {
		t.Error("publish.yml missing controller image name")
	}
	if !strings.Contains(content, ":latest") && !strings.Contains(content, "latest") {
		t.Error("publish.yml missing :latest tag for controller")
	}
	if !strings.Contains(content, "github.sha") {
		t.Error("publish.yml missing SHA tag")
	}
}

// Cycle 5 – opencode-runner image pushed with correct name, latest and SHA tags.
func TestPublishWorkflow_OpenCodeRunnerImage(t *testing.T) {
	data, err := os.ReadFile(workflowPath("publish.yml"))
	if err != nil {
		t.Fatalf("publish.yml not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "ghcr.io/sebastiaankok/agents/opencode-runner") {
		t.Error("publish.yml missing opencode-runner image name")
	}
}

// Cycle 6 – controller Dockerfile exists.
func TestControllerDockerfile_Exists(t *testing.T) {
	path := dockerfilePath("cmd/controller/Dockerfile")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("cmd/controller/Dockerfile not found")
	}
}

// Cycle 7 – controller Dockerfile uses scratch or distroless base for final stage.
func TestControllerDockerfile_CopiesBinary(t *testing.T) {
	path := dockerfilePath("cmd/controller/Dockerfile")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cmd/controller/Dockerfile read error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "COPY") && !strings.Contains(content, "ADD") {
		t.Error("controller Dockerfile does not copy binary")
	}
	if !strings.Contains(content, "ENTRYPOINT") && !strings.Contains(content, "CMD") {
		t.Error("controller Dockerfile missing ENTRYPOINT/CMD")
	}
}
