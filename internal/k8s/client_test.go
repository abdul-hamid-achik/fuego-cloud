package k8s

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNamespaceForApp(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		appName  string
		expected string
	}{
		{
			name:     "standard prefix",
			prefix:   "tenant-",
			appName:  "myapp",
			expected: "tenant-myapp",
		},
		{
			name:     "custom prefix",
			prefix:   "fuego-",
			appName:  "myapp",
			expected: "fuego-myapp",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			appName:  "myapp",
			expected: "myapp",
		},
		{
			name:     "complex app name",
			prefix:   "tenant-",
			appName:  "my-app-123",
			expected: "tenant-my-app-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{namespacePrefix: tt.prefix}
			got := client.NamespaceForApp(tt.appName)
			if got != tt.expected {
				t.Errorf("NamespaceForApp(%q) = %q, want %q", tt.appName, got, tt.expected)
			}
		})
	}
}

func TestGetConfig_ExplicitPath(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Write minimal kubeconfig
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	config, err := getConfig(kubeconfigPath)
	if err != nil {
		t.Fatalf("getConfig failed: %v", err)
	}

	if config.Host != "https://localhost:6443" {
		t.Errorf("expected host https://localhost:6443, got %s", config.Host)
	}
}

func TestGetConfig_FromEnvVar(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://env-cluster:6443
  name: env-cluster
contexts:
- context:
    cluster: env-cluster
    user: env-user
  name: env-context
current-context: env-context
users:
- name: env-user
  user:
    token: env-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	// Set KUBECONFIG env var
	oldKubeconfig := os.Getenv("KUBECONFIG")
	t.Setenv("KUBECONFIG", kubeconfigPath)
	defer func() {
		if oldKubeconfig != "" {
			_ = os.Setenv("KUBECONFIG", oldKubeconfig)
		} else {
			_ = os.Unsetenv("KUBECONFIG")
		}
	}()

	config, err := getConfig("") // Empty path should use env var
	if err != nil {
		t.Fatalf("getConfig failed: %v", err)
	}

	if config.Host != "https://env-cluster:6443" {
		t.Errorf("expected host https://env-cluster:6443, got %s", config.Host)
	}
}

func TestGetConfig_InvalidPath(t *testing.T) {
	_, err := getConfig("/nonexistent/path/to/kubeconfig")
	if err == nil {
		t.Error("expected error for invalid kubeconfig path")
	}
}

func TestGetConfig_MalformedKubeconfig(t *testing.T) {
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	// Write invalid kubeconfig
	if err := os.WriteFile(kubeconfigPath, []byte("not valid yaml: {{{}}}"), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	_, err := getConfig(kubeconfigPath)
	if err == nil {
		t.Error("expected error for malformed kubeconfig")
	}
}

func TestNewClient_InvalidKubeconfig(t *testing.T) {
	_, err := NewClient("/nonexistent/kubeconfig", "tenant-")
	if err == nil {
		t.Error("expected error for invalid kubeconfig")
	}
}

func TestNewClient_ValidKubeconfig(t *testing.T) {
	// Create a temporary kubeconfig file
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	client, err := NewClient(kubeconfigPath, "test-")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Clientset() == nil {
		t.Error("expected non-nil clientset")
	}
	if client.Config() == nil {
		t.Error("expected non-nil config")
	}
	if client.namespacePrefix != "test-" {
		t.Errorf("expected namespace prefix 'test-', got %q", client.namespacePrefix)
	}
}

func TestClient_Getters(t *testing.T) {
	// Create a minimal client for testing getters
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://localhost:6443
    insecure-skip-tls-verify: true
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	client, err := NewClient(kubeconfigPath, "prefix-")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Test Clientset getter
	clientset := client.Clientset()
	if clientset == nil {
		t.Error("Clientset() returned nil")
	}

	// Test Config getter
	config := client.Config()
	if config == nil {
		t.Fatal("Config() returned nil")
	}
	if config.Host != "https://localhost:6443" {
		t.Errorf("expected host https://localhost:6443, got %s", config.Host)
	}
}

func TestGetConfig_Priority(t *testing.T) {
	// Test that explicit path takes priority over env var
	tmpDir := t.TempDir()

	// Create two different kubeconfig files
	explicitPath := filepath.Join(tmpDir, "explicit-config")
	envPath := filepath.Join(tmpDir, "env-config")

	explicitContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://explicit:6443
  name: explicit
contexts:
- context:
    cluster: explicit
    user: explicit
  name: explicit
current-context: explicit
users:
- name: explicit
  user:
    token: token
`
	envContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://from-env:6443
  name: from-env
contexts:
- context:
    cluster: from-env
    user: from-env
  name: from-env
current-context: from-env
users:
- name: from-env
  user:
    token: token
`
	if err := os.WriteFile(explicitPath, []byte(explicitContent), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envPath, []byte(envContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Set env var
	t.Setenv("KUBECONFIG", envPath)

	// Explicit path should take priority
	config, err := getConfig(explicitPath)
	if err != nil {
		t.Fatal(err)
	}
	if config == nil {
		t.Fatal("expected non-nil config")
	}
	if config.Host != "https://explicit:6443" {
		t.Errorf("expected explicit config host, got %s", config.Host)
	}
}
