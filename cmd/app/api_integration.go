package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	stdErrors "errors"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/neat"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/services/appdelete"
	yaml "gopkg.in/yaml.v3"
)

// startLoadingApplications initiates loading applications from ArgoCD API
func (m *Model) startLoadingApplications() tea.Cmd {
	cblog.With("component", "api_integration").Info("startLoadingApplications called")
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AuthErrorMsg{Error: fmt.Errorf("no server configured")}
		}
	}

	return func() tea.Msg {
		cblog.With("component", "api_integration").Info("startLoadingApplications: executing load")

		// Create context with timeout (shorter timeout for initial loading)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Load applications
		// [API] Calling ListApplications - removed printf to avoid TUI interference
		apps, err := apiService.ListApplications(ctx, m.state.Server)
		if err != nil {
			// Unwrap structured errors if wrapped
			var argErr *apperrors.ArgonautError
			if stdErrors.As(err, &argErr) {
				if argErr.IsCategory(apperrors.ErrorAuth) || argErr.Code == "UNAUTHORIZED" || argErr.Code == "AUTHENTICATION_FAILED" || hasHTTPStatusCtx(argErr, 401, 403) {
					return model.AuthErrorMsg{Error: argErr}
				}
				// Surface structured errors so error view can show details/context
				return model.StructuredErrorMsg{Error: argErr}
			}
			// Fallback string matching
			if isAuthenticationError(err.Error()) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: err.Error()}
		}

		// Successfully loaded applications
		// [API] Successfully loaded applications - removed printf to avoid TUI interference
		return model.AppsLoadedMsg{Apps: apps}
	}
}

// WatchStartedMsg indicates the watch stream has started
type watchStartedMsg struct {
	eventChan <-chan services.ArgoApiEvent
}

// startWatchingApplications starts the real-time watch stream
func (m *Model) startWatchingApplications() tea.Cmd {
	cblog.With("component", "api_integration").Info("startWatchingApplications called", "watchChan_nil", m.watchChan == nil)
	if m.state.Server == nil {
		return nil
	}

	return func() tea.Msg {
		cblog.With("component", "api_integration").Info("startWatchingApplications: executing watch setup")
		// Create context for the watch stream
		ctx := context.Background()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Start watching applications
		eventChan, _, err := apiService.WatchApplications(ctx, m.state.Server)
		if err != nil {
			// Promote auth-related errors to AuthErrorMsg
			var argErr *apperrors.ArgonautError
			if stdErrors.As(err, &argErr) {
				if hasHTTPStatusCtx(argErr, 401, 403) || argErr.IsCategory(apperrors.ErrorAuth) || argErr.IsCode("UNAUTHORIZED") || argErr.IsCode("AUTHENTICATION_FAILED") {
					return model.AuthErrorMsg{Error: err}
				}
				return model.StructuredErrorMsg{Error: argErr}
			}
			if isAuthenticationError(err.Error()) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: "Failed to start watch: " + err.Error()}
		}

		// Return message with the event channel so Update can set it properly
		cblog.With("component", "watch").Info("Watch started successfully, returning watchStartedMsg")
		return watchStartedMsg{eventChan: eventChan}
	}
}

// fetchAPIVersion fetches the ArgoCD API version and updates state
func (m *Model) fetchAPIVersion() tea.Cmd {
	if m.state.Server == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		apiService := services.NewArgoApiService(m.state.Server)
		v, err := apiService.GetAPIVersion(ctx, m.state.Server)
		if err != nil {
			return model.StatusChangeMsg{Status: "Version: unknown"}
		}
		return model.SetAPIVersionMsg{Version: v}
	}
}

