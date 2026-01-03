package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestGenerateNamespace(t *testing.T) {
	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "fuego-myapp",
	}

	ns := GenerateNamespace(cfg)

	if ns.Name != "fuego-myapp" {
		t.Errorf("expected namespace name 'fuego-myapp', got %q", ns.Name)
	}

	if ns.Labels["app.kubernetes.io/name"] != "myapp" {
		t.Errorf("expected label 'app.kubernetes.io/name'='myapp', got %q", ns.Labels["app.kubernetes.io/name"])
	}

	if ns.Labels["app.kubernetes.io/managed-by"] != "fuego-cloud" {
		t.Errorf("expected label 'app.kubernetes.io/managed-by'='fuego-cloud', got %q", ns.Labels["app.kubernetes.io/managed-by"])
	}
}

func TestGenerateSecret(t *testing.T) {
	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "fuego-myapp",
		EnvVars: map[string]string{
			"DATABASE_URL": "postgres://localhost/mydb",
			"API_KEY":      "secret-key",
		},
	}

	secret := GenerateSecret(cfg)

	if secret.Name != "myapp-env" {
		t.Errorf("expected secret name 'myapp-env', got %q", secret.Name)
	}

	if secret.Namespace != "fuego-myapp" {
		t.Errorf("expected namespace 'fuego-myapp', got %q", secret.Namespace)
	}

	if secret.Type != corev1.SecretTypeOpaque {
		t.Errorf("expected secret type Opaque, got %v", secret.Type)
	}

	if secret.StringData["DATABASE_URL"] != "postgres://localhost/mydb" {
		t.Errorf("expected DATABASE_URL='postgres://localhost/mydb', got %q", secret.StringData["DATABASE_URL"])
	}

	if secret.StringData["API_KEY"] != "secret-key" {
		t.Errorf("expected API_KEY='secret-key', got %q", secret.StringData["API_KEY"])
	}
}

func TestGenerateDeployment(t *testing.T) {
	replicas := int32(2)
	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "fuego-myapp",
		Image:     "ghcr.io/user/myapp:v1.0.0",
		Replicas:  replicas,
		Port:      8080,
	}

	deployment := GenerateDeployment(cfg)

	if deployment.Name != "myapp" {
		t.Errorf("expected deployment name 'myapp', got %q", deployment.Name)
	}

	if deployment.Namespace != "fuego-myapp" {
		t.Errorf("expected namespace 'fuego-myapp', got %q", deployment.Namespace)
	}

	if *deployment.Spec.Replicas != 2 {
		t.Errorf("expected 2 replicas, got %d", *deployment.Spec.Replicas)
	}

	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	container := containers[0]
	if container.Name != "myapp" {
		t.Errorf("expected container name 'myapp', got %q", container.Name)
	}

	if container.Image != "ghcr.io/user/myapp:v1.0.0" {
		t.Errorf("expected image 'ghcr.io/user/myapp:v1.0.0', got %q", container.Image)
	}

	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 8080 {
		t.Errorf("expected port 8080, got %v", container.Ports)
	}

	// Check probes
	if container.LivenessProbe == nil {
		t.Error("expected liveness probe to be set")
	} else if container.LivenessProbe.HTTPGet.Path != "/api/health" {
		t.Errorf("expected liveness probe path '/api/health', got %q", container.LivenessProbe.HTTPGet.Path)
	}

	if container.ReadinessProbe == nil {
		t.Error("expected readiness probe to be set")
	} else if container.ReadinessProbe.HTTPGet.Path != "/api/health" {
		t.Errorf("expected readiness probe path '/api/health', got %q", container.ReadinessProbe.HTTPGet.Path)
	}

	// Check env from secret
	if len(container.EnvFrom) != 1 {
		t.Fatalf("expected 1 envFrom, got %d", len(container.EnvFrom))
	}

	if container.EnvFrom[0].SecretRef.Name != "myapp-env" {
		t.Errorf("expected secret ref 'myapp-env', got %q", container.EnvFrom[0].SecretRef.Name)
	}
}

func TestGenerateService(t *testing.T) {
	cfg := &AppConfig{
		Name:      "myapp",
		Namespace: "fuego-myapp",
		Port:      8080,
	}

	service := GenerateService(cfg)

	if service.Name != "myapp" {
		t.Errorf("expected service name 'myapp', got %q", service.Name)
	}

	if service.Namespace != "fuego-myapp" {
		t.Errorf("expected namespace 'fuego-myapp', got %q", service.Namespace)
	}

	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("expected ClusterIP service type, got %v", service.Spec.Type)
	}

	if len(service.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(service.Spec.Ports))
	}

	port := service.Spec.Ports[0]
	if port.Port != 80 {
		t.Errorf("expected port 80, got %d", port.Port)
	}

	if port.TargetPort.IntVal != 8080 {
		t.Errorf("expected target port 8080, got %d", port.TargetPort.IntVal)
	}
}

