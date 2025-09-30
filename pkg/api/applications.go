package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// ArgoApplication represents an ArgoCD application from the API
type ArgoApplication struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"metadata"`
	Spec struct {
		Project string `json:"project,omitempty"`
		// Single source (legacy/traditional)
		Source *struct {
			RepoURL        string `json:"repoURL,omitempty"`
			Path           string `json:"path,omitempty"`
			TargetRevision string `json:"targetRevision,omitempty"`
		} `json:"source,omitempty"`
		// Multiple sources (newer multi-source support)
		Sources []struct {
			RepoURL        string `json:"repoURL,omitempty"`
			Path           string `json:"path,omitempty"`
			TargetRevision string `json:"targetRevision,omitempty"`
		} `json:"sources,omitempty"`
		Destination struct {
			Name      string `json:"name,omitempty"`
			Server    string `json:"server,omitempty"`
			Namespace string `json:"namespace,omitempty"`
		} `json:"destination"`
	} `json:"spec"`
	Status struct {
		Sync struct {
			Status     string `json:"status,omitempty"`
			ComparedTo struct {
				Source *struct {
					RepoURL        string `json:"repoURL,omitempty"`
					Path           string `json:"path,omitempty"`
					TargetRevision string `json:"targetRevision,omitempty"`
				} `json:"source,omitempty"`
				Sources []struct {
					RepoURL        string `json:"repoURL,omitempty"`
					Path           string `json:"path,omitempty"`
					TargetRevision string `json:"targetRevision,omitempty"`
				} `json:"sources,omitempty"`
			} `json:"comparedTo"`
			Revision  string   `json:"revision,omitempty"`
			Revisions []string `json:"revisions,omitempty"`
		} `json:"sync"`
		Health struct {
			Status  string `json:"status,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"health"`
		OperationState struct {
			Phase      string    `json:"phase,omitempty"`
			StartedAt  time.Time `json:"startedAt,omitempty"`
			FinishedAt time.Time `json:"finishedAt,omitempty"`
		} `json:"operationState,omitempty"`
		History []DeploymentHistory `json:"history,omitempty"`
	} `json:"status"`
}

// ApplicationWatchEvent represents an event from the watch stream
type ApplicationWatchEvent struct {
	Type        string          `json:"type"`
	Application ArgoApplication `json:"application"`
}

// WatchEventResult wraps the watch event in the expected format
type WatchEventResult struct {
	Result ApplicationWatchEvent `json:"result"`
}

// ListApplicationsResponse represents the response from listing applications
type ListApplicationsResponse struct {
	Items []ArgoApplication `json:"items"`
}

// DeploymentHistory represents a deployment history entry from ArgoCD API
type DeploymentHistory struct {
	ID         int       `json:"id"`
	Revision   string    `json:"revision"`
	DeployedAt time.Time `json:"deployedAt"`
	Source     *struct {
		RepoURL        string `json:"repoURL,omitempty"`
		Path           string `json:"path,omitempty"`
		TargetRevision string `json:"targetRevision,omitempty"`
	} `json:"source,omitempty"`
}

// RevisionMetadataResponse represents git metadata response from ArgoCD API
type RevisionMetadataResponse struct {
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Tags    []string  `json:"tags,omitempty"`
}

// ManagedResourceDiff represents ArgoCD managed resource diff item
type ManagedResourceDiff struct {
	Group               string `json:"group,omitempty"`
	Kind                string `json:"kind,omitempty"`
	Namespace           string `json:"namespace,omitempty"`
	Name                string `json:"name,omitempty"`
	TargetState         string `json:"targetState,omitempty"`
	LiveState           string `json:"liveState,omitempty"`
	Diff                string `json:"diff,omitempty"`
	Hook                bool   `json:"hook,omitempty"`
	NormalizedLiveState string `json:"normalizedLiveState,omitempty"`
	PredictedLiveState  string `json:"predictedLiveState,omitempty"`
}

// ManagedResourcesResponse represents response for managed resources
type ManagedResourcesResponse struct {
	Items []ManagedResourceDiff `json:"items"`
}

// ApplicationService provides ArgoCD application operations
type ApplicationService struct {
	client *Client
}

// NewApplicationService creates a new application service
func NewApplicationService(server *model.Server) *ApplicationService {
	return &ApplicationService{
		client: NewClient(server),
	}
}