// consumeWatchEvent reads a single service event and converts it to a tea message
func (m *Model) consumeWatchEvent() tea.Cmd {
	return func() tea.Msg {
		if m.watchChan == nil {
			cblog.With("component", "watch").Debug("consumeWatchEvent: watchChan is nil")
			return nil
		}
		ev, ok := <-m.watchChan
		if !ok {
			cblog.With("component", "watch").Debug("consumeWatchEvent: watchChan closed")
			return nil
		}
		cblog.With("component", "watch").Debug("consumeWatchEvent: received event",
			"type", ev.Type,
			"has_app", ev.App != nil,
			"app_name", ev.AppName)
		switch ev.Type {
		case "apps-loaded":
			if ev.Apps != nil {
				return model.AppsLoadedMsg{Apps: ev.Apps}
			}
		case "app-updated":
			if ev.App != nil {
				cblog.With("component", "watch").Info("Sending AppUpdatedMsg",
					"app_name", ev.App.Name,
					"health", ev.App.Health,
					"sync", ev.App.Sync,
					"resources_count", len(ev.Resources))
				var resourcesData []byte
				if len(ev.Resources) > 0 {
					resourcesData, _ = json.Marshal(ev.Resources)
				}
				return model.AppUpdatedMsg{App: *ev.App, ResourcesJSON: resourcesData}
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
				// If the service emitted a generic api-error but the error is auth-related,
				// surface it as an AuthErrorMsg so the UI switches to auth-required.
				var argErr *apperrors.ArgonautError
				if stdErrors.As(ev.Error, &argErr) {
					// Treat 401/403 as auth-required regardless of category
					if hasHTTPStatusCtx(argErr, 401, 403) || argErr.IsCategory(apperrors.ErrorAuth) || argErr.IsCode("UNAUTHORIZED") || argErr.IsCode("AUTHENTICATION_FAILED") {
						return model.AuthErrorMsg{Error: ev.Error}
					}
					// Forward structured to error view
					return model.StructuredErrorMsg{Error: argErr}
				}
				if isAuthenticationError(ev.Error.Error()) {
					return model.AuthErrorMsg{Error: ev.Error}
				}
				return model.ApiErrorMsg{Message: ev.Error.Error()}
			}
		}
		return nil
	}
}

// startDiffSession loads diffs and opens the diff pager
func (m *Model) startDiffSession(appName string) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error()}
		}

		normalizedDocs := make([]string, 0)
		predictedDocs := make([]string, 0)
		for _, d := range diffs {
			// Filter out hook resources (like ArgoCD UI does)
			if d.Hook {
				continue
			}

			// Use NormalizedLiveState and PredictedLiveState as per ArgoCD spec
			normalizedYAML := ""
			predictedYAML := ""

			if d.NormalizedLiveState != "" {
				normalizedYAML = cleanManifestToYAML(d.NormalizedLiveState)
			}
			if d.PredictedLiveState != "" {
				predictedYAML = cleanManifestToYAML(d.PredictedLiveState)
			}

			// Filter out resources with identical states (like ArgoCD UI does)
			if normalizedYAML == predictedYAML {
				continue
			}

			if normalizedYAML != "" {
				normalizedDocs = append(normalizedDocs, normalizedYAML)
			}
			if predictedYAML != "" {
				predictedDocs = append(predictedDocs, predictedYAML)
			}
		}

		if len(normalizedDocs) == 0 && len(predictedDocs) == 0 {
			// Clear loading spinner before showing no-diff modal
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = false
			return model.SetModeMsg{Mode: model.ModeNoDiff}
		}

		leftFile, _ := writeTempYAML("current-", normalizedDocs)
		rightFile, _ := writeTempYAML("predicted-", predictedDocs)

		// Build raw unified diff via git (no color so delta can format it)
		cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--no-color", "--", leftFile, rightFile)
		out, err := cmd.CombinedOutput()
		if err != nil && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 1 {
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error()}
		}
		cleaned := stripDiffHeader(string(out))
		if strings.TrimSpace(cleaned) == "" {
			// Clear loading spinner before showing no-diff modal
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = false
			return model.SetModeMsg{Mode: model.ModeNoDiff}
		}

		// Clear loading spinner before handing off to viewer/formatter
		if m.state.Diff == nil {
			m.state.Diff = &model.DiffState{}
		}
		m.state.Diff.Loading = false

		// 1) Interactive diff viewer: replace the terminal (e.g., vimdiff, meld)
		if viewer := m.config.GetDiffViewer(); viewer != "" {
			return m.openInteractiveDiffViewer(leftFile, rightFile, viewer)
		}

		// 2) Non-interactive formatter: pipe to tool (e.g., delta) and then show via pager
		formatted := cleaned
		if formattedOut, ferr := m.runDiffFormatterWithTitle(cleaned, appName); ferr == nil && strings.TrimSpace(formattedOut) != "" {
			formatted = formattedOut
		}
		title := fmt.Sprintf("%s - Live vs Desired", appName)
		return m.openTextPager(title, formatted)()
	}
}

