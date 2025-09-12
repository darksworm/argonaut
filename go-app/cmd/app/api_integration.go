package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/a9s/go-app/pkg/model"
	"github.com/a9s/go-app/pkg/services"
	tea "github.com/charmbracelet/bubbletea/v2"
	yaml "gopkg.in/yaml.v3"
)

// startLoadingApplications initiates loading applications from ArgoCD API
func (m Model) startLoadingApplications() tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AuthErrorMsg{Error: fmt.Errorf("no server configured")}
		}
	}

	return tea.Cmd(func() tea.Msg {
		// Log the API call attempt
		// [API] Starting to load applications - removed printf to avoid TUI interference

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Load applications
		// [API] Calling ListApplications - removed printf to avoid TUI interference
		apps, err := apiService.ListApplications(ctx, m.state.Server)
		if err != nil {
			// [API] Error loading applications - removed printf to avoid TUI interference
			// Check if it's an auth error
			errMsg := err.Error()
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: err.Error()}
		}

		// Successfully loaded applications
		// [API] Successfully loaded applications - removed printf to avoid TUI interference
		return model.AppsLoadedMsg{Apps: apps}
	})
}

// startWatchingApplications starts the real-time watch stream
func (m Model) startWatchingApplications() tea.Cmd {
	if m.state.Server == nil {
		return nil
	}

	return tea.Cmd(func() tea.Msg {
		// Create context for the watch stream
		ctx := context.Background()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Start watching applications
		eventChan, _, err := apiService.WatchApplications(ctx, m.state.Server)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to start watch: " + err.Error()}
		}

		// Store channel and start first consume
		m.watchChan = make(chan services.ArgoApiEvent, 100)
		go func() {
			for ev := range eventChan {
				m.watchChan <- ev
			}
			close(m.watchChan)
		}()
		return model.StatusChangeMsg{Status: "Watching for changes..."}
	})
}

// fetchAPIVersion fetches the ArgoCD API version and updates state
func (m Model) fetchAPIVersion() tea.Cmd {
	if m.state.Server == nil {
		return nil
	}
	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiService := services.NewArgoApiService(m.state.Server)
		v, err := apiService.GetAPIVersion(ctx, m.state.Server)
		if err != nil {
			return model.StatusChangeMsg{Status: "Version: unknown"}
		}
		return model.SetAPIVersionMsg{Version: v}
	})
}

// consumeWatchEvent reads a single service event and converts it to a tea message
func (m Model) consumeWatchEvent() tea.Cmd {
	return func() tea.Msg {
		if m.watchChan == nil {
			return nil
		}
		ev, ok := <-m.watchChan
		if !ok {
			return nil
		}
		switch ev.Type {
		case "apps-loaded":
			if ev.Apps != nil {
				return model.AppsLoadedMsg{Apps: ev.Apps}
			}
		case "app-updated":
			if ev.App != nil {
				return model.AppUpdatedMsg{App: *ev.App}
			}
		case "app-deleted":
			if ev.AppName != "" {
				return model.AppDeletedMsg{AppName: ev.AppName}
			}
		case "status-change":
			if ev.Status != "" {
				return model.StatusChangeMsg{Status: ev.Status}
			}
		case "auth-error":
			if ev.Error != nil {
				return model.AuthErrorMsg{Error: ev.Error}
			}
		case "api-error":
			if ev.Error != nil {
				return model.ApiErrorMsg{Message: ev.Error.Error()}
			}
		}
		return nil
	}
}

// startDiffSession loads diffs and opens the diff pager
func (m Model) startDiffSession(appName string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
		
		// Add artificial delay to demonstrate spinner overlay
		time.Sleep(2 * time.Second)
		
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error()}
		}

		desiredDocs := make([]string, 0)
		liveDocs := make([]string, 0)
		for _, d := range diffs {
			if d.TargetState != "" {
				s := cleanManifestToYAML(d.TargetState)
				if s != "" {
					desiredDocs = append(desiredDocs, s)
				}
			}
			if d.LiveState != "" {
				s := cleanManifestToYAML(d.LiveState)
				if s != "" {
					liveDocs = append(liveDocs, s)
				}
			}
		}

		if len(desiredDocs) == 0 && len(liveDocs) == 0 {
			// Add delay even for "No diffs" case to demonstrate spinner
			time.Sleep(1 * time.Second)
			return model.StatusChangeMsg{Status: "No diffs"}
		}

		leftFile, _ := writeTempYAML("live-", liveDocs)
		rightFile, _ := writeTempYAML("desired-", desiredDocs)

		cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--color=always", "--", leftFile, rightFile)
		out, err := cmd.CombinedOutput()
		if err != nil && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 1 {
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error()}
		}
		cleaned := stripDiffHeader(string(out))
		if strings.TrimSpace(cleaned) == "" {
			return model.StatusChangeMsg{Status: "No differences"}
		}
		lines := strings.Split(cleaned, "\n")
		m.state.Diff = &model.DiffState{Title: fmt.Sprintf("%s - Live vs Desired (Cleaned)", appName), Content: lines, Offset: 0, Loading: false}
		return model.SetModeMsg{Mode: model.ModeDiff}
	})
}

