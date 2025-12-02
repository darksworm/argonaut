// Package portforward provides kubectl port-forward management for ArgoCD access
package portforward

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	cblog "github.com/charmbracelet/log"
)

const (
	// DefaultServerName is the default ArgoCD server app name used in label selector
	DefaultServerName = "argocd-server"

	// DefaultTargetPort is the ArgoCD server port to forward to
	DefaultTargetPort = 8080

	// reconnectDelay is the delay between reconnection attempts
	reconnectDelay = 2 * time.Second

	// maxReconnectAttempts is the maximum number of consecutive reconnection attempts
	maxReconnectAttempts = 5
)

// portRegex matches kubectl port-forward output like "Forwarding from 127.0.0.1:12345 -> 8080"
var portRegex = regexp.MustCompile(`Forwarding from 127\.0\.0\.1:(\d+)`)

// Manager handles kubectl port-forward lifecycle
type Manager struct {
	namespace  string
	serverName string
	targetPort int

	mu              sync.RWMutex
	cmd             *exec.Cmd
	localPort       int
	running         bool
	stopCh          chan struct{}
	reconnectCount  int
	onReconnect     func(port int)
	onDisconnect    func(err error)
}

// Options configures the port-forward manager
type Options struct {
	// Namespace is the Kubernetes namespace where ArgoCD is installed
	Namespace string

	// ServerName is the ArgoCD server app name (default: "argocd-server")
	ServerName string

	// TargetPort is the port to forward to on the ArgoCD server (default: 8080)
	TargetPort int

	// OnReconnect is called when port-forward is re-established with the new port
	OnReconnect func(port int)

	// OnDisconnect is called when port-forward fails permanently
	OnDisconnect func(err error)
}

// NewManager creates a new port-forward manager
func NewManager(opts Options) *Manager {
	if opts.Namespace == "" {
		opts.Namespace = "argocd"
	}
	if opts.ServerName == "" {
		opts.ServerName = DefaultServerName
	}
	if opts.TargetPort == 0 {
		opts.TargetPort = DefaultTargetPort
	}

	return &Manager{
		namespace:    opts.Namespace,
		serverName:   opts.ServerName,
		targetPort:   opts.TargetPort,
		stopCh:       make(chan struct{}),
		onReconnect:  opts.OnReconnect,
		onDisconnect: opts.OnDisconnect,
	}
}

// Start initiates the port-forward connection and returns the local port
func (m *Manager) Start(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return m.localPort, nil
	}

	// Find a ready pod
	podName, err := m.findReadyPod(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find ArgoCD server pod: %w", err)
	}

	cblog.With("component", "portforward").Info("Found ArgoCD server pod", "pod", podName, "namespace", m.namespace)

	// Start port-forward
	port, err := m.startPortForward(ctx, podName)
	if err != nil {
		return 0, err
	}

	m.localPort = port
	m.running = true
	m.reconnectCount = 0

	// Start monitoring goroutine
	go m.monitor(podName)

	return port, nil
}

// Stop terminates the port-forward connection
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false

	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		_ = m.cmd.Wait()
	}

	cblog.With("component", "portforward").Info("Port-forward stopped")
}

// LocalPort returns the current local port
func (m *Manager) LocalPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.localPort
}

// IsRunning returns true if port-forward is active
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// ServerAddress returns the local server address (e.g., "127.0.0.1:12345")
func (m *Manager) ServerAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return fmt.Sprintf("127.0.0.1:%d", m.localPort)
}

// findReadyPod finds a ready ArgoCD server pod using kubectl
func (m *Manager) findReadyPod(ctx context.Context) (string, error) {
	// Use label selector like ArgoCD CLI: app.kubernetes.io/name=argocd-server
	labelSelector := fmt.Sprintf("app.kubernetes.io/name=%s", m.serverName)

	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods",
		"-n", m.namespace,
		"-l", labelSelector,
		"--field-selector=status.phase=Running",
		"-o", "jsonpath={.items[?(@.status.containerStatuses[0].ready==true)].metadata.name}",
	)

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("kubectl error: %s", string(exitErr.Stderr))
		}
		return "", err
	}

	pods := strings.Fields(string(output))
	if len(pods) == 0 {
		return "", fmt.Errorf("no ready pods found with selector %s in namespace %s", labelSelector, m.namespace)
	}

	// Return first ready pod
	return pods[0], nil
}