// startResourceDiffSession loads the diff for a specific resource and opens the diff pager
func (m *Model) startResourceDiffSession(appName, group, kind, namespace, name string) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error()}
		}

		// Find the matching resource diff
		var targetDiff *services.ResourceDiff
		for i := range diffs {
			d := &diffs[i]
			if d.Group == group && d.Kind == kind && d.Name == name && d.Namespace == namespace {
				targetDiff = d
				break
			}
		}

		if targetDiff == nil || targetDiff.Hook {
			// Clear loading and show no-diff modal
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = false
			return model.SetModeMsg{Mode: model.ModeNoDiff}
		}

		normalizedYAML := ""
		predictedYAML := ""
		if targetDiff.NormalizedLiveState != "" {
			normalizedYAML = cleanManifestToYAML(targetDiff.NormalizedLiveState)
		}
		if targetDiff.PredictedLiveState != "" {
			predictedYAML = cleanManifestToYAML(targetDiff.PredictedLiveState)
		}

		if normalizedYAML == predictedYAML {
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = false
			return model.SetModeMsg{Mode: model.ModeNoDiff}
		}

		// Use existing diff generation infrastructure
		leftFile, _ := writeTempYAML("current-", []string{normalizedYAML})
		rightFile, _ := writeTempYAML("predicted-", []string{predictedYAML})

		cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--no-color", "--", leftFile, rightFile)
		out, err := cmd.CombinedOutput()
		if err != nil && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 1 {
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error()}
		}
		cleaned := stripDiffHeader(string(out))
		if strings.TrimSpace(cleaned) == "" {
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = false
			return model.SetModeMsg{Mode: model.ModeNoDiff}
		}

		// Clear loading before showing
		if m.state.Diff == nil {
			m.state.Diff = &model.DiffState{}
		}
		m.state.Diff.Loading = false

		// Support interactive diff viewer
		if viewer := m.config.GetDiffViewer(); viewer != "" {
			return m.openInteractiveDiffViewer(leftFile, rightFile, viewer)
		}

		// Format and display
		resourceTitle := fmt.Sprintf("%s/%s", kind, name)
		if namespace != "" {
			resourceTitle = fmt.Sprintf("%s/%s/%s", namespace, kind, name)
		}
		formatted := cleaned
		if formattedOut, ferr := m.runDiffFormatterWithTitle(cleaned, resourceTitle); ferr == nil && strings.TrimSpace(formattedOut) != "" {
			formatted = formattedOut
		}
		title := fmt.Sprintf("%s - Live vs Desired", resourceTitle)
		return m.openTextPager(title, formatted)()
	}
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
	// Use kubectl-neat implementation to clean the manifest
	cleaned, err := neat.CleanYAMLToJSON(jsonOrYaml)
	if err != nil {
		// If cleaning fails, return original
		return jsonOrYaml
	}

	// Convert cleaned JSON back to YAML
	var obj interface{}
	if err := json.Unmarshal([]byte(cleaned), &obj); err != nil {
		return jsonOrYaml
	}

	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		return jsonOrYaml
	}

	return string(yamlBytes)
}

// startLoadingResourceTree loads the resource tree for the given app
func (m *Model) startLoadingResourceTree(app model.App) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		argo := services.NewArgoApiService(m.state.Server)
		appNamespace := ""
		if app.AppNamespace != nil {
			appNamespace = *app.AppNamespace
		}
		tree, err := argo.GetResourceTree(ctx, m.state.Server, app.Name, appNamespace)
		if err != nil {
			return model.ApiErrorMsg{Message: err.Error()}
		}
		// Marshal to JSON to avoid import cycle in model messages
		data, merr := json.Marshal(tree)
		if merr != nil {
			return model.ApiErrorMsg{Message: merr.Error()}
		}

		// Also fetch app details to get status.resources for sync status
		var resourcesData []byte
		argoApp, appErr := argo.GetApplication(ctx, m.state.Server, app.Name, app.AppNamespace)
		if appErr == nil && argoApp != nil && len(argoApp.Status.Resources) > 0 {
			resourcesData, _ = json.Marshal(argoApp.Status.Resources)
		}

		return model.ResourceTreeLoadedMsg{
			AppName:       app.Name,
			Health:        app.Health,
			Sync:          app.Sync,
			TreeJSON:      data,
			ResourcesJSON: resourcesData,
		}
	}
}

