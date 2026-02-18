package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	stdErrors "errors"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	appcontext "github.com/darksworm/argonaut/pkg/context"
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
		epoch := m.switchEpoch
		return func() tea.Msg {
			return model.AuthErrorMsg{Error: fmt.Errorf("no server configured"), SwitchEpoch: epoch}
		}
	}

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		cblog.With("component", "api_integration").Info("startLoadingApplications: executing load")

		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Load applications with metadata (resourceVersion for watch coordination)
		result, err := apiService.ListApplicationsWithMeta(ctx, m.state.Server)
		if err != nil {
			// Unwrap structured errors if wrapped
			var argErr *apperrors.ArgonautError
			if stdErrors.As(err, &argErr) {
				if argErr.IsCategory(apperrors.ErrorAuth) || argErr.Code == "UNAUTHORIZED" || argErr.Code == "AUTHENTICATION_FAILED" || hasHTTPStatusCtx(argErr, 401, 403) {
					return model.AuthErrorMsg{Error: argErr, SwitchEpoch: epoch}
				}
				// Surface structured errors so error view can show details/context
				return model.StructuredErrorMsg{Error: argErr, SwitchEpoch: epoch}
			}
			// Fallback string matching
			if isAuthenticationError(err.Error()) {
				return model.AuthErrorMsg{Error: err, SwitchEpoch: epoch}
			}
			return model.ApiErrorMsg{Message: err.Error(), SwitchEpoch: epoch}
		}

		// Successfully loaded applications
		return model.AppsLoadedMsg{
			Apps:            result.Apps,
			ResourceVersion: result.ResourceVersion,
			SwitchEpoch:     epoch,
		}
	}
}

// watchStartedMsg indicates the watch stream has started
type watchStartedMsg struct {
	eventChan        <-chan services.ArgoApiEvent
	cleanup          func()
	generation       int
	scopeProjects    []string
	replaceOldWatch  func()
	startSequenceNum int
}

// watchScopeDebounceMsg is sent after 500ms to trigger a scoped watch restart.
// The version field prevents stale debounce ticks from restarting the watch.
type watchScopeDebounceMsg struct {
	version int
}

// startWatchingApplications starts the real-time watch stream
func (m *Model) startWatchingApplications() tea.Cmd {
	return m.startWatchingApplicationsWithConfig(m.watchScopeProjects, m.watchGeneration, nil)
}

func (m *Model) startWatchingApplicationsWithConfig(projects []string, generation int, replaceOldWatch func()) tea.Cmd {
	cblog.With("component", "api_integration").Info("startWatchingApplications called",
		"watchChan_nil", m.watchChan == nil,
		"resourceVersion", m.lastResourceVersion,
		"scopeProjects", projects,
		"generation", generation)
	if m.state.Server == nil {
		return nil
	}

	m.watchStartSequence++
	startSeq := m.watchStartSequence
	// Capture values at call time (before closure executes)
	resourceVersion := m.lastResourceVersion
	capturedProjects := append([]string(nil), projects...)
	capturedGeneration := generation
	capturedReplaceOldWatch := replaceOldWatch
	epoch := m.switchEpoch

	return func() tea.Msg {
		cblog.With("component", "api_integration").Info("startWatchingApplications: executing watch setup",
			"resourceVersion", resourceVersion,
			"projects", capturedProjects,
			"generation", capturedGeneration,
			"start_seq", startSeq)
		// Create context for the watch stream
		ctx := context.Background()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)

		// Build watch options with resourceVersion, field selection, and project filter
		watchOpts := &api.WatchOptions{
			ResourceVersion: resourceVersion,
			Fields:          api.AppWatchFields,
			Projects:        capturedProjects,
		}

		// Start watching applications with options
		eventChan, cleanup, err := apiService.WatchApplicationsWithOptions(ctx, m.state.Server, watchOpts)
		if err != nil {
			// Promote auth-related errors to AuthErrorMsg
			var argErr *apperrors.ArgonautError
			if stdErrors.As(err, &argErr) {
				if hasHTTPStatusCtx(argErr, 401, 403) || argErr.IsCategory(apperrors.ErrorAuth) || argErr.IsCode("UNAUTHORIZED") || argErr.IsCode("AUTHENTICATION_FAILED") {
					return model.AuthErrorMsg{Error: err, SwitchEpoch: epoch}
				}
				return model.StructuredErrorMsg{Error: argErr, SwitchEpoch: epoch}
			}
			if isAuthenticationError(err.Error()) {
				return model.AuthErrorMsg{Error: err, SwitchEpoch: epoch}
			}
			return model.ApiErrorMsg{Message: "Failed to start watch: " + err.Error(), SwitchEpoch: epoch}
		}

		// Return message with the event channel so Update can set it properly
		cblog.With("component", "watch").Info("Watch started successfully, returning watchStartedMsg")
		return watchStartedMsg{
			eventChan:        eventChan,
			cleanup:          cleanup,
			generation:       capturedGeneration,
			scopeProjects:    capturedProjects,
			replaceOldWatch:  capturedReplaceOldWatch,
			startSequenceNum: startSeq,
		}
	}
}

