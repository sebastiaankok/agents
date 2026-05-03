package main

import (
	"github.com/sebastiaankok/agents/internal/jobspec"
	"github.com/spf13/cobra"
)

func newRunCmd(onConfig func(jobspec.Config)) *cobra.Command {
	var cfg jobspec.Config

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Fetch ready-for-agent issues and create Kubernetes Jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if onConfig != nil {
				onConfig(cfg)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.JobImage, "job-image", "", "override the default runner image")

	return cmd
}
