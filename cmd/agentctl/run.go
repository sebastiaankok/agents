package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	gitinternal "github.com/sebastiaankok/agents/internal/git"
	"github.com/sebastiaankok/agents/internal/github"
	"github.com/sebastiaankok/agents/internal/jobspec"
	k8sinternal "github.com/sebastiaankok/agents/internal/k8s"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func newRunCmd(onConfig func(jobspec.Config)) *cobra.Command {
	var (
		cfg           jobspec.Config
		flIssue       int
		flNamespace   string
		flMaxParallel int
		flDryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Fetch ready-for-agent issues and create Kubernetes Jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Namespace = flNamespace
			if onConfig != nil {
				onConfig(cfg)
			}

			if flDryRun {
				return nil
			}

			kclient, err := newK8sClient()
			if err != nil {
				return err
			}

			c := k8sinternal.NewClient(kclient)

			if err := validatePrereqs(cmd.Context(), c, cfg.Namespace); err != nil {
				return err
			}

			repo, err := gitinternal.InferRepo()
			if err != nil {
				return fmt.Errorf("infer repo: %w", err)
			}

			defaultBranch, err := defaultBranch()
			if err != nil {
				return fmt.Errorf("get default branch: %w", err)
			}

			gc := github.NewClient("")

			var issues []github.Issue
			if flIssue > 0 {
				issue, err := gc.GetIssue(cmd.Context(), repo, flIssue)
				if err != nil {
					return fmt.Errorf("fetch issue %d: %w", flIssue, err)
				}
				issues = append(issues, issue)
			} else {
				issues, err = gc.GetReadyIssues(cmd.Context(), repo)
				if err != nil {
					return fmt.Errorf("fetch ready issues: %w", err)
				}
			}

			if len(issues) == 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no ready-for-agent issues found")
				return nil
			}

			created, skipped := createJobs(cmd.Context(), cmd.ErrOrStderr(), c, issues, repo, defaultBranch, cfg, flMaxParallel)

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\ncreated %d job(s), skipped %d (already active)\n", created, skipped)
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.JobImage, "job-image", "", "override the default runner image")
	cmd.Flags().IntVar(&flIssue, "issue", 0, "target a single issue by number (0 = all ready-for-agent)")
	cmd.Flags().StringVar(&flNamespace, "namespace", "agent-runners", "target namespace")
	cmd.Flags().IntVar(&flMaxParallel, "max-parallel", 3, "maximum concurrent jobs (stored as annotation)")
	cmd.Flags().BoolVar(&flDryRun, "dry-run", false, "parse flags only without executing")
	_ = cmd.Flags().MarkHidden("dry-run")

	return cmd
}

func newK8sClient() (*kubernetes.Clientset, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("no cluster reachable: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("no cluster reachable: %w", err)
	}
	return client, nil
}

func validatePrereqs(ctx context.Context, c *k8sinternal.Client, namespace string) error {
	if err := c.ValidateSecret(ctx, namespace, "agentctl-credentials"); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", err)
		_, _ = fmt.Fprintln(os.Stderr, secretYAMLExample(namespace))
		return fmt.Errorf("missing prerequisite: agentctl-credentials Secret")
	}

	if err := c.ValidateConfigMap(ctx, namespace, jobspec.DefaultConfigMapName); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n\n", err)
		_, _ = fmt.Fprintln(os.Stderr, configMapYAMLExample(namespace))
		return fmt.Errorf("missing prerequisite: %s ConfigMap", jobspec.DefaultConfigMapName)
	}

	return nil
}

func secretYAMLExample(namespace string) string {
	return fmt.Sprintf(`# Create the credentials Secret:
kubectl create secret generic agentctl-credentials \
  --from-literal=GITHUB_TOKEN=<your-token> \
  -n %s

# Or apply this YAML:
apiVersion: v1
kind: Secret
metadata:
  name: agentctl-credentials
  namespace: %s
type: Opaque
stringData:
  GITHUB_TOKEN: "<your-token>"`, namespace, namespace)
}

func configMapYAMLExample(namespace string) string {
	return fmt.Sprintf(`# Create the opencode-config ConfigMap. Choose ONE provider:

---
# Anthropic
apiVersion: v1
kind: ConfigMap
metadata:
  name: opencode-config
  namespace: %s
data:
  providers.json: |
    {
      "providers": [
        {
          "name": "anthropic",
          "apiKeyEnvar": "ANTHROPIC_API_KEY"
        }
      ]
    }

---
# OpenRouter
apiVersion: v1
kind: ConfigMap
metadata:
  name: opencode-config
  namespace: %s
data:
  providers.json: |
    {
      "providers": [
        {
          "name": "openrouter",
          "apiKeyEnvar": "OPENROUTER_API_KEY"
        }
      ]
    }

---
# LM Studio (local)
apiVersion: v1
kind: ConfigMap
metadata:
  name: opencode-config
  namespace: %s
data:
  providers.json: |
    {
      "providers": [
        {
          "name": "lm-studio",
          "baseUrl": "http://localhost:1211/v1"
        }
      ]
    }`, namespace, namespace, namespace)
}

func defaultBranch() (string, error) {
	out, err := exec.Command("git", "remote", "show", "origin").Output()
	if err != nil {
		return "main", nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "HEAD branch:") {
			branch := strings.TrimSpace(strings.TrimPrefix(trimmed, "HEAD branch:"))
			if branch != "" {
				return branch, nil
			}
		}
	}
	return "main", nil
}

func createJobs(ctx context.Context, w interface{ Write([]byte) (int, error) }, c *k8sinternal.Client, issues []github.Issue, repoURL, defaultBranch string, cfg jobspec.Config, maxParallel int) (created, skipped int) {
	for _, issue := range issues {
		exists, err := c.JobExists(ctx, cfg.Namespace, issue.Number)
		if err != nil {
			_, _ = fmt.Fprintf(w, "warning: could not check job for issue %d: %v\n", issue.Number, err)
			continue
		}
		if exists {
			_, _ = fmt.Fprintf(w, "skip issue %d (job already exists)\n", issue.Number)
			skipped++
			continue
		}

		jobCfg := cfg
		jobCfg.MaxParallel = maxParallel
		job := jobspec.Build(issue, repoURL, defaultBranch, jobCfg)

		if err := c.CreateJob(ctx, cfg.Namespace, job); err != nil {
			_, _ = fmt.Fprintf(w, "error creating job for issue %d: %v\n", issue.Number, err)
			continue
		}

		_, _ = fmt.Fprintf(w, "created job agent-issue-%d for \"%s\"\n", issue.Number, issue.Title)
		created++
	}
	return
}
