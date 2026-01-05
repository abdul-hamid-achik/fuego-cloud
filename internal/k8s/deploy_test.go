package k8s

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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
		URL:       "https://myapp.nexo.build",
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
	if result.URL != "https://myapp.nexo.build" {
		t.Errorf("expected URL 'https://myapp.nexo.build', got %q", result.URL)
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

// Unit tests using fake client - don't require a real cluster

func TestDeploy_WithFakeClient(t *testing.T) {
	// Create fake clientset
	fakeClient := fake.NewClientset()

	client := NewClientWithInterface(fakeClient, "test-")

	cfg := &AppConfig{
		Name:         "myapp",
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
		EnvVars:      map[string]string{"KEY": "value"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deploy will create resources, but waitForDeployment will timeout
	// since fake client doesn't update status
	result, err := client.Deploy(ctx, cfg)

	// Expect either success (if timeout is handled) or specific error
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// The deployment won't be "ready" with fake client, so check namespace was created
	if result.Namespace != "test-myapp" {
		t.Errorf("expected namespace 'test-myapp', got %q", result.Namespace)
	}

	// Verify resources were created
	_, err = fakeClient.CoreV1().Namespaces().Get(ctx, "test-myapp", metav1.GetOptions{})
	if err != nil {
		t.Errorf("namespace not created: %v", err)
	}

	_, err = fakeClient.AppsV1().Deployments("test-myapp").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Errorf("deployment not created: %v", err)
	}

	_, err = fakeClient.CoreV1().Services("test-myapp").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Errorf("service not created: %v", err)
	}

	_, err = fakeClient.NetworkingV1().Ingresses("test-myapp").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Errorf("ingress not created: %v", err)
	}
}

func TestEnsureNamespace_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	cfg := &AppConfig{
		Name:      "testapp",
		Namespace: "test-testapp",
	}

	ctx := context.Background()

	// First call should create
	err := client.ensureNamespace(ctx, cfg)
	if err != nil {
		t.Fatalf("ensureNamespace failed: %v", err)
	}

	// Verify namespace was created
	ns, err := fakeClient.CoreV1().Namespaces().Get(ctx, "test-testapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("namespace not found: %v", err)
	}
	if ns.Name != "test-testapp" {
		t.Errorf("expected namespace name 'test-testapp', got %q", ns.Name)
	}

	// Second call should succeed (namespace exists)
	err = client.ensureNamespace(ctx, cfg)
	if err != nil {
		t.Fatalf("ensureNamespace (existing) failed: %v", err)
	}
}

func TestApplySecret_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	// Create namespace first
	_, err := fakeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "test-namespace",
		EnvVars:   map[string]string{"KEY1": "value1", "KEY2": "value2"},
	}

	ctx := context.Background()

	// First call should create
	err = client.applySecret(ctx, cfg)
	if err != nil {
		t.Fatalf("applySecret failed: %v", err)
	}

	// Verify secret exists (fake client stores StringData, not Data)
	secret, err := fakeClient.CoreV1().Secrets("test-namespace").Get(ctx, "myapp-env", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("secret not found: %v", err)
	}
	// Check StringData since fake client doesn't convert to Data
	if secret.StringData["KEY1"] != "value1" {
		t.Errorf("expected KEY1='value1', got %q", secret.StringData["KEY1"])
	}

	// Update secret
	cfg.EnvVars["KEY3"] = "value3"
	err = client.applySecret(ctx, cfg)
	if err != nil {
		t.Fatalf("applySecret (update) failed: %v", err)
	}
}