// startWatchingResourceTree starts a streaming watcher for resource tree updates
type treeWatchStartedMsg struct{ cleanup func() }

func (m *Model) startWatchingResourceTree(app model.App) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return nil
		}
		ctx := context.Background()
		apiService := services.NewArgoApiService(m.state.Server)
		appNamespace := ""
		if app.AppNamespace != nil {
			appNamespace = *app.AppNamespace
		}
		cblog.With("component", "ui").Info("Starting tree watch", "app", app.Name)
		ch, cleanup, err := apiService.WatchResourceTree(ctx, m.state.Server, app.Name, appNamespace)
		if err != nil {
			cblog.With("component", "ui").Error("Tree watch failed", "err", err, "app", app.Name)
			return model.StatusChangeMsg{Status: "Tree watch failed: " + err.Error()}
		}
		go func() {
			eventCount := 0
			for t := range ch {
				if t == nil {
					continue
				}
				eventCount++
				cblog.With("component", "ui").Debug("Received tree event", "app", app.Name, "event", eventCount)
				data, _ := json.Marshal(t)
				m.watchTreeDeliver(model.ResourceTreeStreamMsg{AppName: app.Name, TreeJSON: data})
			}
			cblog.With("component", "ui").Info("Tree watch channel closed", "app", app.Name, "events", eventCount)
		}()
		return treeWatchStartedMsg{cleanup: cleanup}
	}
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
func (m *Model) syncSelectedApplications(prune bool) tea.Cmd {
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

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds max for sync operations
		defer cancel()

		apiService := services.NewEnhancedArgoApiService(m.state.Server)

		for _, appName := range selectedApps {
			err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
			if err != nil {
				// Convert to structured error and return via TUI error handling
				if argErr, ok := err.(*apperrors.ArgonautError); ok {
					return model.StructuredErrorMsg{
						Error:   argErr,
						Context: map[string]interface{}{"operation": "multi-sync", "appName": appName},
						Retry:   argErr.Recoverable,
					}
				}
				// Fallback for non-structured errors
				errorMsg := fmt.Sprintf("Failed to sync %s: %v", appName, err)
				return model.StructuredErrorMsg{
					Error: apperrors.New(apperrors.ErrorAPI, "SYNC_FAILED", errorMsg).
						WithSeverity(apperrors.SeverityHigh).
						AsRecoverable().
						WithUserAction("Check your connection to ArgoCD and try again"),
					Context: map[string]interface{}{"operation": "multi-sync", "appName": appName},
					Retry:   true,
				}
			}
		}

		return model.MultiSyncCompletedMsg{AppCount: len(selectedApps), Success: true}
	}
}

// deleteApplication deletes a specific application
func (m *Model) deleteApplication(req model.AppDeleteRequestMsg) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AppDeleteErrorMsg{
				AppName: req.AppName,
				Error:   "No server configured",
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10 seconds for delete operations
		defer cancel()

		// Create delete service
		deleteService := appdelete.NewAppDeleteService(m.state.Server)

		// Convert to delete request
		deleteReq := appdelete.AppDeleteRequest{
			AppName:           req.AppName,
			AppNamespace:      req.AppNamespace,
			Cascade:           req.Cascade,
			PropagationPolicy: req.PropagationPolicy,
		}

		cblog.With("component", "app-delete").Info("Starting delete", "app", req.AppName, "cascade", req.Cascade)

		// Execute deletion
		response, err := deleteService.DeleteApplication(ctx, m.state.Server, deleteReq)
		if err != nil {
			cblog.With("component", "app-delete").Error("Delete failed", "app", req.AppName, "err", err)
			return model.AppDeleteErrorMsg{
				AppName: req.AppName,
				Error:   err.Error(),
			}
		}

		if !response.Success {
			errorMsg := "Unknown error"
			if response.Error != nil {
				errorMsg = response.Error.Message
			}
			cblog.With("component", "app-delete").Error("Delete returned failure", "app", req.AppName, "error", errorMsg)
			return model.AppDeleteErrorMsg{
				AppName: req.AppName,
				Error:   errorMsg,
			}
		}

		cblog.With("component", "app-delete").Info("Delete completed", "app", req.AppName)
		return model.AppDeleteSuccessMsg{AppName: req.AppName}
	}
}

