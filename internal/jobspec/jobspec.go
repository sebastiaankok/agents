package jobspec

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DefaultImage      = "ghcr.io/sebastiaankok/agents/opencode-runner:latest"
	opencodeConfigMap = "opencode-config"
	opencodeConfigDir = "/root/.config/opencode"
	opencodeConfigKey = "config.json"
	ttlSeconds        = int32(604800)
)

type Issue struct {
	Number int
	Body   string
}

type Config struct {
	Image         string
	GitHubToken   string
	RepoURL       string
	DefaultBranch string
	Namespace     string
}

func Build(issue Issue, cfg Config) *batchv1.Job {
	image := cfg.Image
	if image == "" {
		image = DefaultImage
	}

	issueStr := fmt.Sprintf("%d", issue.Number)
	ttl := ttlSeconds
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-job-" + issueStr,
			Namespace: cfg.Namespace,
			Labels:    map[string]string{"issue-number": issueStr},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "runner",
							Image: image,
							Env: []corev1.EnvVar{
								{Name: "ISSUE_NUMBER", Value: fmt.Sprintf("%d", issue.Number)},
								{Name: "GITHUB_TOKEN", Value: cfg.GitHubToken},
								{Name: "REPO_URL", Value: cfg.RepoURL},
								{Name: "DEFAULT_BRANCH", Value: cfg.DefaultBranch},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opencode-config",
									MountPath: opencodeConfigDir + "/" + opencodeConfigKey,
									SubPath:   opencodeConfigKey,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "opencode-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: opencodeConfigMap,
									},
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}