// startPortForward starts kubectl port-forward and returns the allocated local port
func (m *Manager) startPortForward(ctx context.Context, podName string) (int, error) {
	// Use :0 to let kubectl pick an available port
	portSpec := fmt.Sprintf(":%d", m.targetPort)

	m.cmd = exec.CommandContext(ctx, "kubectl", "port-forward",
		"-n", m.namespace,
		podName,
		portSpec,
	)

	// Capture stdout to parse the port
	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := m.cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start kubectl port-forward: %w", err)
	}

	// Read stderr in background for logging
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			cblog.With("component", "portforward").Debug("kubectl stderr", "line", scanner.Text())
		}
	}()

	// Parse stdout for port assignment
	portCh := make(chan int, 1)
	errCh := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			cblog.With("component", "portforward").Debug("kubectl stdout", "line", line)

			matches := portRegex.FindStringSubmatch(line)
			if len(matches) >= 2 {
				port, err := strconv.Atoi(matches[1])
				if err == nil {
					portCh <- port
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		} else {
			errCh <- fmt.Errorf("kubectl port-forward exited without providing port")
		}
	}()

	// Wait for port or error with timeout
	select {
	case port := <-portCh:
		cblog.With("component", "portforward").Info("Port-forward established", "localPort", port, "targetPort", m.targetPort)
		return port, nil
	case err := <-errCh:
		_ = m.cmd.Process.Kill()
		return 0, err
	case <-time.After(10 * time.Second):
		_ = m.cmd.Process.Kill()
		return 0, fmt.Errorf("timeout waiting for port-forward to establish")
	case <-ctx.Done():
		_ = m.cmd.Process.Kill()
		return 0, ctx.Err()
	}
}

// monitor watches the port-forward process and handles reconnection
func (m *Manager) monitor(lastPodName string) {
	for {
		select {
		case <-m.stopCh:
			return
		default:
		}

		// Wait for process to exit
		if m.cmd != nil {
			err := m.cmd.Wait()
			cblog.With("component", "portforward").Warn("Port-forward disconnected", "err", err)
		}

		m.mu.Lock()
		if !m.running {
			m.mu.Unlock()
			return
		}

		m.reconnectCount++
		if m.reconnectCount > maxReconnectAttempts {
			m.running = false
			m.mu.Unlock()

			cblog.With("component", "portforward").Error("Max reconnection attempts reached")
			if m.onDisconnect != nil {
				m.onDisconnect(fmt.Errorf("port-forward failed after %d reconnection attempts", maxReconnectAttempts))
			}
			return
		}
		m.mu.Unlock()

		cblog.With("component", "portforward").Info("Attempting to reconnect", "attempt", m.reconnectCount)

		// Wait before reconnecting
		select {
		case <-m.stopCh:
			return
		case <-time.After(reconnectDelay):
		}

		// Try to reconnect
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Find a ready pod (might be different after restart)
		podName, err := m.findReadyPod(ctx)
		if err != nil {
			cblog.With("component", "portforward").Warn("Failed to find pod for reconnection", "err", err)
			cancel()
			continue
		}

		port, err := m.startPortForward(ctx, podName)
		cancel()

		if err != nil {
			cblog.With("component", "portforward").Warn("Failed to reconnect", "err", err)
			continue
		}

		m.mu.Lock()
		m.localPort = port
		m.reconnectCount = 0 // Reset on successful reconnection
		m.mu.Unlock()

		cblog.With("component", "portforward").Info("Reconnected successfully", "port", port)

		if m.onReconnect != nil {
			m.onReconnect(port)
		}
	}
}

// CheckKubectl verifies that kubectl is available and configured
func CheckKubectl(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "kubectl", "version", "--client", "--output=json")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("kubectl not found or not configured: %w", err)
	}
	return nil
}