func writeTempYAML(prefix string, docs []string) (string, error) {
	f, err := os.CreateTemp("", prefix+"*.yaml")
	if err != nil {
		return "", err
	}
	defer f.Close()
	content := strings.Join(docs, "\n---\n")
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func cleanManifestToYAML(jsonOrYaml string) string {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOrYaml), &obj); err == nil {
		if m, ok := obj["metadata"].(map[string]interface{}); ok {
			delete(m, "creationTimestamp")
			delete(m, "resourceVersion")
			delete(m, "uid")
			delete(m, "managedFields")
			if ann, ok := m["annotations"].(map[string]interface{}); ok {
				delete(ann, "kubectl.kubernetes.io/last-applied-configuration")
				delete(ann, "deployment.kubernetes.io/revision")
				if len(ann) == 0 {
					delete(m, "annotations")
				}
			}
			if len(m) == 0 {
				delete(obj, "metadata")
			}
		}
		delete(obj, "status")
		if spec, ok := obj["spec"].(map[string]interface{}); ok {
			delete(spec, "serviceAccount")
			if tpl, ok := spec["template"].(map[string]interface{}); ok {
				if ps, ok := tpl["spec"].(map[string]interface{}); ok {
					if cs, ok := ps["containers"].([]interface{}); ok {
						for _, c := range cs {
							if cm, ok := c.(map[string]interface{}); ok {
								if cm["imagePullPolicy"] == "IfNotPresent" {
									delete(cm, "imagePullPolicy")
								}
								delete(cm, "terminationMessagePath")
								delete(cm, "terminationMessagePolicy")
							}
						}
					}
				}
			}
		}
		by, err := yaml.Marshal(obj)
		if err == nil {
			return string(by)
		}
	}
	return jsonOrYaml
}

func stripDiffHeader(out string) string {
	lines := strings.Split(out, "\n")
	start := 0
	for i, ln := range lines {
		s := strings.TrimSpace(ln)
		if s == "" {
			continue
		}
		if strings.HasPrefix(s, "@@") || strings.HasPrefix(s, "+") || strings.HasPrefix(s, "-") || strings.Contains(s, "│") {
			start = i
			break
		}
	}
	return strings.Join(lines[start:], "\n")
}

// syncSelectedApplications syncs the currently selected applications
func (m Model) syncSelectedApplications(prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	selectedApps := make([]string, 0, len(m.state.Selections.SelectedApps))
	for appName := range m.state.Selections.SelectedApps {
		selectedApps = append(selectedApps, appName)
	}

	if len(selectedApps) == 0 {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No applications selected"}
		}
	}

	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		for _, appName := range selectedApps {
			err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
			if err != nil {
				return model.ApiErrorMsg{Message: fmt.Sprintf("Failed to sync %s: %v", appName, err)}
			}
		}

		return model.MultiSyncCompletedMsg{AppCount: len(selectedApps), Success: true}
	})
}

// syncSingleApplication syncs a specific application
func (m Model) syncSingleApplication(appName string, prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
		if err != nil {
			return model.ApiErrorMsg{Message: fmt.Sprintf("Failed to sync %s: %v", appName, err)}
		}

		return model.SyncCompletedMsg{AppName: appName, Success: true}
	})
}

// isAuthenticationError checks if an error is related to authentication
func isAuthenticationError(errMsg string) bool {
	authIndicators := []string{
		"401", "403", "unauthorized", "forbidden", "authentication", "auth",
		"login", "token", "invalid credentials", "access denied",
	}

	for _, indicator := range authIndicators {
		if strings.Contains(strings.ToLower(errMsg), indicator) {
			return true
		}
	}
	return false
}

// startLogsSession opens application logs in pager
func (m Model) startLogsSession() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		data, err := os.ReadFile("logs/a9s.log")
		if err != nil {
			return model.ApiErrorMsg{Message: "No logs available"}
		}
		lines := strings.Split(string(data), "\n")
		offset := len(lines) - (m.state.Terminal.Rows - 4)
		if offset < 0 {
			offset = 0
		}
		m.state.Diff = &model.DiffState{Title: "Logs", Content: lines, Offset: offset}
		return model.SetModeMsg{Mode: model.ModeDiff}
	})
}
