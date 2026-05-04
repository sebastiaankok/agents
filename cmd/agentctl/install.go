package main

import (
	"fmt"

	k8sinternal "github.com/sebastiaankok/agents/internal/k8s"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func newInstallCmd() *cobra.Command {
	var namespace string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Bootstrap agentctl Kubernetes resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
				clientcmd.NewDefaultClientConfigLoadingRules(),
				&clientcmd.ConfigOverrides{},
			).ClientConfig()
			if err != nil {
				return fmt.Errorf("no cluster reachable: %w", err)
			}

			client, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				return fmt.Errorf("no cluster reachable: %w", err)
			}

			installer := k8sinternal.NewInstaller(client, namespace)
			if err := installer.Install(cmd.Context()); err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "installed agentctl resources in namespace %q\n", namespace)
			return nil
		},
	}

	cmd.Flags().StringVar(&namespace, "namespace", "agent-runners", "target namespace")
	return cmd
}