// syncSingleApplication syncs a specific application
func (m *Model) syncSingleApplication(appName string, prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds max for sync operations
		defer cancel()

		apiService := services.NewEnhancedArgoApiService(m.state.Server)

		cblog.With("component", "api").Info("Starting sync", "app", appName)
		err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
		if err != nil {
			cblog.With("component", "api").Error("Sync failed", "app", appName, "err", err)
			// Convert to structured error and return via TUI error handling
			if argErr, ok := err.(*apperrors.ArgonautError); ok {
				return model.StructuredErrorMsg{
					Error:   argErr,
					Context: map[string]interface{}{"operation": "sync", "appName": appName},
					Retry:   argErr.Recoverable,
				}
			}
			// Fallback for non-structured errors
			errorMsg := fmt.Sprintf("Failed to sync %s: %v", appName, err)
			return model.StructuredErrorMsg{
				Error: apperrors.New(apperrors.ErrorAPI, "SYNC_FAILED", errorMsg).
					WithSeverity(apperrors.SeverityHigh).
					AsRecoverable().
					WithUserAction("Check your connection to ArgoCD and try again"),
				Context: map[string]interface{}{"operation": "sync", "appName": appName},
				Retry:   true,
			}
		}

		cblog.With("component", "api").Info("Sync completed", "app", appName)
		return model.SyncCompletedMsg{AppName: appName, Success: true}
	}
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

// hasHTTPStatusCtx checks ArgonautError.Context for specific HTTP status codes
func hasHTTPStatusCtx(err *apperrors.ArgonautError, statuses ...int) bool {
	if err == nil || err.Context == nil {
		return false
	}
	v, ok := err.Context["statusCode"]
	if !ok {
		return false
	}
	switch n := v.(type) {
	case int:
		for _, s := range statuses {
			if n == s {
				return true
			}
		}
	case int64:
		for _, s := range statuses {
			if int(n) == s {
				return true
			}
		}
	case float64:
		for _, s := range statuses {
			if int(n) == s {
				return true
			}
		}
	}
	return false
}

// startRollbackSession loads deployment history for rollback
func (m *Model) startRollbackSession(appName string) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		// Get application with history
		app, err := apiService.GetApplication(ctx, m.state.Server, appName, nil)
		if err != nil {
			errMsg := err.Error()
			cblog.With("component", "rollback").Error("Rollback session failed", "app", appName, "err", err)
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: "Failed to load application: " + err.Error()}
		}

		cblog.With("component", "rollback").Info("Loaded application history", "app", appName, "count", len(app.Status.History))

		// Convert history to rollback rows
		rows := api.ConvertDeploymentHistoryToRollbackRows(app.Status.History)

		// Get current revision from sync status
		currentRevision := ""
		if app.Status.Sync.Revision != "" {
			currentRevision = app.Status.Sync.Revision
		} else if len(app.Status.Sync.Revisions) > 0 {
			currentRevision = app.Status.Sync.Revisions[0]
		}

		cblog.With("component", "rollback").Debug("Rollback session loaded", "app", appName, "rows", len(rows), "currentRevision", currentRevision)

		return model.RollbackHistoryLoadedMsg{
			AppName:         appName,
			Rows:            rows,
			CurrentRevision: currentRevision,
		}
	}
}

