package k8s

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type DeployResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Namespace string `json:"namespace"`
	URL       string `json:"url"`
}

func (c *Client) Deploy(ctx context.Context, cfg *AppConfig) (*DeployResult, error) {
	cfg.Namespace = c.NamespaceForApp(cfg.Name)

	if err := c.ensureNamespace(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	if err := c.applySecret(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply secret: %w", err)
	}

	if err := c.applyDeployment(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply deployment: %w", err)
	}

	if err := c.applyService(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply service: %w", err)
	}

	if err := c.applyIngress(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply ingress: %w", err)
	}

	if err := c.waitForDeployment(ctx, cfg); err != nil {
		return &DeployResult{
			Success:   false,
			Message:   fmt.Sprintf("deployment did not become ready: %v", err),
			Namespace: cfg.Namespace,
		}, nil
	}

	url := fmt.Sprintf("https://%s.%s", cfg.Name, cfg.DomainSuffix)
	if cfg.Domain != "" {
		url = fmt.Sprintf("https://%s", cfg.Domain)
	}

	return &DeployResult{
		Success:   true,
		Message:   "deployment successful",
		Namespace: cfg.Namespace,
		URL:       url,
	}, nil
}

func (c *Client) ensureNamespace(ctx context.Context, cfg *AppConfig) error {
	ns := GenerateNamespace(cfg)

	_, err := c.clientset.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
	if err == nil {
		return nil
	}

	if k8serrors.IsNotFound(err) {
		_, err = c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		return err
	}

	return err
}

func (c *Client) applySecret(ctx context.Context, cfg *AppConfig) error {
	secret := GenerateSecret(cfg)
	secrets := c.clientset.CoreV1().Secrets(cfg.Namespace)

	existing, err := secrets.Get(ctx, secret.Name, metav1.GetOptions{})
	if err == nil {
		secret.ResourceVersion = existing.ResourceVersion
		_, err = secrets.Update(ctx, secret, metav1.UpdateOptions{})
		return err
	}

	if k8serrors.IsNotFound(err) {
		_, err = secrets.Create(ctx, secret, metav1.CreateOptions{})
		return err
	}

	return err
}

func (c *Client) applyDeployment(ctx context.Context, cfg *AppConfig) error {
	deployment := GenerateDeployment(cfg)
	deployments := c.clientset.AppsV1().Deployments(cfg.Namespace)

	existing, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{})
	if err == nil {
		deployment.ResourceVersion = existing.ResourceVersion
		_, err = deployments.Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	}

	if k8serrors.IsNotFound(err) {
		_, err = deployments.Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}

	return err
}

func (c *Client) applyService(ctx context.Context, cfg *AppConfig) error {
	service := GenerateService(cfg)
	services := c.clientset.CoreV1().Services(cfg.Namespace)

	existing, err := services.Get(ctx, service.Name, metav1.GetOptions{})
	if err == nil {
		service.ResourceVersion = existing.ResourceVersion
		service.Spec.ClusterIP = existing.Spec.ClusterIP
		_, err = services.Update(ctx, service, metav1.UpdateOptions{})
		return err
	}

	if k8serrors.IsNotFound(err) {
		_, err = services.Create(ctx, service, metav1.CreateOptions{})
		return err
	}

	return err
}

func (c *Client) applyIngress(ctx context.Context, cfg *AppConfig) error {
	ingress := GenerateIngress(cfg)
	ingresses := c.clientset.NetworkingV1().Ingresses(cfg.Namespace)

	existing, err := ingresses.Get(ctx, ingress.Name, metav1.GetOptions{})
	if err == nil {
		ingress.ResourceVersion = existing.ResourceVersion
		_, err = ingresses.Update(ctx, ingress, metav1.UpdateOptions{})
		return err
	}

	if k8serrors.IsNotFound(err) {
		_, err = ingresses.Create(ctx, ingress, metav1.CreateOptions{})
		return err
	}

	return err
}

func (c *Client) waitForDeployment(ctx context.Context, cfg *AppConfig) error {
	return wait.PollUntilContextTimeout(ctx, 2*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		deployment, err := c.clientset.AppsV1().Deployments(cfg.Namespace).Get(ctx, cfg.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		if deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas {
			return true, nil
		}

		return false, nil
	})
}

func (c *Client) DeleteApp(ctx context.Context, appName string) error {
	namespace := c.NamespaceForApp(appName)
	return c.clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (c *Client) GetDeploymentStatus(ctx context.Context, appName string) (*appsv1.Deployment, error) {
	namespace := c.NamespaceForApp(appName)
	return c.clientset.AppsV1().Deployments(namespace).Get(ctx, appName, metav1.GetOptions{})
}

func (c *Client) GetPods(ctx context.Context, appName string) (*corev1.PodList, error) {
	namespace := c.NamespaceForApp(appName)
	return c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", appName),
	})
}

func (c *Client) GetIngress(ctx context.Context, appName string) (*networkingv1.Ingress, error) {
	namespace := c.NamespaceForApp(appName)
	return c.clientset.NetworkingV1().Ingresses(namespace).Get(ctx, appName, metav1.GetOptions{})
}