// ListApplications retrieves all applications from ArgoCD
func (s *ApplicationService) ListApplications(ctx context.Context) ([]model.App, error) {
	data, err := s.client.Get(ctx, "/api/v1/applications")
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// First, try to parse as { items: [...] }
	var withItems struct {
		Items []json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(data, &withItems); err != nil {
		return nil, fmt.Errorf("failed to parse applications response: %w", err)
	}

	var rawItems []json.RawMessage
	if len(withItems.Items) > 0 {
		rawItems = withItems.Items
	} else {
		// Some servers may return a bare array instead of an object with items
		if err := json.Unmarshal(data, &rawItems); err != nil {
			return nil, fmt.Errorf("failed to parse applications array: %w", err)
		}
	}

	apps := make([]model.App, 0, len(rawItems))
	for _, raw := range rawItems {
		// Unmarshal into our typed struct first
		var argoApp ArgoApplication
		if err := json.Unmarshal(raw, &argoApp); err != nil {
			// Skip malformed entry
			continue
		}

		app := s.ConvertToApp(argoApp)

		// Fallback: if sync/health are empty, extract directly from raw JSON
		if app.Sync == "" || app.Health == "" || app.Sync == "Unknown" || app.Health == "Unknown" {
			var root map[string]interface{}
			if err := json.Unmarshal(raw, &root); err == nil {
				if sMap, ok := root["status"].(map[string]interface{}); ok {
					if app.Sync == "" || app.Sync == "Unknown" {
						if syncMap, ok := sMap["sync"].(map[string]interface{}); ok {
							if v, ok := syncMap["status"].(string); ok && v != "" {
								app.Sync = v
							}
						}
					}
					if app.Health == "" || app.Health == "Unknown" {
						if healthMap, ok := sMap["health"].(map[string]interface{}); ok {
							if v, ok := healthMap["status"].(string); ok && v != "" {
								app.Health = v
							}
						}
					}
				}
			}
			if app.Sync == "" {
				app.Sync = "Unknown"
			}
			if app.Health == "" {
				app.Health = "Unknown"
			}
		}

		apps = append(apps, app)
	}

	return apps, nil
}

// GetManagedResourceDiffs fetches managed resource diffs for an application
func (s *ApplicationService) GetManagedResourceDiffs(ctx context.Context, appName string) ([]ManagedResourceDiff, error) {
	if appName == "" {
		return nil, fmt.Errorf("application name is required")
	}
	path := fmt.Sprintf("/api/v1/applications/%s/managed-resources", url.PathEscape(appName))
	data, err := s.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get managed resources: %w", err)
	}

	// Accept both {items:[...]} and bare array
	var withItems ManagedResourcesResponse
	if err := json.Unmarshal(data, &withItems); err == nil && len(withItems.Items) > 0 {
		return withItems.Items, nil
	}
	var arr []ManagedResourceDiff
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	return []ManagedResourceDiff{}, nil
}

// SyncApplication triggers a sync for the specified application
func (s *ApplicationService) SyncApplication(ctx context.Context, appName string, opts *SyncOptions) error {
	if opts == nil {
		opts = &SyncOptions{}
	}

	reqBody := map[string]interface{}{
		"prune":        opts.Prune,
		"dryRun":       opts.DryRun,
		"appNamespace": opts.AppNamespace,
	}

	path := fmt.Sprintf("/api/v1/applications/%s/sync", url.PathEscape(appName))
	if opts.AppNamespace != "" {
		path += "?appNamespace=" + url.QueryEscape(opts.AppNamespace)
	}

	_, err := s.client.Post(ctx, path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to sync application %s: %w", appName, err)
	}

	return nil
}