// fetchAPIVersion fetches the ArgoCD API version and updates state
func (m *Model) fetchAPIVersion() tea.Cmd {
	if m.state.Server == nil {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()
		apiService := services.NewArgoApiService(m.state.Server)
		v, err := apiService.GetAPIVersion(ctx, m.state.Server)
		if err != nil {
			return model.StatusChangeMsg{Status: "Version: unknown"}
		}
		return model.SetAPIVersionMsg{Version: v}
	}
}

// eventResult holds the classification of a single watch event
type eventResult struct {
	update     *model.AppUpdatedMsg // non-nil for app-updated events
	deleteName string               // non-empty for app-deleted events
	immediate  tea.Msg              // non-nil for non-batchable events (auth-error, api-error, etc.)
}

func (r eventResult) toBatchOperation() (model.AppBatchOperation, bool) {
	if r.update != nil {
		return model.AppBatchOperation{
			Type:   model.AppBatchOperationUpdate,
			Update: r.update,
		}, true
	}
	if r.deleteName != "" {
		return model.AppBatchOperation{
			Type:   model.AppBatchOperationDelete,
			Delete: r.deleteName,
		}, true
	}
	return model.AppBatchOperation{}, false
}

// classifyWatchEvent converts a service event into an eventResult for batching.
// Batchable events (app-updated, app-deleted) are returned via update/deleteName fields.
// Non-batchable events (auth-error, api-error, etc.) are returned via immediate field.
// epoch is included in all immediate messages so they pass epoch gating when re-dispatched.
func classifyWatchEvent(ev services.ArgoApiEvent, epoch int) eventResult {
	switch ev.Type {
	case "app-updated":
		if ev.App != nil {
			var resourcesData []byte
			if len(ev.Resources) > 0 {
				resourcesData, _ = json.Marshal(ev.Resources)
			}
			return eventResult{update: &model.AppUpdatedMsg{App: *ev.App, ResourcesJSON: resourcesData}}
		}
	case "app-deleted":
		if ev.AppName != "" {
			return eventResult{deleteName: ev.AppName}
		}
	case "apps-loaded":
		if ev.Apps != nil {
			return eventResult{immediate: model.AppsLoadedMsg{Apps: ev.Apps, SwitchEpoch: epoch}}
		}
	case "status-change":
		if ev.Status != "" {
			return eventResult{immediate: model.StatusChangeMsg{Status: ev.Status}}
		}
	case "auth-error":
		if ev.Error != nil {
			return eventResult{immediate: model.AuthErrorMsg{Error: ev.Error, SwitchEpoch: epoch}}
		}
	case "api-error":
		if ev.Error != nil {
			var argErr *apperrors.ArgonautError
			if stdErrors.As(ev.Error, &argErr) {
				if hasHTTPStatusCtx(argErr, 401, 403) || argErr.IsCategory(apperrors.ErrorAuth) || argErr.IsCode("UNAUTHORIZED") || argErr.IsCode("AUTHENTICATION_FAILED") {
					return eventResult{immediate: model.AuthErrorMsg{Error: ev.Error, SwitchEpoch: epoch}}
				}
				return eventResult{immediate: model.StructuredErrorMsg{Error: argErr, SwitchEpoch: epoch}}
			}
			if isAuthenticationError(ev.Error.Error()) {
				return eventResult{immediate: model.AuthErrorMsg{Error: ev.Error, SwitchEpoch: epoch}}
			}
			return eventResult{immediate: model.ApiErrorMsg{Message: ev.Error.Error(), SwitchEpoch: epoch}}
		}
	}
	return eventResult{}
}