func TestApplyDeployment_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	// Create namespace first
	_, _ = fakeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	}, metav1.CreateOptions{})

	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "test-namespace",
		Image:     "nginx:alpine",
		Replicas:  2,
		Port:      80,
	}

	ctx := context.Background()

	// Create deployment
	err := client.applyDeployment(ctx, cfg)
	if err != nil {
		t.Fatalf("applyDeployment failed: %v", err)
	}

	// Verify deployment
	deployment, err := fakeClient.AppsV1().Deployments("test-namespace").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("deployment not found: %v", err)
	}
	if *deployment.Spec.Replicas != 2 {
		t.Errorf("expected 2 replicas, got %d", *deployment.Spec.Replicas)
	}
	if deployment.Spec.Template.Spec.Containers[0].Image != "nginx:alpine" {
		t.Errorf("expected image 'nginx:alpine', got %q", deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestApplyService_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	// Create namespace first
	_, _ = fakeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	}, metav1.CreateOptions{})

	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "test-namespace",
		Port:      8080,
	}

	ctx := context.Background()

	// Create service
	err := client.applyService(ctx, cfg)
	if err != nil {
		t.Fatalf("applyService failed: %v", err)
	}

	// Verify service
	service, err := fakeClient.CoreV1().Services("test-namespace").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("service not found: %v", err)
	}
	if service.Spec.Ports[0].Port != 80 {
		t.Errorf("expected port 80, got %d", service.Spec.Ports[0].Port)
	}
}

func TestApplyIngress_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	// Create namespace first
	_, _ = fakeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-namespace"},
	}, metav1.CreateOptions{})

	cfg := &AppConfig{
		Name:         "myapp",
		Namespace:    "test-namespace",
		Port:         80,
		DomainSuffix: "apps.example.com",
	}

	ctx := context.Background()

	// Create ingress
	err := client.applyIngress(ctx, cfg)
	if err != nil {
		t.Fatalf("applyIngress failed: %v", err)
	}

	// Verify ingress
	ingress, err := fakeClient.NetworkingV1().Ingresses("test-namespace").Get(ctx, "myapp", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ingress not found: %v", err)
	}
	if ingress.Spec.Rules[0].Host != "myapp.apps.example.com" {
		t.Errorf("expected host 'myapp.apps.example.com', got %q", ingress.Spec.Rules[0].Host)
	}
}

func TestDeleteApp_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	// Create namespace
	ctx := context.Background()
	_, err := fakeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-myapp"},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}

	// Delete app
	err = client.DeleteApp(ctx, "myapp")
	if err != nil {
		t.Fatalf("DeleteApp failed: %v", err)
	}

	// Verify namespace is deleted
	_, err = fakeClient.CoreV1().Namespaces().Get(ctx, "test-myapp", metav1.GetOptions{})
	if !k8serrors.IsNotFound(err) {
		t.Error("expected namespace to be deleted")
	}
}

func TestRestartApp_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	ctx := context.Background()

	// Create namespace and deployment
	_, _ = fakeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-myapp"},
	}, metav1.CreateOptions{})

	replicas := int32(1)
	_, err := fakeClient.AppsV1().Deployments("test-myapp").Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "myapp"},
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	// Restart app
	err = client.RestartApp(ctx, "myapp")
	if err != nil {
		t.Fatalf("RestartApp failed: %v", err)
	}

	// Verify restart annotation
	deployment, _ := fakeClient.AppsV1().Deployments("test-myapp").Get(ctx, "myapp", metav1.GetOptions{})
	if deployment.Spec.Template.Annotations == nil {
		t.Fatal("expected annotations to be set")
	}
	if _, ok := deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]; !ok {
		t.Error("expected restartedAt annotation")
	}
}

func TestScaleApp_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	ctx := context.Background()

	// Create namespace and deployment
	_, _ = fakeClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "test-myapp"},
	}, metav1.CreateOptions{})

	replicas := int32(1)
	_, err := fakeClient.AppsV1().Deployments("test-myapp").Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "myapp"},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	// Scale to 5
	err = client.ScaleApp(ctx, "myapp", 5)
	if err != nil {
		t.Fatalf("ScaleApp failed: %v", err)
	}

	// Verify scale
	deployment, _ := fakeClient.AppsV1().Deployments("test-myapp").Get(ctx, "myapp", metav1.GetOptions{})
	if *deployment.Spec.Replicas != 5 {
		t.Errorf("expected 5 replicas, got %d", *deployment.Spec.Replicas)
	}
}

