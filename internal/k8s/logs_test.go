package k8s

import (
	"context"
	"testing"
	"time"
)

func TestLogLine_Struct(t *testing.T) {
	line := LogLine{
		Pod:       "myapp-abc123",
		Container: "main",
		Message:   "Starting server on port 8080\n",
	}

	if line.Pod != "myapp-abc123" {
		t.Errorf("expected Pod 'myapp-abc123', got %q", line.Pod)
	}
	if line.Container != "main" {
		t.Errorf("expected Container 'main', got %q", line.Container)
	}
	if line.Message != "Starting server on port 8080\n" {
		t.Errorf("expected Message with newline, got %q", line.Message)
	}
}

func TestLogStreamOptions_Struct(t *testing.T) {
	opts := LogStreamOptions{
		Follow:     true,
		TailLines:  100,
		Timestamps: true,
	}

	if !opts.Follow {
		t.Error("expected Follow to be true")
	}
	if opts.TailLines != 100 {
		t.Errorf("expected TailLines 100, got %d", opts.TailLines)
	}
	if !opts.Timestamps {
		t.Error("expected Timestamps to be true")
	}
}

func TestLogStreamOptions_Defaults(t *testing.T) {
	opts := LogStreamOptions{}

	if opts.Follow {
		t.Error("expected default Follow to be false")
	}
	if opts.TailLines != 0 {
		t.Errorf("expected default TailLines 0, got %d", opts.TailLines)
	}
	if opts.Timestamps {
		t.Error("expected default Timestamps to be false")
	}
}

// Integration tests for logs - require a real K8s cluster

func TestStreamLogs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "logs-stream-test-app"
	namespace := client.NamespaceForApp(appName)

	defer cleanupNamespace(t, client, namespace)

	ctx := context.Background()

	// Deploy app first
	cfg := &AppConfig{
		Name:         appName,
		Image:        "nginx:alpine",
		Replicas:     1,
		Port:         80,
		DomainSuffix: "test.local",
	}

	result, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if !result.Success {
		t.Skipf("Deployment didn't succeed: %s", result.Message)
	}

	// Wait a bit for logs to be generated
	time.Sleep(5 * time.Second)

	// Create a channel for log output
	logCh := make(chan LogLine, 100)

	// Use a context with timeout for streaming
	streamCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Start streaming in a goroutine
	errCh := make(chan error, 1)
	go func() {
		err := client.StreamLogs(streamCtx, appName, LogStreamOptions{
			Follow:    false,
			TailLines: 10,
		}, logCh)
		errCh <- err
	}()

	// Collect logs
	var logs []LogLine
	done := false
	for !done {
		select {
		case log := <-logCh:
			logs = append(logs, log)
		case err := <-errCh:
			if err != nil && err != context.DeadlineExceeded && err != context.Canceled {
				t.Logf("StreamLogs ended with error: %v", err)
			}
			done = true
		case <-time.After(15 * time.Second):
			done = true
		}
	}

	t.Logf("Collected %d log lines", len(logs))
}

func TestGetRecentLogs_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "logs-recent-test-app"
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

	result, err := client.Deploy(ctx, cfg)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}
	if !result.Success {
		t.Skipf("Deployment didn't succeed: %s", result.Message)
	}

	// Wait for logs to be generated
	time.Sleep(5 * time.Second)

	// Get recent logs
	logs, err := client.GetRecentLogs(ctx, appName, 50)
	if err != nil {
		t.Fatalf("GetRecentLogs failed: %v", err)
	}

	t.Logf("Got %d recent log lines", len(logs))

	// Verify log structure if we got any
	for _, log := range logs {
		if log.Pod == "" {
			t.Error("expected non-empty Pod name")
		}
		if log.Message == "" {
			t.Error("expected non-empty Message")
		}
	}
}

func TestStreamLogs_NoPods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "nonexistent-logs-app"

	ctx := context.Background()
	logCh := make(chan LogLine, 10)

	err := client.StreamLogs(ctx, appName, LogStreamOptions{TailLines: 10}, logCh)
	if err == nil {
		t.Error("expected error for app with no pods")
	}
}

func TestGetRecentLogs_NoPods(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := skipIfNoCluster(t)
	appName := "nonexistent-logs-app"

	ctx := context.Background()

	logs, err := client.GetRecentLogs(ctx, appName, 50)
	if err != nil {
		// Error is expected since namespace doesn't exist
		t.Logf("Got expected error: %v", err)
		return
	}

	// If no error, should have empty logs
	if len(logs) != 0 {
		t.Errorf("expected 0 logs for nonexistent app, got %d", len(logs))
	}
}