// consumeWatchEvents reads events from the watch channel, batching app-updated
// and app-deleted events for up to 500ms to reduce render cycles.
// Non-batchable events (auth-error, api-error, etc.) are returned immediately
// or included in the batch's Immediate field if encountered during batching.
//
// IMPORTANT: ch and gen are captured at call time so that if the watch is restarted
// (m.watchChan replaced), this closure operates on the old channel and produces a
// batch tagged with the old generation. The handler checks the generation to avoid
// spawning duplicate consumers for the new channel.
func (m *Model) consumeWatchEvents() tea.Cmd {
	ch := m.watchChan        // capture at call time
	gen := m.watchGeneration // capture at call time
	done := m.watchDone      // capture at call time
	epoch := m.switchEpoch   // capture at call time
	return func() tea.Msg {
		if ch == nil {
			cblog.With("component", "watch").Debug("consumeWatchEvents: watchChan is nil")
			return nil
		}

		// Block on first event (also listen on done so we don't block forever
		// when cleanupAppWatcher stops the forwarding goroutine).
		var (
			ev services.ArgoApiEvent
			ok bool
		)
		select {
		case ev, ok = <-ch:
			if !ok {
				cblog.With("component", "watch").Debug("consumeWatchEvents: watchChan closed")
				return nil
			}
		case <-done:
			cblog.With("component", "watch").Debug("consumeWatchEvents: watchDone signaled")
			return nil
		}

		result := classifyWatchEvent(ev, epoch)

		// If first event is non-batchable, wrap it in a batch so the
		// AppsBatchUpdateMsg handler continues the watch consumer chain.
		if result.immediate != nil {
			return model.AppsBatchUpdateMsg{
				Immediate:   result.immediate,
				Generation:  gen,
				SwitchEpoch: epoch,
			}
		}

		// Start batching
		var updates []model.AppUpdatedMsg
		var deletes []string
		var operations []model.AppBatchOperation
		if result.update != nil {
			updates = append(updates, *result.update)
		}
		if result.deleteName != "" {
			deletes = append(deletes, result.deleteName)
		}
		if op, ok := result.toBatchOperation(); ok {
			operations = append(operations, op)
		}

		// Drain for up to 500ms
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()

		var immediate tea.Msg
	loop:
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					break loop
				}
				result := classifyWatchEvent(ev, epoch)
				if result.immediate != nil {
					immediate = result.immediate
					break loop
				}
				if result.update != nil {
					updates = append(updates, *result.update)
				}
				if result.deleteName != "" {
					deletes = append(deletes, result.deleteName)
				}
				if op, ok := result.toBatchOperation(); ok {
					operations = append(operations, op)
				}
			case <-timer.C:
				break loop
			}
		}

		cblog.With("component", "watch").Debug("consumeWatchEvents: batch complete",
			"updates", len(updates),
			"deletes", len(deletes),
			"has_immediate", immediate != nil,
			"generation", gen)

		return model.AppsBatchUpdateMsg{
			Updates:     updates,
			Deletes:     deletes,
			Operations:  operations,
			Immediate:   immediate,
			Generation:  gen,
			SwitchEpoch: epoch,
		}
	}
}

// maybeRestartWatchForScope checks if the current watch stream's project filter
// differs from the user's active ScopeProjects selection. If different, it schedules
// a debounced watch restart (500ms) to avoid thrashing during rapid navigation.
func (m *Model) maybeRestartWatchForScope() tea.Cmd {
	newProjects := sortedScopeProjects(m.state.Selections.ScopeProjects)
	if stringSlicesEqual(newProjects, m.watchScopeProjects) {
		return nil // No change
	}
	cblog.With("component", "watch").Info("maybeRestartWatchForScope: scope changed, scheduling restart",
		"current", m.watchScopeProjects,
		"new", newProjects)
	m.scopeVersion++
	version := m.scopeVersion
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return watchScopeDebounceMsg{version: version}
	})
}