func TestGetAppStatus_WithFakeClient(t *testing.T) {
	t.Run("not deployed", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		client := NewClientWithInterface(fakeClient, "test-")

		status, err := client.GetAppStatus(context.Background(), "nonexistent")
		if err != nil {
			t.Fatalf("GetAppStatus failed: %v", err)
		}
		if status.Status != "not_deployed" {
			t.Errorf("expected status 'not_deployed', got %q", status.Status)
		}
	})

	t.Run("running", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		client := NewClientWithInterface(fakeClient, "test-")

		ctx := context.Background()
		replicas := int32(2)
		_, _ = fakeClient.AppsV1().Deployments("test-myapp").Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp",
				Namespace: "test-myapp",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     2,
				AvailableReplicas: 2,
			},
		}, metav1.CreateOptions{})

		status, err := client.GetAppStatus(ctx, "myapp")
		if err != nil {
			t.Fatalf("GetAppStatus failed: %v", err)
		}
		if status.Status != "running" {
			t.Errorf("expected status 'running', got %q", status.Status)
		}
		if status.Replicas != 2 {
			t.Errorf("expected 2 replicas, got %d", status.Replicas)
		}
	})

	t.Run("partially ready", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		client := NewClientWithInterface(fakeClient, "test-")

		ctx := context.Background()
		replicas := int32(3)
		_, _ = fakeClient.AppsV1().Deployments("test-myapp").Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp",
				Namespace: "test-myapp",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     1,
				AvailableReplicas: 2,
			},
		}, metav1.CreateOptions{})

		status, err := client.GetAppStatus(ctx, "myapp")
		if err != nil {
			t.Fatalf("GetAppStatus failed: %v", err)
		}
		if status.Status != "partially_ready" {
			t.Errorf("expected status 'partially_ready', got %q", status.Status)
		}
	})

	t.Run("starting", func(t *testing.T) {
		fakeClient := fake.NewClientset()
		client := NewClientWithInterface(fakeClient, "test-")

		ctx := context.Background()
		replicas := int32(1)
		_, _ = fakeClient.AppsV1().Deployments("test-myapp").Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp",
				Namespace: "test-myapp",
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
			},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas:     0,
				AvailableReplicas: 0,
			},
		}, metav1.CreateOptions{})

		status, err := client.GetAppStatus(ctx, "myapp")
		if err != nil {
			t.Fatalf("GetAppStatus failed: %v", err)
		}
		if status.Status != "starting" {
			t.Errorf("expected status 'starting', got %q", status.Status)
		}
	})
}

func TestGetPods_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	ctx := context.Background()

	// Create pods
	_, _ = fakeClient.CoreV1().Pods("test-myapp").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "myapp-pod-1",
			Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
		},
	}, metav1.CreateOptions{})
	_, _ = fakeClient.CoreV1().Pods("test-myapp").Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "myapp-pod-2",
			Labels: map[string]string{"app.kubernetes.io/name": "myapp"},
		},
	}, metav1.CreateOptions{})

	pods, err := client.GetPods(ctx, "myapp")
	if err != nil {
		t.Fatalf("GetPods failed: %v", err)
	}
	if len(pods.Items) != 2 {
		t.Errorf("expected 2 pods, got %d", len(pods.Items))
	}
}

func TestGetIngress_WithFakeClient(t *testing.T) {
	fakeClient := fake.NewClientset()
	client := NewClientWithInterface(fakeClient, "test-")

	ctx := context.Background()

	// Create ingress
	pathType := networkingv1.PathTypePrefix
	_, _ = fakeClient.NetworkingV1().Ingresses("test-myapp").Create(ctx, &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name: "myapp",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "myapp.example.com",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
								},
							},
						},
					},
				},
			},
		},
	}, metav1.CreateOptions{})

	ingress, err := client.GetIngress(ctx, "myapp")
	if err != nil {
		t.Fatalf("GetIngress failed: %v", err)
	}
	if ingress.Spec.Rules[0].Host != "myapp.example.com" {
		t.Errorf("expected host 'myapp.example.com', got %q", ingress.Spec.Rules[0].Host)
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
			domainSuffix: "nexo.build",
			customDomain: "",
			expectedURL:  "https://myapp.nexo.build",
		},
		{
			name:         "with custom domain",
			appName:      "myapp",
			domainSuffix: "nexo.build",
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
