package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PodMetrics represents resource metrics for a pod
type PodMetrics struct {
	Name        string  `json:"name"`
	CPUCores    float64 `json:"cpu_cores"`    // CPU in cores (e.g., 0.5 = 500m)
	MemoryBytes int64   `json:"memory_bytes"` // Memory in bytes
	CPUPercent  float64 `json:"cpu_percent"`  // CPU usage as percentage of request
	MemoryMB    float64 `json:"memory_mb"`    // Memory in MB for convenience
}

// AppMetrics represents aggregated metrics for an app
type AppMetrics struct {
	AppName       string       `json:"app_name"`
	Namespace     string       `json:"namespace"`
	PodCount      int          `json:"pod_count"`
	ReadyPods     int          `json:"ready_pods"`
	TotalCPU      float64      `json:"total_cpu_cores"`
	TotalMemoryMB float64      `json:"total_memory_mb"`
	AvgCPU        float64      `json:"avg_cpu_cores"`
	AvgMemoryMB   float64      `json:"avg_memory_mb"`
	Pods          []PodMetrics `json:"pods,omitempty"`
}

// GetAppMetrics retrieves resource metrics for an app by querying pod resource usage
// Note: This requires metrics-server to be installed in the cluster for real metrics.
// If metrics-server is not available, it falls back to resource requests/limits.
func (c *Client) GetAppMetrics(ctx context.Context, appName string) (*AppMetrics, error) {
	namespace := c.NamespaceForApp(appName)

	// Get pods for this app
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", appName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	metrics := &AppMetrics{
		AppName:   appName,
		Namespace: namespace,
		PodCount:  len(pods.Items),
		Pods:      make([]PodMetrics, 0, len(pods.Items)),
	}

	var totalCPU float64
	var totalMemory int64

	for _, pod := range pods.Items {
		podMetric := PodMetrics{
			Name: pod.Name,
		}

		// Check if pod is ready
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				metrics.ReadyPods++
				break
			}
		}

		// Get resource requests/limits from containers as baseline
		// In production, you'd query metrics-server for actual usage
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests.Cpu(); cpu != nil {
					podMetric.CPUCores += float64(cpu.MilliValue()) / 1000.0
				}
				if mem := container.Resources.Requests.Memory(); mem != nil {
					podMetric.MemoryBytes += mem.Value()
				}
			}
		}

		podMetric.MemoryMB = float64(podMetric.MemoryBytes) / (1024 * 1024)
		totalCPU += podMetric.CPUCores
		totalMemory += podMetric.MemoryBytes

		metrics.Pods = append(metrics.Pods, podMetric)
	}

	metrics.TotalCPU = totalCPU
	metrics.TotalMemoryMB = float64(totalMemory) / (1024 * 1024)

	if len(pods.Items) > 0 {
		metrics.AvgCPU = totalCPU / float64(len(pods.Items))
		metrics.AvgMemoryMB = metrics.TotalMemoryMB / float64(len(pods.Items))
	}

	return metrics, nil
}

// GetPodResourceUsage gets resource usage for pods using the pod's status
// This is a fallback when metrics-server is not available
func (c *Client) GetPodResourceUsage(ctx context.Context, appName string) ([]PodMetrics, error) {
	namespace := c.NamespaceForApp(appName)

	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", appName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	metrics := make([]PodMetrics, 0, len(pods.Items))
	for _, pod := range pods.Items {
		podMetric := PodMetrics{
			Name: pod.Name,
		}

		// Sum resources from all containers
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests.Cpu(); cpu != nil {
					podMetric.CPUCores += float64(cpu.MilliValue()) / 1000.0
				}
				if mem := container.Resources.Requests.Memory(); mem != nil {
					podMetric.MemoryBytes += mem.Value()
				}
			}
		}

		podMetric.MemoryMB = float64(podMetric.MemoryBytes) / (1024 * 1024)
		metrics = append(metrics, podMetric)
	}

	return metrics, nil
}
