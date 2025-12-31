package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
)

type LogLine struct {
	Pod       string `json:"pod"`
	Container string `json:"container"`
	Message   string `json:"message"`
}

type LogStreamOptions struct {
	Follow     bool
	TailLines  int64
	Timestamps bool
}

func (c *Client) StreamLogs(ctx context.Context, appName string, opts LogStreamOptions, outputCh chan<- LogLine) error {
	namespace := c.NamespaceForApp(appName)

	pods, err := c.GetPods(ctx, appName)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for app %s", appName)
	}

	errCh := make(chan error, len(pods.Items))

	for _, pod := range pods.Items {
		go func(pod corev1.Pod) {
			err := c.streamPodLogs(ctx, namespace, pod.Name, opts, outputCh)
			if err != nil {
				errCh <- err
			}
		}(pod)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (c *Client) streamPodLogs(ctx context.Context, namespace, podName string, opts LogStreamOptions, outputCh chan<- LogLine) error {
	logOpts := &corev1.PodLogOptions{
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
	}

	if opts.TailLines > 0 {
		logOpts.TailLines = &opts.TailLines
	}

	req := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, logOpts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer func() { _ = stream.Close() }()

	reader := bufio.NewReader(stream)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("error reading log stream: %w", err)
			}

			outputCh <- LogLine{
				Pod:     podName,
				Message: line,
			}
		}
	}
}

func (c *Client) GetRecentLogs(ctx context.Context, appName string, tailLines int64) ([]LogLine, error) {
	namespace := c.NamespaceForApp(appName)

	pods, err := c.GetPods(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	var logs []LogLine

	for _, pod := range pods.Items {
		logOpts := &corev1.PodLogOptions{
			TailLines:  &tailLines,
			Timestamps: true,
		}

		req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, logOpts)
		stream, err := req.Stream(ctx)
		if err != nil {
			continue
		}

		reader := bufio.NewReader(stream)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			logs = append(logs, LogLine{
				Pod:     pod.Name,
				Message: line,
			})
		}
		_ = stream.Close()
	}

	return logs, nil
}