// loadRevisionMetadata loads git metadata for a specific rollback row
func (m *Model) loadRevisionMetadata(appName string, rowIndex int, revision string) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		metadata, err := apiService.GetRevisionMetadata(ctx, m.state.Server, appName, revision, nil)
		if err != nil {
			return model.RollbackMetadataErrorMsg{
				RowIndex: rowIndex,
				Error:    err.Error(),
			}
		}

		return model.RollbackMetadataLoadedMsg{
			RowIndex: rowIndex,
			Metadata: *metadata,
		}
	}
}

// executeRollback performs the actual rollback operation
func (m *Model) executeRollback(request model.RollbackRequest) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		err := apiService.RollbackApplication(ctx, m.state.Server, request)
		if err != nil {
			errMsg := err.Error()
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: "Failed to rollback application: " + err.Error()}
		}

		// Determine if we should watch after rollback
		watchAfter := false
		if m.state.Rollback != nil {
			watchAfter = m.state.Rollback.Watch
		}

		return model.RollbackExecutedMsg{
			AppName: request.Name,
			Success: true,
			Watch:   watchAfter,
		}
	}
}

// startRollbackDiffSession shows diff between current and selected revision
func (m *Model) startRollbackDiffSession(appName string, revision string) tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		// Get diff between current and target revision
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error()}
		}

		// Process diffs (same logic as regular diff)
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
			return model.StatusChangeMsg{Status: "No diffs to show"}
		}

		leftFile, _ := writeTempYAML("live-", liveDocs)
		rightFile, _ := writeTempYAML("rollback-", desiredDocs)

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
		m.state.Diff = &model.DiffState{
			Title:   fmt.Sprintf("Rollback %s to %s", appName, revision[:8]),
			Content: lines,
			Offset:  0,
			Loading: false,
		}
		return model.SetModeMsg{Mode: model.ModeDiff}
	}
}

// deleteSelectedApplications deletes the currently selected applications
func (m *Model) deleteSelectedApplications(cascade bool, propagationPolicy string) tea.Cmd {
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

	return func() tea.Msg {
		cblog.With("component", "app-delete").Info("Starting sequential multi-delete", "count", len(selectedApps), "cascade", cascade, "policy", propagationPolicy)

		// Reasonable timeout for sequential operations
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Create delete service
		deleteService := appdelete.NewAppDeleteService(m.state.Server)

		// Delete applications sequentially to avoid race conditions and dependency issues
		var failedApps []string
		successCount := 0

		for _, appName := range selectedApps {
			cblog.With("component", "app-delete").Debug("Deleting app", "app", appName, "progress", fmt.Sprintf("%d/%d", successCount+len(failedApps)+1, len(selectedApps)))

			// Find the app namespace for this app
			var appNamespace *string
			for _, app := range m.state.Apps {
				if app.Name == appName {
					appNamespace = app.AppNamespace
					break
				}
			}

			err := m.deleteApplicationHelper(ctx, deleteService, appName, appNamespace, cascade, propagationPolicy)
			if err != nil {
				cblog.With("component", "app-delete").Error("Failed to delete app", "app", appName, "err", err)
				failedApps = append(failedApps, fmt.Sprintf("%s (%v)", appName, err))
			} else {
				cblog.With("component", "app-delete").Info("Successfully deleted app", "app", appName)
				successCount++
			}
		}

		// Handle results
		if len(failedApps) > 0 {
			cblog.With("component", "app-delete").Error("Multi-delete partially failed",
				"failed", len(failedApps), "succeeded", successCount, "total", len(selectedApps))
			errorMsg := fmt.Sprintf("Failed to delete %d/%d apps: %s",
				len(failedApps), len(selectedApps), strings.Join(failedApps, ", "))
			return model.AppDeleteErrorMsg{
				AppName: "multiple",
				Error:   errorMsg,
			}
		}

		cblog.With("component", "app-delete").Info("Sequential multi-delete completed successfully", "count", successCount)
		// Clear selections after successful multi-delete
		return model.MultiDeleteCompletedMsg{AppCount: successCount, Success: true}
	}
}