// restartWatchWithScope stops the current watch stream and starts a new one
// filtered to the active ScopeProjects. This reduces ongoing SSE traffic when
// the user has drilled down to specific projects.
func (m *Model) restartWatchWithScope() tea.Cmd {
	newProjects := sortedScopeProjects(m.state.Selections.ScopeProjects)
	targetGeneration := m.watchGeneration + 1
	cblog.With("component", "watch").Info("restartWatchWithScope: restarting watch",
		"projects", newProjects,
		"target_generation", targetGeneration)

	// Start the new watch first; once it is confirmed via watchStartedMsg,
	// we stop the old stream to avoid dropping updates during setup failures.
	return m.startWatchingApplicationsWithConfig(newProjects, targetGeneration, m.watchCleanup)
}

// sortedScopeProjects extracts project names from a scope map and returns them sorted.
func sortedScopeProjects(scope map[string]bool) []string {
	var projects []string
	for p, ok := range scope {
		if ok {
			projects = append(projects, p)
		}
	}
	sort.Strings(projects)
	return projects
}

// stringSlicesEqual returns true if two string slices have equal content.
func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// startDiffSession loads diffs and opens the diff pager
func (m *Model) startDiffSession(appName string) tea.Cmd {
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured", SwitchEpoch: epoch}
		}

		ctx, cancel := appcontext.WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error(), SwitchEpoch: epoch}
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
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error(), SwitchEpoch: epoch}
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
func (m *Model) startResourceDiffSession(res ResourceIdentifier) tea.Cmd {
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured", SwitchEpoch: epoch}
		}

		ctx, cancel := appcontext.WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, res.AppName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error(), SwitchEpoch: epoch}
		}

		// Find the matching resource diff
		var targetDiff *services.ResourceDiff
		for i := range diffs {
			d := &diffs[i]
			if d.Group == res.Group && d.Kind == res.Kind && d.Name == res.Name && d.Namespace == res.Namespace {
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
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error(), SwitchEpoch: epoch}
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
		resourceTitle := fmt.Sprintf("%s/%s", res.Kind, res.Name)
		if res.Namespace != "" {
			resourceTitle = fmt.Sprintf("%s/%s/%s", res.Namespace, res.Kind, res.Name)
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
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()

		argo := services.NewArgoApiService(m.state.Server)
		appNamespace := ""
		if app.AppNamespace != nil {
			appNamespace = *app.AppNamespace
		}
		tree, err := argo.GetResourceTree(ctx, m.state.Server, app.Name, appNamespace)
		if err != nil {
			return model.ApiErrorMsg{Message: err.Error(), SwitchEpoch: epoch}
		}
		// Marshal to JSON to avoid import cycle in model messages
		data, merr := json.Marshal(tree)
		if merr != nil {
			return model.ApiErrorMsg{Message: merr.Error(), SwitchEpoch: epoch}
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
			SwitchEpoch:   epoch,
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

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		apiService := services.NewEnhancedArgoApiService(m.state.Server)

		for _, appName := range selectedApps {
			ctx, cancel := appcontext.WithAPITimeout(context.Background())
			err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
			cancel()
			if err != nil {
				// Convert to structured error and return via TUI error handling
				if argErr, ok := err.(*apperrors.ArgonautError); ok {
					return model.StructuredErrorMsg{
						Error:       argErr,
						Context:     map[string]interface{}{"operation": "multi-sync", "appName": appName},
						Retry:       argErr.Recoverable,
						SwitchEpoch: epoch,
					}
				}
				// Fallback for non-structured errors
				errorMsg := fmt.Sprintf("Failed to sync %s: %v", appName, err)
				return model.StructuredErrorMsg{
					Error: apperrors.New(apperrors.ErrorAPI, "SYNC_FAILED", errorMsg).
						WithSeverity(apperrors.SeverityHigh).
						AsRecoverable().
						WithUserAction("Check your connection to ArgoCD and try again"),
					Context:     map[string]interface{}{"operation": "multi-sync", "appName": appName},
					Retry:       true,
					SwitchEpoch: epoch,
				}
			}
		}

		return model.MultiSyncCompletedMsg{AppCount: len(selectedApps), Success: true, SwitchEpoch: epoch}
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

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
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
		return model.AppDeleteSuccessMsg{AppName: req.AppName, SwitchEpoch: epoch}
	}
}

// syncSingleApplication syncs a specific application
func (m *Model) syncSingleApplication(appName string, prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()

		apiService := services.NewEnhancedArgoApiService(m.state.Server)

		cblog.With("component", "api").Info("Starting sync", "app", appName)
		err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
		if err != nil {
			cblog.With("component", "api").Error("Sync failed", "app", appName, "err", err)
			// Convert to structured error and return via TUI error handling
			if argErr, ok := err.(*apperrors.ArgonautError); ok {
				return model.StructuredErrorMsg{
					Error:       argErr,
					Context:     map[string]interface{}{"operation": "sync", "appName": appName},
					Retry:       argErr.Recoverable,
					SwitchEpoch: epoch,
				}
			}
			// Fallback for non-structured errors
			errorMsg := fmt.Sprintf("Failed to sync %s: %v", appName, err)
			return model.StructuredErrorMsg{
				Error: apperrors.New(apperrors.ErrorAPI, "SYNC_FAILED", errorMsg).
					WithSeverity(apperrors.SeverityHigh).
					AsRecoverable().
					WithUserAction("Check your connection to ArgoCD and try again"),
				Context:     map[string]interface{}{"operation": "sync", "appName": appName},
				Retry:       true,
				SwitchEpoch: epoch,
			}
		}

		cblog.With("component", "api").Info("Sync completed", "app", appName)
		return model.SyncCompletedMsg{AppName: appName, Success: true, SwitchEpoch: epoch}
	}
}

