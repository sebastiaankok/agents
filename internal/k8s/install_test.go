package k8s_test

import (
	"context"
	"testing"

	k8sinternal "github.com/sebastiaankok/agents/internal/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInstall_CreatesNamespace(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	ns, err := client.CoreV1().Namespaces().Get(context.Background(), "agent-runners", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("namespace not created: %v", err)
	}
	if ns.Name != "agent-runners" {
		t.Errorf("namespace name = %q, want %q", ns.Name, "agent-runners")
	}
}

func TestInstall_CreatesServiceAccount(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	sa, err := client.CoreV1().ServiceAccounts("agent-runners").Get(context.Background(), "agentctl-controller", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service account not created: %v", err)
	}
	if sa.Name != "agentctl-controller" {
		t.Errorf("service account name = %q, want %q", sa.Name, "agentctl-controller")
	}
}

func TestInstall_CreatesClusterRole(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	cr, err := client.RbacV1().ClusterRoles().Get(context.Background(), "agentctl-controller", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("cluster role not created: %v", err)
	}

	ruleMap := make(map[string][]string)
	for _, rule := range cr.Rules {
		for _, resource := range rule.Resources {
			ruleMap[resource] = rule.Verbs
		}
	}

	for _, verb := range []string{"get", "list", "watch", "update"} {
		if !contains(ruleMap["jobs"], verb) {
			t.Errorf("jobs rule missing verb %q", verb)
		}
	}
	for _, resource := range []string{"secrets", "configmaps"} {
		if !contains(ruleMap[resource], "get") {
			t.Errorf("%s rule missing verb %q", resource, "get")
		}
	}
}

func TestInstall_CreatesClusterRoleBinding(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	crb, err := client.RbacV1().ClusterRoleBindings().Get(context.Background(), "agentctl-controller", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("cluster role binding not created: %v", err)
	}
	if crb.RoleRef.Name != "agentctl-controller" {
		t.Errorf("role ref = %q, want %q", crb.RoleRef.Name, "agentctl-controller")
	}
	if len(crb.Subjects) == 0 || crb.Subjects[0].Name != "agentctl-controller" {
		t.Errorf("binding subject missing or wrong: %+v", crb.Subjects)
	}
}

func TestInstall_CreatesDeployment(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	dep, err := client.AppsV1().Deployments("agent-runners").Get(context.Background(), "agentctl-controller", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment not created: %v", err)
	}
	containers := dep.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		t.Fatal("deployment has no containers")
	}
	wantImage := "ghcr.io/sebastiaankok/agents/controller:latest"
	if containers[0].Image != wantImage {
		t.Errorf("image = %q, want %q", containers[0].Image, wantImage)
	}
}

func TestInstall_Idempotent(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "agent-runners")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("first Install() error = %v", err)
	}
	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("second Install() error = %v", err)
	}
}

func TestInstall_CustomNamespace(t *testing.T) {
	client := fake.NewSimpleClientset()
	installer := k8sinternal.NewInstaller(client, "custom-ns")

	if err := installer.Install(context.Background()); err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if _, err := client.CoreV1().Namespaces().Get(context.Background(), "custom-ns", metav1.GetOptions{}); err != nil {
		t.Fatalf("custom namespace not created: %v", err)
	}
	if _, err := client.CoreV1().ServiceAccounts("custom-ns").Get(context.Background(), "agentctl-controller", metav1.GetOptions{}); err != nil {
		t.Fatalf("service account not in custom namespace: %v", err)
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// ensure corev1 import used
var _ = corev1.Namespace{}