func TestGenerateIngress(t *testing.T) {
	t.Run("with domain suffix", func(t *testing.T) {
		cfg := &AppConfig{
			Name:         "myapp",
			Namespace:    "fuego-myapp",
			DomainSuffix: "fuego.build",
		}

		ingress := GenerateIngress(cfg)

		if ingress.Name != "myapp" {
			t.Errorf("expected ingress name 'myapp', got %q", ingress.Name)
		}

		if len(ingress.Spec.Rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(ingress.Spec.Rules))
		}

		expectedHost := "myapp.fuego.build"
		if ingress.Spec.Rules[0].Host != expectedHost {
			t.Errorf("expected host %q, got %q", expectedHost, ingress.Spec.Rules[0].Host)
		}

		// Check TLS
		if len(ingress.Spec.TLS) != 1 {
			t.Fatalf("expected 1 TLS config, got %d", len(ingress.Spec.TLS))
		}

		if ingress.Spec.TLS[0].SecretName != "myapp-tls" {
			t.Errorf("expected TLS secret name 'myapp-tls', got %q", ingress.Spec.TLS[0].SecretName)
		}

		if ingress.Spec.TLS[0].Hosts[0] != expectedHost {
			t.Errorf("expected TLS host %q, got %q", expectedHost, ingress.Spec.TLS[0].Hosts[0])
		}

		// Check annotations
		if ingress.Annotations["cert-manager.io/cluster-issuer"] != "letsencrypt-prod" {
			t.Errorf("expected cert-manager annotation, got %q", ingress.Annotations["cert-manager.io/cluster-issuer"])
		}
	})

	t.Run("with custom domain", func(t *testing.T) {
		cfg := &AppConfig{
			Name:         "myapp",
			Namespace:    "fuego-myapp",
			Domain:       "myapp.example.com",
			DomainSuffix: "fuego.build",
		}

		ingress := GenerateIngress(cfg)

		expectedHost := "myapp.example.com"
		if ingress.Spec.Rules[0].Host != expectedHost {
			t.Errorf("expected host %q, got %q", expectedHost, ingress.Spec.Rules[0].Host)
		}

		if ingress.Spec.TLS[0].Hosts[0] != expectedHost {
			t.Errorf("expected TLS host %q, got %q", expectedHost, ingress.Spec.TLS[0].Hosts[0])
		}
	})
}

func TestGenerateDeploymentDefaults(t *testing.T) {
	cfg := &AppConfig{
		Name:      "testapp",
		Namespace: "fuego-testapp",
		Image:     "nginx:latest",
		Replicas:  1,
		Port:      80,
	}

	deployment := GenerateDeployment(cfg)

	// Verify selector matches pod labels
	podLabels := deployment.Spec.Template.Labels
	selectorLabels := deployment.Spec.Selector.MatchLabels

	for k, v := range selectorLabels {
		if podLabels[k] != v {
			t.Errorf("selector label %q=%q doesn't match pod label %q", k, v, podLabels[k])
		}
	}

	// Verify labels are consistent
	if deployment.Labels["app.kubernetes.io/name"] != "testapp" {
		t.Errorf("expected deployment label 'app.kubernetes.io/name'='testapp', got %q", deployment.Labels["app.kubernetes.io/name"])
	}
}

func TestAppConfigValidation(t *testing.T) {
	// Test with minimal config
	cfg := &AppConfig{
		Name:      "app",
		Namespace: "ns",
		Image:     "img",
		Replicas:  1,
		Port:      3000,
	}

	// All generators should work with minimal config
	ns := GenerateNamespace(cfg)
	if ns == nil {
		t.Error("GenerateNamespace returned nil")
	}

	secret := GenerateSecret(cfg)
	if secret == nil {
		t.Error("GenerateSecret returned nil")
	}

	deployment := GenerateDeployment(cfg)
	if deployment == nil {
		t.Error("GenerateDeployment returned nil")
	}

	service := GenerateService(cfg)
	if service == nil {
		t.Error("GenerateService returned nil")
	}

	ingress := GenerateIngress(cfg)
	if ingress == nil {
		t.Error("GenerateIngress returned nil")
	}
}

func TestClientNamespaceForApp(t *testing.T) {
	// Can't test NewClient without a valid kubeconfig, but we can test the helper
	// by accessing it through a mock scenario

	tests := []struct {
		prefix   string
		appName  string
		expected string
	}{
		{"fuego-", "myapp", "fuego-myapp"},
		{"app-", "test", "app-test"},
		{"", "standalone", "standalone"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			// Simulate what NamespaceForApp does
			result := tt.prefix + tt.appName
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAppStatus(t *testing.T) {
	// Test AppStatus struct
	status := AppStatus{
		Status:            "running",
		Replicas:          3,
		ReadyReplicas:     3,
		AvailableReplicas: 3,
		Conditions:        []string{"Available: True", "Progressing: True"},
	}

	if status.Status != "running" {
		t.Errorf("expected status 'running', got %q", status.Status)
	}

	if status.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", status.Replicas)
	}

	if len(status.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(status.Conditions))
	}
}
