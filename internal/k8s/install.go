package k8s

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	controllerName  = "agentctl-controller"
	controllerImage = "ghcr.io/sebastiaankok/agents/controller:latest"
)

type Installer struct {
	client    kubernetes.Interface
	namespace string
}

func NewInstaller(client kubernetes.Interface, namespace string) *Installer {
	return &Installer{client: client, namespace: namespace}
}

func (i *Installer) Install(ctx context.Context) error {
	if err := i.applyNamespace(ctx); err != nil {
		return fmt.Errorf("namespace: %w", err)
	}
	if err := i.applyServiceAccount(ctx); err != nil {
		return fmt.Errorf("service account: %w", err)
	}
	if err := i.applyClusterRole(ctx); err != nil {
		return fmt.Errorf("cluster role: %w", err)
	}
	if err := i.applyClusterRoleBinding(ctx); err != nil {
		return fmt.Errorf("cluster role binding: %w", err)
	}
	if err := i.applyDeployment(ctx); err != nil {
		return fmt.Errorf("deployment: %w", err)
	}
	return nil
}

func (i *Installer) applyNamespace(ctx context.Context) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: i.namespace}}
	_, err := i.client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (i *Installer) applyServiceAccount(ctx context.Context) error {
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: controllerName, Namespace: i.namespace}}
	_, err := i.client.CoreV1().ServiceAccounts(i.namespace).Create(ctx, sa, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (i *Installer) applyClusterRole(ctx context.Context) error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: controllerName},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets", "configmaps"},
				Verbs:     []string{"get"},
			},
		},
	}
	_, err := i.client.RbacV1().ClusterRoles().Create(ctx, cr, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (i *Installer) applyClusterRoleBinding(ctx context.Context) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: controllerName},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     controllerName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      controllerName,
				Namespace: i.namespace,
			},
		},
	}
	_, err := i.client.RbacV1().ClusterRoleBindings().Create(ctx, crb, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (i *Installer) applyDeployment(ctx context.Context) error {
	replicas := int32(1)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: controllerName, Namespace: i.namespace},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": controllerName},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": controllerName}},
				Spec: corev1.PodSpec{
					ServiceAccountName: controllerName,
					Containers: []corev1.Container{
						{Name: "controller", Image: controllerImage},
					},
				},
			},
		},
	}
	_, err := i.client.AppsV1().Deployments(i.namespace).Create(ctx, dep, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}
	return err
}
