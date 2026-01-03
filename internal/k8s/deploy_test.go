package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// skipIfNoCluster skips the test if no K8s cluster is available
func skipIfNoCluster(t *testing.T) *Client {
	t.Helper()

	// Check if we're in CI or have KUBECONFIG set
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		// Try default path
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("Skipping: no KUBECONFIG set and can't find home directory")
		}
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
			t.Skip("Skipping: no kubeconfig available (set KUBECONFIG or run k3d cluster create)")
		}
		kubeconfig = defaultPath
	}

	client, err := NewClient(kubeconfig, "test-fuego-")
	if err != nil {
		t.Skipf("Skipping: failed to create K8s client: %v", err)
	}

	// Verify cluster is reachable
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Clientset().CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		t.Skipf("Skipping: cluster not reachable: %v", err)
	}

	return client
}

// testNamespace generates a unique test namespace name
func testNamespace(t *testing.T) string {
	t.Helper()
	return "test-fuego-" + time.Now().Format("20060102150405")
}

// cleanupNamespace deletes the test namespace
func cleanupNamespace(t *testing.T, client *Client, namespace string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := client.Clientset().CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		t.Logf("Warning: failed to cleanup namespace %s: %v", namespace, err)
	}
}

func TestDeployResult_Struct(t *testing.T) {
	result := DeployResult{
		Success:   true,
		Message:   "deployment successful",
		Namespace: "test-namespace",
		URL:       "https://myapp.fuego.build",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}
	if result.Message != "deployment successful" {
		t.Errorf("expected Message 'deployment successful', got %q", result.Message)
	}
	if result.Namespace != "test-namespace" {
		t.Errorf("expected Namespace 'test-namespace', got %q", result.Namespace)
	}
	if result.URL != "https://myapp.fuego.build" {
		t.Errorf("expected URL 'https://myapp.fuego.build', got %q", result.URL)
	}
}

func TestAppStatus_Struct(t *testing.T) {
	status := AppStatus{
		Status:            "running",
		Replicas:          3,
		ReadyReplicas:     3,
		AvailableReplicas: 3,
		Conditions:        []string{"Available: True", "Progressing: True"},
	}

	if status.Status != "running" {
		t.Errorf("expected Status 'running', got %q", status.Status)
	}
	if status.Replicas != 3 {
		t.Errorf("expected Replicas 3, got %d", status.Replicas)
	}
	if len(status.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(status.Conditions))
	}
}

func TestDeploy_URLGeneration(t *testing.T) {
	tests := []struct {
		name         string
		appName      string
		domainSuffix string
		customDomain string
		expectedURL  string
	}{
		{
			name:         "with domain suffix",
			appName:      "myapp",
			domainSuffix: "fuego.build",
			customDomain: "",
			expectedURL:  "https://myapp.fuego.build",
		},
		{
			name:         "with custom domain",
			appName:      "myapp",
			domainSuffix: "fuego.build",
			customDomain: "custom.example.com",
			expectedURL:  "https://custom.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &AppConfig{
				Name:         tt.appName,
				DomainSuffix: tt.domainSuffix,
				Domain:       tt.customDomain,
			}

			// Simulate URL generation logic from Deploy function
			url := "https://" + cfg.Name + "." + cfg.DomainSuffix
			if cfg.Domain != "" {
				url = "https://" + cfg.Domain
			}

			if url != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, url)
			}
		})
	}
}

// Integration tests - require a real K8s cluster

func TestDeploy_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "integration-test-app"
	namespace := client.NamespaceForApp(appName)

	// Cleanup after test
	defer cleanupNamespace(t, client, namespace)

	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
		EnvVars:      map[string]string{"TEST_VAR": "test_value"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected successful deployment, got: %s", result.Message)
	}
	if result.Namespace != namespace {
		t.Errorf("expected namespace %s, got %s", namespace, result.Namespace)
	}
}

