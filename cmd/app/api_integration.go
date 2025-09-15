package main

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"

    tea "github.com/charmbracelet/bubbletea/v2"
    cblog "github.com/charmbracelet/log"
    "github.com/darksworm/argonaut/pkg/api"
    apperrors "github.com/darksworm/argonaut/pkg/errors"
    "github.com/darksworm/argonaut/pkg/model"
    "github.com/darksworm/argonaut/pkg/neat"
    "github.com/darksworm/argonaut/pkg/services"
    yaml "gopkg.in/yaml.v3"
    stdErrors "errors"
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
func (m Model) startDiffSession(appName string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
			return model.StatusChangeMsg{Status: "No diffs"}
		}

		leftFile, _ := writeTempYAML("live-", liveDocs)
		rightFile, _ := writeTempYAML("desired-", desiredDocs)

		// Build raw unified diff via git
		cmd := exec.Command("git", "--no-pager", "diff", "--no-index", "--color=always", "--", leftFile, rightFile)
		out, err := cmd.CombinedOutput()
		if err != nil && cmd.ProcessState != nil && cmd.ProcessState.ExitCode() != 1 {
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error()}
		}
		cleaned := stripDiffHeader(string(out))
		if strings.TrimSpace(cleaned) == "" {
			return model.StatusChangeMsg{Status: "No differences"}
		}

		// Clear loading spinner before handing off to viewer/formatter
		if m.state.Diff == nil {
			m.state.Diff = &model.DiffState{}
		}
		m.state.Diff.Loading = false

		// 1) Interactive diff viewer: replace the terminal (e.g., vimdiff, meld)
		if viewer := os.Getenv("ARGONAUT_DIFF_VIEWER"); viewer != "" {
			return m.openInteractiveDiffViewer(leftFile, rightFile, viewer)
		}

		// 2) Non-interactive formatter: pipe to tool (e.g., delta) and then show via built-in pager (ov)
		formatted := cleaned
		if formattedOut, ferr := m.runDiffFormatter(cleaned); ferr == nil && strings.TrimSpace(formattedOut) != "" {
			formatted = formattedOut
		}
		title := fmt.Sprintf("%s - Live vs Desired", appName)
		return m.openTextPager(title, formatted)()
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
func (m Model) startLoadingResourceTree(app model.App) tea.Cmd {
    return tea.Cmd(func() tea.Msg {
        if m.state.Server == nil {
            return model.ApiErrorMsg{Message: "No server configured"}
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        argo := services.NewArgoApiService(m.state.Server)
        appNamespace := ""
        if app.AppNamespace != nil { appNamespace = *app.AppNamespace }
        tree, err := argo.GetResourceTree(ctx, m.state.Server, app.Name, appNamespace)
        if err != nil {
            return model.ApiErrorMsg{Message: err.Error()}
        }
        // Marshal to JSON to avoid import cycle in model messages
        data, merr := json.Marshal(tree)
        if merr != nil {
            return model.ApiErrorMsg{Message: merr.Error()}
        }
        return model.ResourceTreeLoadedMsg{AppName: app.Name, Health: app.Health, Sync: app.Sync, TreeJSON: data}
    })
}

// startWatchingResourceTree starts a streaming watcher for resource tree updates
func (m Model) startWatchingResourceTree(app model.App) tea.Cmd {
    return tea.Cmd(func() tea.Msg {
        if m.state.Server == nil { return nil }
        ctx := context.Background()
        apiService := services.NewArgoApiService(m.state.Server)
        appNamespace := ""
        if app.AppNamespace != nil { appNamespace = *app.AppNamespace }
        ch, _, err := apiService.WatchResourceTree(ctx, m.state.Server, app.Name, appNamespace)
        if err != nil { return model.StatusChangeMsg{Status: "Tree watch failed: "+err.Error()} }
        go func() {
            for t := range ch {
                if t == nil { continue }
                data, _ := json.Marshal(t)
                m.watchTreeDeliver(model.ResourceTreeStreamMsg{AppName: app.Name, TreeJSON: data})
            }
        }()
        return model.StatusChangeMsg{Status: "Watching tree…"}
    })
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

// hasHTTPStatusCtx checks ArgonautError.Context for specific HTTP status codes
func hasHTTPStatusCtx(err *apperrors.ArgonautError, statuses ...int) bool {
    if err == nil || err.Context == nil { return false }
    v, ok := err.Context["statusCode"]
    if !ok { return false }
    switch n := v.(type) {
    case int:
        for _, s := range statuses { if n == s { return true } }
    case int64:
        for _, s := range statuses { if int(n) == s { return true } }
    case float64:
        for _, s := range statuses { if int(n) == s { return true } }
    }
    return false
}

// startLogsSession opens application logs in pager
func (m Model) startLogsSession() tea.Cmd {
    return tea.Cmd(func() tea.Msg {
        path := os.Getenv("ARGONAUT_LOG_FILE")
        if strings.TrimSpace(path) == "" { path = "logs/a9s.log" }
        data, err := os.ReadFile(path)
        if err != nil {
            return model.ApiErrorMsg{Message: "No logs available"}
        }
        return m.openTextPager("Logs", string(data))()
    })
}

// startRollbackSession loads deployment history for rollback
func (m Model) startRollbackSession(appName string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
	})
}

// loadRevisionMetadata loads git metadata for a specific rollback row
func (m Model) loadRevisionMetadata(appName string, rowIndex int, revision string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
	})
}

// executeRollback performs the actual rollback operation
func (m Model) executeRollback(request model.RollbackRequest) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
	})
}

// startRollbackDiffSession shows diff between current and selected revision
func (m Model) startRollbackDiffSession(appName string, revision string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
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
	})
}
