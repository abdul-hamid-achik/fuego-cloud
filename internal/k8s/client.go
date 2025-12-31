package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset       *kubernetes.Clientset
	config          *rest.Config
	namespacePrefix string
}

func NewClient(kubeconfig, namespacePrefix string) (*Client, error) {
	config, err := getConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &Client{
		clientset:       clientset,
		config:          config,
		namespacePrefix: namespacePrefix,
	}, nil
}

func getConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}

	if envConfig := os.Getenv("KUBECONFIG"); envConfig != "" {
		return clientcmd.BuildConfigFromFlags("", envConfig)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(home, ".kube", "config")
		if _, err := os.Stat(defaultPath); err == nil {
			return clientcmd.BuildConfigFromFlags("", defaultPath)
		}
	}

	return rest.InClusterConfig()
}

func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

func (c *Client) Config() *rest.Config {
	return c.config
}

func (c *Client) NamespaceForApp(appName string) string {
	return c.namespacePrefix + appName
}