// refreshSingleApplication refreshes a specific application
func (m *Model) refreshSingleApplication(appName string, appNamespace *string, hard bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()

		appService := api.NewApplicationService(m.state.Server)

		opts := &api.RefreshOptions{
			Hard:         hard,
			AppNamespace: appNamespace,
		}

		refreshType := "refresh"
		if hard {
			refreshType = "hard refresh"
		}

		cblog.With("component", "api").Info("Starting "+refreshType, "app", appName)
		err := appService.RefreshApplication(ctx, appName, opts)
		if err != nil {
			cblog.With("component", "api").Error("Refresh failed", "app", appName, "err", err)
			if argErr, ok := err.(*apperrors.ArgonautError); ok {
				return model.StructuredErrorMsg{
					Error:       argErr,
					Context:     map[string]interface{}{"operation": "refresh", "appName": appName, "hard": hard},
					Retry:       argErr.Recoverable,
					SwitchEpoch: epoch,
				}
			}
			errorMsg := fmt.Sprintf("Failed to refresh %s: %v", appName, err)
			return model.StructuredErrorMsg{
				Error: apperrors.New(apperrors.ErrorAPI, "REFRESH_FAILED", errorMsg).
					WithSeverity(apperrors.SeverityMedium).
					AsRecoverable().
					WithUserAction("Check your connection to ArgoCD and try again"),
				Context:     map[string]interface{}{"operation": "refresh", "appName": appName, "hard": hard},
				Retry:       true,
				SwitchEpoch: epoch,
			}
		}

		cblog.With("component", "api").Info("Refresh completed", "app", appName, "hard", hard)
		return model.RefreshCompletedMsg{AppName: appName, Success: true, Hard: hard}
	}
}