// WatchApplications starts watching for application changes
func (s *ApplicationService) WatchApplications(ctx context.Context, eventChan chan<- ApplicationWatchEvent) error {
	cblog.With("component", "api").Info("WatchApplications: attempting to establish stream", "endpoint", "/api/v1/stream/applications")
	stream, err := s.client.Stream(ctx, "/api/v1/stream/applications")
	if err != nil {
		cblog.With("component", "api").Error("WatchApplications: failed to establish stream", "error", err)
		return fmt.Errorf("failed to start watch stream: %w", err)
	}
	cblog.With("component", "api").Info("WatchApplications: stream established successfully")
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	cblog.With("component", "api").Info("WatchApplications: starting to read from stream")
	for scanner.Scan() {
		if ctx.Err() != nil {
			cblog.With("component", "api").Debug("WatchApplications: context cancelled")
			return ctx.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// SSE format: lines with just ":" are keep-alive messages
		if line == ":" {
			continue // Keep-alive message
		}

		cblog.With("component", "api").Debug("WatchApplications: received line from stream", "line", line)

		// Handle Server-Sent Events format (lines starting with "data: ")
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
		} else {
			// Skip non-data lines
			cblog.With("component", "api").Debug("WatchApplications: skipping non-data line", "line", line)
			continue
		}

		var eventResult WatchEventResult
		if err := json.Unmarshal([]byte(line), &eventResult); err != nil {
			cblog.With("component", "api").Warn("WatchApplications: failed to unmarshal event", "error", err, "line", line)
			// Skip malformed lines
			continue
		}
		cblog.With("component", "api").Debug("WatchApplications: parsed event", "type", eventResult.Result.Type, "app", eventResult.Result.Application.Metadata.Name)

		select {
		case eventChan <- eventResult.Result:
			cblog.With("component", "api").Debug("WatchApplications: sent event to channel")
		case <-ctx.Done():
			cblog.With("component", "api").Debug("WatchApplications: context cancelled during send")
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream scanning error: %w", err)
	}

	return nil
}

// SyncOptions represents options for syncing an application
type SyncOptions struct {
	Prune        bool   `json:"prune,omitempty"`
	DryRun       bool   `json:"dryRun,omitempty"`
	AppNamespace string `json:"appNamespace,omitempty"`
}

// ConvertToApp converts an ArgoApplication to our model.App
func (s *ApplicationService) ConvertToApp(argoApp ArgoApplication) model.App {
	app := model.App{
		Name:   argoApp.Metadata.Name,
		Sync:   argoApp.Status.Sync.Status,
		Health: argoApp.Status.Health.Status,
	}

	// Set optional fields
	if argoApp.Spec.Project != "" {
		app.Project = &argoApp.Spec.Project
	}

	if argoApp.Metadata.Namespace != "" {
		app.AppNamespace = &argoApp.Metadata.Namespace
	}

	if argoApp.Spec.Destination.Namespace != "" {
		app.Namespace = &argoApp.Spec.Destination.Namespace
	}

	// Extract cluster info preferring destination.name, else from destination.server host
	if argoApp.Spec.Destination.Name != "" || argoApp.Spec.Destination.Server != "" {
		var id string
		var label string
		if argoApp.Spec.Destination.Name != "" {
			id = argoApp.Spec.Destination.Name
			label = id
		} else {
			server := argoApp.Spec.Destination.Server
			if server == "https://kubernetes.default.svc" {
				id = "in-cluster"
				label = id
			} else {
				if u, err := url.Parse(server); err == nil && u.Host != "" {
					id = u.Host
					label = u.Host
				} else {
					id = server
					label = server
				}
			}
		}
		app.ClusterID = &id
		app.ClusterLabel = &label
	}

	// Handle sync timestamp
	if !argoApp.Status.OperationState.FinishedAt.IsZero() {
		app.LastSyncAt = &argoApp.Status.OperationState.FinishedAt
	} else if !argoApp.Status.OperationState.StartedAt.IsZero() {
		app.LastSyncAt = &argoApp.Status.OperationState.StartedAt
	}

	// Normalize status values to match TypeScript app
	if app.Sync == "" {
		app.Sync = "Unknown"
	}
	if app.Health == "" {
		app.Health = "Unknown"
	}

	return app
}

// HasMultipleSources returns true if the application uses multiple sources
func (app *ArgoApplication) HasMultipleSources() bool {
	return len(app.Spec.Sources) > 0
}

// GetPrimarySources returns either the single source or the first source from multiple sources
func (app *ArgoApplication) GetPrimarySource() *struct {
	RepoURL        string `json:"repoURL,omitempty"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
} {
	if app.Spec.Source != nil {
		return app.Spec.Source
	}
	if len(app.Spec.Sources) > 0 {
		return &app.Spec.Sources[0]
	}
	return nil
}

// ResourceNode represents a Kubernetes resource from ArgoCD API
type ResourceNode struct {
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	Namespace      *string         `json:"namespace,omitempty"`
	Version        string          `json:"version"`
	Group          string          `json:"group"`
	UID            string          `json:"uid"`
	Health         *ResourceHealth `json:"health,omitempty"`
	Status         string          `json:"status"`
	NetworkingInfo *NetworkingInfo `json:"networkingInfo,omitempty"`
	ResourceRef    ResourceRef     `json:"resourceRef"`
	ParentRefs     []ResourceRef   `json:"parentRefs,omitempty"`
	Info           []ResourceInfo  `json:"info,omitempty"`
	CreatedAt      *time.Time      `json:"createdAt,omitempty"`
}

// ResourceHealth represents the health status from ArgoCD API
type ResourceHealth struct {
	Status  *string `json:"status,omitempty"`
	Message *string `json:"message,omitempty"`
}

// NetworkingInfo represents networking information from ArgoCD API
type NetworkingInfo struct {
	TargetLabels map[string]string `json:"targetLabels,omitempty"`
	TargetRefs   []ResourceRef     `json:"targetRefs,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Ingress      []IngressInfo     `json:"ingress,omitempty"`
}

// IngressInfo represents ingress information from ArgoCD API
type IngressInfo struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// ResourceRef represents a reference to a Kubernetes resource from ArgoCD API
type ResourceRef struct {
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
	Group     string  `json:"group"`
	Version   string  `json:"version"`
	UID       string  `json:"uid"`
}

// ResourceInfo represents additional information about a resource from ArgoCD API
type ResourceInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ResourceTree represents the resource tree response from ArgoCD API
type ResourceTree struct {
	Nodes []ResourceNode `json:"nodes"`
}

// GetResourceTree retrieves the resource tree for an application
func (s *ApplicationService) GetResourceTree(ctx context.Context, appName, appNamespace string) (*ResourceTree, error) {
	path := fmt.Sprintf("/api/v1/applications/%s/resource-tree", url.PathEscape(appName))
	if appNamespace != "" {
		path += "?appNamespace=" + url.QueryEscape(appNamespace)
	}

	resp, err := s.client.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource tree for application %s: %w", appName, err)
	}

	var tree ResourceTree
	if err := json.Unmarshal(resp, &tree); err != nil {
		return nil, fmt.Errorf("failed to decode resource tree response: %w", err)
	}

	return &tree, nil
}

