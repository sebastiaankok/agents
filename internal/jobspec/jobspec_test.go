package jobspec_test

import (
	"testing"

	"github.com/sebastiaankok/agents/internal/jobspec"
	corev1 "k8s.io/api/core/v1"
)

const defaultImage = "ghcr.io/sebastiaankok/agents/opencode-runner:latest"

func TestBuild_CustomImage(t *testing.T) {
	job := jobspec.Build(jobspec.Issue{Number: 7}, jobspec.Config{Image: "custom:v1"})

	got := job.Spec.Template.Spec.Containers[0].Image
	if got != "custom:v1" {
		t.Errorf("image = %q, want %q", got, "custom:v1")
	}
}

func TestBuild_EnvVars(t *testing.T) {
	issue := jobspec.Issue{Number: 42}
	cfg := jobspec.Config{
		GitHubToken:   "tok",
		RepoURL:       "https://github.com/org/repo",
		DefaultBranch: "main",
	}

	job := jobspec.Build(issue, cfg)
	envs := envMap(job.Spec.Template.Spec.Containers[0].Env)

	cases := map[string]string{
		"ISSUE_NUMBER":   "42",
		"GITHUB_TOKEN":   "tok",
		"REPO_URL":       "https://github.com/org/repo",
		"DEFAULT_BRANCH": "main",
	}
	for k, want := range cases {
		if got := envs[k]; got != want {
			t.Errorf("env %s = %q, want %q", k, got, want)
		}
	}
}

func envMap(envs []corev1.EnvVar) map[string]string {
	m := make(map[string]string, len(envs))
	for _, e := range envs {
		m[e.Name] = e.Value
	}
	return m
}

func TestBuild_ConfigMapMount(t *testing.T) {
	job := jobspec.Build(jobspec.Issue{Number: 1}, jobspec.Config{})

	spec := job.Spec.Template.Spec

	// volume must exist
	var volFound bool
	for _, v := range spec.Volumes {
		if v.ConfigMap != nil && v.ConfigMap.LocalObjectReference.Name == "opencode-config" {
			volFound = true
		}
	}
	if !volFound {
		t.Error("no volume for opencode-config ConfigMap")
	}

	// container must mount it at the opencode config path
	const wantPath = "/root/.config/opencode/config.json"
	var mountFound bool
	for _, vm := range spec.Containers[0].VolumeMounts {
		if vm.MountPath == wantPath {
			mountFound = true
		}
	}
	if !mountFound {
		t.Errorf("no VolumeMount at %q", wantPath)
	}
}

func TestBuild_TTL(t *testing.T) {
	job := jobspec.Build(jobspec.Issue{Number: 1}, jobspec.Config{})

	if job.Spec.TTLSecondsAfterFinished == nil {
		t.Fatal("TTLSecondsAfterFinished is nil")
	}
	const want int32 = 604800
	if *job.Spec.TTLSecondsAfterFinished != want {
		t.Errorf("TTL = %d, want %d", *job.Spec.TTLSecondsAfterFinished, want)
	}
}

func TestBuild_NameAndLabels(t *testing.T) {
	job := jobspec.Build(jobspec.Issue{Number: 42}, jobspec.Config{})

	if job.Name != "agent-job-42" {
		t.Errorf("name = %q, want %q", job.Name, "agent-job-42")
	}

	if got := job.Labels["issue-number"]; got != "42" {
		t.Errorf("label issue-number = %q, want %q", got, "42")
	}
}

func TestBuild_DefaultImage(t *testing.T) {
	job := jobspec.Build(jobspec.Issue{Number: 7}, jobspec.Config{})

	containers := job.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("want 1 container, got %d", len(containers))
	}
	if containers[0].Image != defaultImage {
		t.Errorf("image = %q, want %q", containers[0].Image, defaultImage)
	}
}