// refreshMultipleApplications refreshes multiple selected applications
func (m *Model) refreshMultipleApplications(hard bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	selectedApps := make([]string, 0, len(m.state.Selections.SelectedApps))
	for appName, selected := range m.state.Selections.SelectedApps {
		if selected {
			selectedApps = append(selectedApps, appName)
		}
	}

	if len(selectedApps) == 0 {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No applications selected"}
		}
	}

	// Build app namespace map
	appNamespaces := make(map[string]*string)
	for _, app := range m.state.Apps {
		appNamespaces[app.Name] = app.AppNamespace
	}

	return func() tea.Msg {
		appService := api.NewApplicationService(m.state.Server)

		for _, appName := range selectedApps {
			ctx, cancel := appcontext.WithAPITimeout(context.Background())
			opts := &api.RefreshOptions{
				Hard:         hard,
				AppNamespace: appNamespaces[appName],
			}

			err := appService.RefreshApplication(ctx, appName, opts)
			cancel()
			if err != nil {
				cblog.With("component", "api").Error("Refresh failed for app", "app", appName, "err", err)
				// Continue with other apps
			}
		}

		return model.MultiRefreshCompletedMsg{AppCount: len(selectedApps), Success: true, Hard: hard}
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
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured", SwitchEpoch: epoch}
		}

		ctx, cancel := appcontext.WithMinAPITimeout(context.Background(), 30*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		// Get application with history
		app, err := apiService.GetApplication(ctx, m.state.Server, appName, nil)
		if err != nil {
			errMsg := err.Error()
			cblog.With("component", "rollback").Error("Rollback session failed", "app", appName, "err", err)
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err, SwitchEpoch: epoch}
			}
			return model.ApiErrorMsg{Message: "Failed to load application: " + err.Error(), SwitchEpoch: epoch}
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

		ctx, cancel := appcontext.WithAPITimeout(context.Background())
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
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured", SwitchEpoch: epoch}
		}

		ctx, cancel := appcontext.WithMinAPITimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		err := apiService.RollbackApplication(ctx, m.state.Server, request)
		if err != nil {
			errMsg := err.Error()
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err, SwitchEpoch: epoch}
			}
			return model.ApiErrorMsg{Message: "Failed to rollback application: " + err.Error(), SwitchEpoch: epoch}
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
	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		if m.state.Server == nil {
			return model.ApiErrorMsg{Message: "No server configured", SwitchEpoch: epoch}
		}

		ctx, cancel := appcontext.WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)

		// Get diff between current and target revision
		diffs, err := apiService.GetResourceDiffs(ctx, m.state.Server, appName)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to load diffs: " + err.Error(), SwitchEpoch: epoch}
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
			return model.ApiErrorMsg{Message: "Diff failed: " + err.Error(), SwitchEpoch: epoch}
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

			ctx, cancel := appcontext.WithAPITimeout(context.Background())
			err := m.deleteApplicationHelper(ctx, deleteService, AppDeleteParams{
				AppName:   appName,
				Namespace: appNamespace,
				Options: DeleteOptions{
					Cascade:           cascade,
					PropagationPolicy: propagationPolicy,
				},
			})
			cancel()
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
func (m *Model) deleteApplicationHelper(ctx context.Context, deleteService appdelete.AppDeleteService, params AppDeleteParams) error {
	deleteReq := appdelete.AppDeleteRequest{
		AppName:           params.AppName,
		AppNamespace:      params.Namespace,
		Cascade:           params.Options.Cascade,
		PropagationPolicy: params.Options.PropagationPolicy,
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
func (m *Model) deleteSingleApplication(params AppDeleteParams) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AppDeleteErrorMsg{
				AppName: params.AppName,
				Error:   "No server configured",
			}
		}
	}

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		ctx, cancel := appcontext.WithAPITimeout(context.Background())
		defer cancel()

		deleteService := appdelete.NewAppDeleteService(m.state.Server)

		if err := m.deleteApplicationHelper(ctx, deleteService, params); err != nil {
			return model.AppDeleteErrorMsg{
				AppName: params.AppName,
				Error:   fmt.Sprintf("Failed to delete application: %v", err),
			}
		}

		return model.AppDeleteSuccessMsg{AppName: params.AppName, SwitchEpoch: epoch}
	}
}

// deleteSelectedResources deletes the specified resources from the cluster
func (m *Model) deleteSelectedResources(targets []model.ResourceDeleteTarget, opts DeleteOptions) tea.Cmd {
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
	orphan := !opts.Cascade || opts.PropagationPolicy == "orphan"

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
			"count", len(targets), "cascade", opts.Cascade, "policy", opts.PropagationPolicy, "orphan", orphan, "force", opts.Force)

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
				Force:        opts.Force,
			}

			ctx, cancel := appcontext.WithAPITimeout(context.Background())
			err := appService.DeleteResource(ctx, req)
			cancel()
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