// ResourceTreeStreamResult wraps streaming responses for resource tree
type ResourceTreeStreamResult struct {
	Result ResourceTree `json:"result"`
}

// WatchResourceTree starts a streaming watch for an application's resource tree
func (s *ApplicationService) WatchResourceTree(ctx context.Context, appName, appNamespace string, out chan<- ResourceTree) error {
	if appName == "" {
		return fmt.Errorf("application name is required")
	}
	path := fmt.Sprintf("/api/v1/stream/applications/%s/resource-tree", url.PathEscape(appName))
	if appNamespace != "" {
		path += "?appNamespace=" + url.QueryEscape(appNamespace)
	}
	cblog.With("component", "api").Debug("Starting resource tree watch", "app", appName, "path", path)
	stream, err := s.client.Stream(ctx, path)
	if err != nil {
		cblog.With("component", "api").Error("Failed to start resource tree watch", "err", err, "app", appName)
		return fmt.Errorf("failed to start resource tree watch: %w", err)
	}
	defer stream.Close()
	cblog.With("component", "api").Debug("Resource tree stream established", "app", appName)

	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	eventCount := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			cblog.With("component", "api").Debug("Context cancelled, stopping tree watch", "app", appName)
			return ctx.Err()
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// SSE format: lines starting with "data: " contain the JSON payload
		// Lines with just ":" are keep-alive messages
		if line == ":" {
			continue // Keep-alive message
		}

		if !strings.HasPrefix(line, "data: ") {
			cblog.With("component", "api").Debug("Skipping non-data SSE line", "line", line)
			continue
		}

		// Strip the "data: " prefix to get the JSON
		jsonData := strings.TrimPrefix(line, "data: ")
		cblog.With("component", "api").Debug("Received tree stream event", "app", appName, "data", jsonData)

		var res ResourceTreeStreamResult
		if err := json.Unmarshal([]byte(jsonData), &res); err != nil {
			cblog.With("component", "api").Warn("Failed to parse tree stream event", "err", err, "data", jsonData)
			continue
		}
		eventCount++
		cblog.With("component", "api").Debug("Sending tree update", "app", appName, "event", eventCount)
		select {
		case out <- res.Result:
			cblog.With("component", "api").Debug("Tree update sent", "app", appName, "event", eventCount)
		case <-ctx.Done():
			cblog.With("component", "api").Debug("Context done while sending, stopping", "app", appName)
			return ctx.Err()
		}
	}
	if err := scanner.Err(); err != nil {
		cblog.With("component", "api").Error("Stream scanning error", "err", err, "app", appName)
		return fmt.Errorf("stream scanning error: %w", err)
	}
	cblog.With("component", "api").Info("Tree watch stream ended", "app", appName, "events", eventCount)
	return nil
}

