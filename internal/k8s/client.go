package k8s

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Client wraps the Kubernetes API for agent Job operations.
type Client struct {
	kube kubernetes.Interface
}

// NewClient returns a Client backed by the given kubernetes.Interface.
func NewClient(kube kubernetes.Interface) *Client {
	return &Client{kube: kube}
}

// CreateJob creates the Job in the given namespace.
func (c *Client) CreateJob(ctx context.Context, namespace string, job *batchv1.Job) error {
	_, err := c.kube.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}

// ListJobs returns Jobs in namespace matching labelSelector.
func (c *Client) ListJobs(ctx context.Context, namespace, labelSelector string) ([]batchv1.Job, error) {
	list, err := c.kube.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// JobExists returns true if a Job with label issue-number=<issueNumber> exists in namespace.
func (c *Client) JobExists(ctx context.Context, namespace string, issueNumber int) (bool, error) {
	selector := fmt.Sprintf("issue-number=%d", issueNumber)
	jobs, err := c.ListJobs(ctx, namespace, selector)
	if err != nil {
		return false, err
	}
	return len(jobs) > 0, nil
}

// UnsuspendJob sets spec.suspend=false on the named Job.
func (c *Client) UnsuspendJob(ctx context.Context, namespace, name string) error {
	job, err := c.kube.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	f := false
	job.Spec.Suspend = &f
	_, err = c.kube.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	return err
}

// SetJobAnnotation updates a single annotation on the named Job.
func (c *Client) SetJobAnnotation(ctx context.Context, namespace, name, key, value string) error {
	job, err := c.kube.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if job.Annotations == nil {
		job.Annotations = make(map[string]string)
	}
	job.Annotations[key] = value
	_, err = c.kube.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
	return err
}

// ValidateSecret returns an error if the named Secret does not exist in namespace.
func (c *Client) ValidateSecret(ctx context.Context, namespace, name string) error {
	_, err := c.kube.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return fmt.Errorf("secret %q not found in namespace %q", name, namespace)
	}
	return err
}

// ValidateConfigMap returns an error if the named ConfigMap does not exist in namespace.
func (c *Client) ValidateConfigMap(ctx context.Context, namespace, name string) error {
	_, err := c.kube.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		return fmt.Errorf("configmap %q not found in namespace %q", name, namespace)
	}
	return err
}