// extractUserFriendlyError extracts a user-friendly error message from an error chain.
// It looks for ArgonautError in the chain and returns its Message field,
// which typically contains the ArgoCD error message parsed from the API response.
func extractUserFriendlyError(err error) string {
	if err == nil {
		return ""
	}

	// Try to unwrap to ArgonautError
	var argoErr *apperrors.ArgonautError
	if stdErrors.As(err, &argoErr) {
		// The Message field contains the parsed ArgoCD error (or our default message)
		return argoErr.Message
	}

	// Fallback: return the error string, but try to clean it up
	errStr := err.Error()
	// Remove common wrapper prefixes
	prefixes := []string{
		"failed to sync application ",
	}
	for _, prefix := range prefixes {
		if idx := strings.Index(errStr, prefix); idx != -1 {
			// Find the actual error after the wrapper
			afterPrefix := errStr[idx+len(prefix):]
			if colonIdx := strings.Index(afterPrefix, ": "); colonIdx != -1 {
				return strings.TrimSpace(afterPrefix[colonIdx+2:])
			}
		}
	}

	return errStr
}

// syncSelectedResources syncs the specified resources via ArgoCD
func (m *Model) syncSelectedResources(targets []model.ResourceSyncTarget, prune, force bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ResourceSyncErrorMsg{Error: "No server configured"}
		}
	}

	if len(targets) == 0 {
		return func() tea.Msg {
			return model.ResourceSyncErrorMsg{Error: "No resources selected"}
		}
	}

	// Group targets by app name (ArgoCD API requires separate calls per app)
	appResources := make(map[string][]api.SyncResourceTarget)
	for _, target := range targets {
		appResources[target.AppName] = append(appResources[target.AppName], api.SyncResourceTarget{
			Group:     target.Group,
			Kind:      target.Kind,
			Name:      target.Name,
			Namespace: target.Namespace,
		})
	}

	// Collect unique app names for refresh after sync
	appNames := make([]string, 0, len(appResources))
	for name := range appResources {
		appNames = append(appNames, name)
	}

	epoch := m.switchEpoch // capture at call time
	return func() tea.Msg {
		cblog.With("component", "resource-sync").Info("Starting resource sync",
			"count", len(targets), "apps", len(appResources), "prune", prune, "force", force)

		appService := api.NewApplicationService(m.state.Server)

		var failedApps []string
		successCount := 0

		for appName, resources := range appResources {
			cblog.With("component", "resource-sync").Debug("Syncing resources for app",
				"app", appName, "resourceCount", len(resources))

			// Look up app namespace from state (required for namespaced ArgoCD installations)
			var appNamespace string
			for _, app := range m.state.Apps {
				if app.Name == appName && app.AppNamespace != nil {
					appNamespace = *app.AppNamespace
					break
				}
			}

			opts := &api.SyncOptions{
				Prune:        prune,
				Force:        force,
				Resources:    resources,
				AppNamespace: appNamespace,
			}

			ctx, cancel := appcontext.WithAPITimeout(context.Background())
			err := appService.SyncApplication(ctx, appName, opts)
			cancel()
			if err != nil {
				// Extract user-friendly message from the error chain
				errMsg := extractUserFriendlyError(err)
				cblog.With("component", "resource-sync").Error("Failed to sync resources for app",
					"app", appName, "err", err, "userMsg", errMsg)
				failedApps = append(failedApps, errMsg)
			} else {
				cblog.With("component", "resource-sync").Info("Successfully synced resources for app",
					"app", appName, "resourceCount", len(resources))
				successCount += len(resources)
			}
		}

		// Handle results
		if len(failedApps) > 0 {
			cblog.With("component", "resource-sync").Error("Resource sync partially failed",
				"failed", len(failedApps), "apps", len(appResources))
			errorMsg := fmt.Sprintf("Failed to sync resources: %s", strings.Join(failedApps, "; "))
			return model.ResourceSyncErrorMsg{Error: errorMsg}
		}

		cblog.With("component", "resource-sync").Info("Resource sync completed successfully", "count", successCount)
		return model.ResourceSyncSuccessMsg{Count: successCount, AppNames: appNames, SwitchEpoch: epoch}
	}
}