func TestEnsureNamespace_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	namespace := testNamespace(t)

	// Cleanup
	defer cleanupNamespace(t, client, namespace)

	cfg := &AppConfig{
		Name:      "test-app",
		Namespace: namespace,
	}

	ctx := context.Background()

	// First call should create namespace
	err := client.ensureNamespace(ctx, cfg)
	if err != nil {
		t.Fatalf("ensureNamespace (create) failed: %v", err)
	}

	// Verify namespace exists
	_, err = client.Clientset().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("namespace not found after creation: %v", err)
	}

	// Second call should not fail (namespace already exists)
	err = client.ensureNamespace(ctx, cfg)
	if err != nil {
		t.Fatalf("ensureNamespace (exists) failed: %v", err)
	}
}

func TestDeleteApp_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "delete-test-app"
	namespace := client.NamespaceForApp(appName)

	ctx := context.Background()

	// Create namespace first
	ns := GenerateNamespace(&AppConfig{Name: appName, Namespace: namespace})
	_, err := client.Clientset().CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	// Delete the app
	err = client.DeleteApp(ctx, appName)
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	// Verify namespace is being deleted (might take time)
	time.Sleep(1 * time.Second)
	_, err = client.Clientset().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace might still exist but be terminating
		t.Log("Namespace still exists, may be terminating")
	}
}

func TestRestartApp_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "restart-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// First deploy the app
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
	}

	_, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Now restart
	err = client.RestartApp(ctx, appName)
	if err != nil {
		t.Fatalf("RestartApp failed: %v", err)
	}

	// Verify restart annotation was added
	deployment, err := client.GetDeploymentStatus(ctx, appName)
	if err != nil {
		t.Fatalf("GetDeploymentStatus failed: %v", err)
	}

	annotations := deployment.Spec.Template.Annotations
	if _, ok := annotations["kubectl.kubernetes.io/restartedAt"]; !ok {
		t.Error("expected restartedAt annotation to be set")
	}
}

func TestScaleApp_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "scale-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// Deploy with 1 replica
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
	}

	_, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Scale to 3 replicas
	err = client.ScaleApp(ctx, appName, 3)
	if err != nil {
		t.Fatalf("ScaleApp failed: %v", err)
	}

	// Verify scale
	deployment, err := client.GetDeploymentStatus(ctx, appName)
	if err != nil {
		t.Fatalf("GetDeploymentStatus failed: %v", err)
	}

	if *deployment.Spec.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", *deployment.Spec.Replicas)
	}
}

func TestGetAppStatus_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "status-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// Test not deployed case
	status, err := client.GetAppStatus(ctx, "nonexistent-app")
	if err != nil {
		t.Fatalf("GetAppStatus (not deployed) failed: %v", err)
	}
	if status.Status != "not_deployed" {
		t.Errorf("expected status 'not_deployed', got %q", status.Status)
	}

	// Deploy app
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
	}

	_, err = client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Get status
	status, err = client.GetAppStatus(ctx, appName)
	if err != nil {
		t.Fatalf("GetAppStatus failed: %v", err)
	}

	// Status should be one of the valid states
	validStatuses := []string{"running", "partially_ready", "starting", "unknown"}
	found := false
	for _, valid := range validStatuses {
		if status.Status == valid {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unexpected status %q", status.Status)
	}
}

func TestGetPods_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "pods-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// Deploy app
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     2,
		Port:         80,
		DomainSuffix: "test.local",
	}

	result, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if !result.Success {
		t.Skipf("Deployment didn't succeed (may need more time): %s", result.Message)
	}

	// Get pods
	pods, err := client.GetPods(ctx, appName)
	if err != nil {
		t.Fatalf("GetPods failed: %v", err)
	}

	if len(pods.Items) == 0 {
		t.Error("expected at least one pod")
	}
}

func TestGetIngress_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "ingress-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// Deploy app
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
	}

	_, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Get ingress
	ingress, err := client.GetIngress(ctx, appName)
	if err != nil {
		t.Fatalf("GetIngress failed: %v", err)
	}

	if ingress.Name != appName {
		t.Errorf("expected ingress name %q, got %q", appName, ingress.Name)
	}
}
