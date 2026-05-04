package jobspec

import (
	"fmt"
	"strconv"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sebastiaankok/agents/internal/github"
)

const (
	DefaultJobImage      = "ghcr.io/sebastiaankok/agents/opencode-runner:latest"
	DefaultConfigMapName = "opencode-config"
	credentialsSecret    = "agentctl-credentials"
	opencodeConfigDir    = "/root/.config/opencode"
	jobTTL               = int32(604800)
)

// Config controls Job creation parameters.
type Config struct {
	JobImage           string
	ConfigMapName      string
	ServiceAccountName string
	Namespace          string
	MaxParallel        int
}

// Build returns a suspended Kubernetes Job spec for the given issue.
func Build(issue github.Issue, repoURL, defaultBranch string, cfg Config) *batchv1.Job {
	image := cfg.JobImage
	if image == "" {
		image = DefaultJobImage
	}
	cmName := cfg.ConfigMapName
	if cmName == "" {
		cmName = DefaultConfigMapName
	}

	suspend := true
	ttl := jobTTL

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("agent-issue-%d", issue.Number),
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"issue-number": fmt.Sprintf("%d", issue.Number),
			},
			Annotations: map[string]string{
				"agentctl.max-parallel": strconv.Itoa(cfg.MaxParallel),
			},
		},
		Spec: batchv1.JobSpec{
			Suspend:                 &suspend,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: cfg.ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "runner",
							Image: image,
							Env: []corev1.EnvVar{
								{Name: "ISSUE_NUMBER", Value: fmt.Sprintf("%d", issue.Number)},
								{
									Name: "GITHUB_TOKEN",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: credentialsSecret},
											Key:                  "GITHUB_TOKEN",
										},
									},
								},
								{Name: "REPO_URL", Value: repoURL},
								{Name: "DEFAULT_BRANCH", Value: defaultBranch},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opencode-config",
									MountPath: opencodeConfigDir,
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "opencode-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
					},
				},
			},
		},
	}
}
