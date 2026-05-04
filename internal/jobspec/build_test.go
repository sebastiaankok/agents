package jobspec_test

import (
	"fmt"
	"testing"

	"github.com/sebastiaankok/agents/internal/github"
	"github.com/sebastiaankok/agents/internal/jobspec"
)

func issue(n int) github.Issue {
	return github.Issue{Number: n, Title: "fix something", Body: "body"}
}

func cfg() jobspec.Config {
	return jobspec.Config{
		Namespace: "agent-runners",
	}
}

func TestBuild_Name(t *testing.T) {
	job := jobspec.Build(issue(42), "https://github.com/org/repo", "main", cfg())
	if job == nil {
		t.Fatal("Build returned nil")
	}
	want := "agent-issue-42"
	if job.Name != want {
		t.Errorf("Name = %q, want %q", job.Name, want)
	}
}

func TestBuild_Suspend(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	if job.Spec.Suspend == nil || !*job.Spec.Suspend {
		t.Error("spec.suspend must be true")
	}
}

func TestBuild_TTL(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	if job.Spec.TTLSecondsAfterFinished == nil || *job.Spec.TTLSecondsAfterFinished != 604800 {
		t.Errorf("TTLSecondsAfterFinished = %v, want 604800", job.Spec.TTLSecondsAfterFinished)
	}
}

func TestBuild_IssueLabel(t *testing.T) {
	job := jobspec.Build(issue(99), "https://github.com/org/repo", "main", cfg())
	got := job.Labels["issue-number"]
	if got != "99" {
		t.Errorf("label issue-number = %q, want %q", got, "99")
	}
}

func TestBuild_EnvVars(t *testing.T) {
	job := jobspec.Build(issue(7), "https://github.com/org/repo", "develop", cfg())
	envMap := make(map[string]string)
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Value != "" {
			envMap[e.Name] = e.Value
		}
	}
	cases := map[string]string{
		"ISSUE_NUMBER":   "7",
		"REPO_URL":       "https://github.com/org/repo",
		"DEFAULT_BRANCH": "develop",
	}
	for k, v := range cases {
		if envMap[k] != v {
			t.Errorf("env %s = %q, want %q", k, envMap[k], v)
		}
	}
}

func TestBuild_GitHubTokenFromSecret(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	var tokenEnv *struct{ name, secretName, key string }
	for _, e := range job.Spec.Template.Spec.Containers[0].Env {
		if e.Name == "GITHUB_TOKEN" && e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			tokenEnv = &struct{ name, secretName, key string }{
				name:       e.Name,
				secretName: e.ValueFrom.SecretKeyRef.Name,
				key:        e.ValueFrom.SecretKeyRef.Key,
			}
		}
	}
	if tokenEnv == nil {
		t.Fatal("GITHUB_TOKEN env var with SecretKeyRef not found")
	}
	if tokenEnv.secretName != "agentctl-credentials" {
		t.Errorf("secret name = %q, want %q", tokenEnv.secretName, "agentctl-credentials")
	}
	if tokenEnv.key != "GITHUB_TOKEN" {
		t.Errorf("secret key = %q, want %q", tokenEnv.key, "GITHUB_TOKEN")
	}
}

func TestBuild_ConfigMapMount(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	containers := job.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("no containers")
	}
	var found bool
	for _, vm := range containers[0].VolumeMounts {
		if vm.MountPath == "/root/.config/opencode" {
			found = true
		}
	}
	if !found {
		t.Error("no volume mount at /root/.config/opencode")
	}
	for _, v := range job.Spec.Template.Spec.Volumes {
		if v.ConfigMap != nil && v.ConfigMap.Name == "opencode-config" {
			return
		}
	}
	t.Error("no volume referencing opencode-config ConfigMap")
}

func TestBuild_DefaultImage(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	got := job.Spec.Template.Spec.Containers[0].Image
	if got != jobspec.DefaultJobImage {
		t.Errorf("Image = %q, want %q", got, jobspec.DefaultJobImage)
	}
}

func TestBuild_CustomImage(t *testing.T) {
	c := cfg()
	c.JobImage = "my-registry/custom:v1"
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", c)
	got := job.Spec.Template.Spec.Containers[0].Image
	if got != "my-registry/custom:v1" {
		t.Errorf("Image = %q, want %q", got, "my-registry/custom:v1")
	}
}

func TestBuild_CustomConfigMap(t *testing.T) {
	c := cfg()
	c.ConfigMapName = "my-opencode-config"
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", c)
	for _, v := range job.Spec.Template.Spec.Volumes {
		if v.ConfigMap != nil && v.ConfigMap.Name == "my-opencode-config" {
			return
		}
	}
	t.Errorf("no volume referencing ConfigMap %q", "my-opencode-config")
}

func TestBuild_Namespace(t *testing.T) {
	job := jobspec.Build(issue(5), "https://github.com/org/repo", "main", cfg())
	if job.Namespace != "agent-runners" {
		t.Errorf("Namespace = %q, want %q", job.Namespace, "agent-runners")
	}
}

func TestBuild_MaxParallelAnnotation(t *testing.T) {
	c := cfg()
	c.MaxParallel = 5
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", c)
	got := job.Annotations["agentctl.max-parallel"]
	if got != "5" {
		t.Errorf("annotation agentctl.max-parallel = %q, want %q", got, "5")
	}
}

func TestBuild_MaxParallelDefault(t *testing.T) {
	job := jobspec.Build(issue(1), "https://github.com/org/repo", "main", cfg())
	got := job.Annotations["agentctl.max-parallel"]
	if got != "0" {
		t.Errorf("annotation agentctl.max-parallel = %q, want %q (zero value)", got, "0")
	}
}

// keep compiler happy
var _ = fmt.Sprintf
