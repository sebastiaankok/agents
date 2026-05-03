package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agentctl",
		Short: "Dispatch opencode agent jobs to work on GitHub Issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