// deleteApplicationHelper performs the actual deletion of a single app
func (m *Model) deleteApplicationHelper(ctx context.Context, deleteService appdelete.AppDeleteService, appName string, namespace *string, cascade bool, propagationPolicy string) error {
	deleteReq := appdelete.AppDeleteRequest{
		AppName:           appName,
		AppNamespace:      namespace,
		Cascade:           cascade,
		PropagationPolicy: propagationPolicy,
	}

	response, err := deleteService.DeleteApplication(ctx, m.state.Server, deleteReq)
	if err != nil {
		return err
	}

	if !response.Success {
		errorMsg := "Unknown error"
		if response.Error != nil {
			errorMsg = response.Error.Message
		}
		return fmt.Errorf("delete failed: %s", errorMsg)
	}

	return nil
}

// deleteSingleApplication deletes a specific application
func (m *Model) deleteSingleApplication(appName string, namespace *string, cascade bool, propagationPolicy string) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AppDeleteErrorMsg{
				AppName: appName,
				Error:   "No server configured",
			}
		}
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // 10 seconds for delete operations
		defer cancel()

		deleteService := appdelete.NewAppDeleteService(m.state.Server)

		if err := m.deleteApplicationHelper(ctx, deleteService, appName, namespace, cascade, propagationPolicy); err != nil {
			return model.AppDeleteErrorMsg{
				AppName: appName,
				Error:   fmt.Sprintf("Failed to delete application: %v", err),
			}
		}

		return model.AppDeleteSuccessMsg{AppName: appName}
	}
}

// deleteSelectedResources deletes the specified resources from the cluster
func (m *Model) deleteSelectedResources(targets []model.ResourceDeleteTarget, cascade bool, propagationPolicy string, force bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ResourceDeleteErrorMsg{Error: "No server configured"}
		}
	}

	if len(targets) == 0 {
		return func() tea.Msg {
			return model.ResourceDeleteErrorMsg{Error: "No resources selected"}
		}
	}

	// Map cascade/propagationPolicy to orphan parameter
	// orphan=true when cascade=false OR propagationPolicy="orphan"
	orphan := !cascade || propagationPolicy == "orphan"

	// Collect unique app names for refresh after deletion
	appNameSet := make(map[string]bool)
	for _, target := range targets {
		appNameSet[target.AppName] = true
	}
	appNames := make([]string, 0, len(appNameSet))
	for name := range appNameSet {
		appNames = append(appNames, name)
	}

	return func() tea.Msg {
		cblog.With("component", "resource-delete").Info("Starting resource deletion",
			"count", len(targets), "cascade", cascade, "policy", propagationPolicy, "orphan", orphan, "force", force)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		appService := api.NewApplicationService(m.state.Server)

		var failedResources []string
		successCount := 0

		for _, target := range targets {
			cblog.With("component", "resource-delete").Debug("Deleting resource",
				"kind", target.Kind, "name", target.Name, "namespace", target.Namespace,
				"version", target.Version, "group", target.Group,
				"progress", fmt.Sprintf("%d/%d", successCount+len(failedResources)+1, len(targets)))

			req := api.DeleteResourceRequest{
				AppName:      target.AppName,
				ResourceName: target.Name,
				Kind:         target.Kind,
				Namespace:    target.Namespace,
				Version:      target.Version,
				Group:        target.Group,
				Orphan:       orphan,
				Force:        force,
			}

			err := appService.DeleteResource(ctx, req)
			if err != nil {
				cblog.With("component", "resource-delete").Error("Failed to delete resource",
					"kind", target.Kind, "name", target.Name, "err", err)
				failedResources = append(failedResources, fmt.Sprintf("%s/%s: %v", target.Kind, target.Name, err))
			} else {
				cblog.With("component", "resource-delete").Info("Successfully deleted resource",
					"kind", target.Kind, "name", target.Name)
				successCount++
			}
		}

		// Handle results
		if len(failedResources) > 0 {
			cblog.With("component", "resource-delete").Error("Resource deletion partially failed",
				"failed", len(failedResources), "succeeded", successCount, "total", len(targets))
			errorMsg := fmt.Sprintf("Failed to delete %d/%d resources: %s",
				len(failedResources), len(targets), strings.Join(failedResources, "; "))
			return model.ResourceDeleteErrorMsg{Error: errorMsg}
		}

		cblog.With("component", "resource-delete").Info("Resource deletion completed successfully", "count", successCount)
		return model.ResourceDeleteSuccessMsg{Count: successCount, AppNames: appNames}
	}
}