// GetUserInfo validates user authentication by checking session info
func (s *ApplicationService) GetUserInfo(ctx context.Context) error {
	resp, err := s.client.Get(ctx, "/api/v1/session/userinfo")
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// We don't need to parse the response, just verify it's successful
	// The existence of a successful response indicates the user is authenticated
	_ = resp // Acknowledge we received the response

	return nil
}

// GetApplication fetches a single application with full details including history
func (s *ApplicationService) GetApplication(ctx context.Context, name string, appNamespace *string) (*ArgoApplication, error) {
	endpoint := fmt.Sprintf("/api/v1/applications/%s", name)
	if appNamespace != nil && *appNamespace != "" {
		endpoint += "?appNamespace=" + url.QueryEscape(*appNamespace)
	}

	resp, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get application %s: %w", name, err)
	}

	var app ArgoApplication
	if err := json.Unmarshal(resp, &app); err != nil {
		return nil, fmt.Errorf("failed to decode application response: %w", err)
	}

	return &app, nil
}

// GetRevisionMetadata fetches git metadata for a specific revision
func (s *ApplicationService) GetRevisionMetadata(ctx context.Context, name string, revision string, appNamespace *string) (*model.RevisionMetadata, error) {
	endpoint := fmt.Sprintf("/api/v1/applications/%s/revisions/%s/metadata", name, revision)
	if appNamespace != nil && *appNamespace != "" {
		endpoint += "?appNamespace=" + url.QueryEscape(*appNamespace)
	}

	resp, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get revision metadata for %s@%s: %w", name, revision, err)
	}

	var metadata RevisionMetadataResponse
	if err := json.Unmarshal(resp, &metadata); err != nil {
		return nil, fmt.Errorf("failed to decode revision metadata response: %w", err)
	}

	return &model.RevisionMetadata{
		Author:  metadata.Author,
		Date:    metadata.Date,
		Message: metadata.Message,
		Tags:    metadata.Tags,
	}, nil
}

// RollbackApplication performs a rollback operation
func (s *ApplicationService) RollbackApplication(ctx context.Context, request model.RollbackRequest) error {
	endpoint := fmt.Sprintf("/api/v1/applications/%s/rollback", request.Name)
	if request.AppNamespace != nil && *request.AppNamespace != "" {
		endpoint += "?appNamespace=" + url.QueryEscape(*request.AppNamespace)
	}

	body := map[string]interface{}{
		"id":   request.ID,
		"name": request.Name,
	}

	if request.DryRun {
		body["dryRun"] = true
	}
	if request.Prune {
		body["prune"] = true
	}
	if request.AppNamespace != nil {
		body["appNamespace"] = *request.AppNamespace
	}

	// Pass the structured body directly; the client marshals it to JSON.
	_, err := s.client.Post(ctx, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to rollback application %s to deployment %d: %w", request.Name, request.ID, err)
	}

	return nil
}

// ConvertDeploymentHistoryToRollbackRows converts ArgoCD deployment history to rollback rows
func ConvertDeploymentHistoryToRollbackRows(history []DeploymentHistory) []model.RollbackRow {
	rows := make([]model.RollbackRow, 0, len(history))

	for _, deployment := range history {
		row := model.RollbackRow{
			ID:         deployment.ID,
			Revision:   deployment.Revision,
			DeployedAt: &deployment.DeployedAt,
			Author:     nil, // Will be loaded asynchronously
			Date:       nil, // Will be loaded asynchronously
			Message:    nil, // Will be loaded asynchronously
			MetaError:  nil,
		}
		rows = append(rows, row)
	}

	return rows
}
