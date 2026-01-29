package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// OwnerReference represents a Kubernetes owner reference
type OwnerReference struct {
	APIVersion string `json:"apiVersion,omitempty"`
	Kind       string `json:"kind,omitempty"`
	Name       string `json:"name,omitempty"`
	UID        string `json:"uid,omitempty"`
}

// ArgoApplication represents an ArgoCD application from the API
type ArgoApplication struct {
	Metadata struct {
		Name            string           `json:"name"`
		Namespace       string           `json:"namespace,omitempty"`
		OwnerReferences []OwnerReference `json:"ownerReferences,omitempty"`
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
		History   []DeploymentHistory `json:"history,omitempty"`
		Resources []ResourceStatus    `json:"resources,omitempty"`
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

	// Add resources array if provided (for selective resource sync)
	if len(opts.Resources) > 0 {
		reqBody["resources"] = opts.Resources
	}

	// Add force option via strategy if enabled
	if opts.Force {
		reqBody["strategy"] = map[string]interface{}{
			"apply": map[string]interface{}{
				"force": true,
			},
		}
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
	streamResp, err := s.client.Stream(ctx, "/api/v1/stream/applications")
	if err != nil {
		cblog.With("component", "api").Error("WatchApplications: failed to establish stream", "error", err)
		return fmt.Errorf("failed to start watch stream: %w", err)
	}
	cblog.With("component", "api").Info("WatchApplications: stream established successfully")
	
	// Create AccumulatingSSEReader for reliable event processing
	sseConfig := DefaultSSEConfig()
	sseReader := NewAccumulatingSSEReader(streamResp.Body, sseConfig)
	defer sseReader.Close()

	cblog.With("component", "api").Info("WatchApplications: configured AccumulatingSSEReader", 
		"initialBuffer", sseConfig.InitialBuffer, 
		"maxBuffer", sseConfig.MaxBuffer,
		"maxAccumulated", sseConfig.MaxAccumulated)
	
	cblog.With("component", "api").Info("WatchApplications: starting to read from stream")
	
	for {
		if ctx.Err() != nil {
			cblog.With("component", "api").Debug("WatchApplications: context cancelled")
			return ctx.Err()
		}

		// Read next SSE event
		eventData, err := sseReader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				// Stream ended normally
				cblog.With("component", "api").Debug("WatchApplications: stream ended normally")
				return nil
			}
			
			// Check for event too large error
			if errors.Is(err, ErrEventTooLarge) {
				metrics := sseReader.Metrics()
				cblog.With("component", "api").Error("WatchApplications: SSE event too large",
					"maxEventSize", metrics.MaxEventSize,
					"bufferResizes", metrics.BufferResizes,
					"error", err)
				return fmt.Errorf("SSE event exceeds maximum size: %w", err)
			}
			
			return fmt.Errorf("error reading SSE event: %w", err)
		}
		
		if len(eventData) == 0 {
			continue
		}
		
		// Process SSE event lines
		lines := strings.Split(string(eventData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == ":" {
				// Skip empty lines and keep-alive messages
				continue
			}

			cblog.With("component", "api").Debug("WatchApplications: processing line from event", "line", line)

			// Handle Server-Sent Events format (lines starting with "data: ")
			if strings.HasPrefix(line, "data: ") {
				dataLine := strings.TrimPrefix(line, "data: ")
				
				var eventResult WatchEventResult
				if err := json.Unmarshal([]byte(dataLine), &eventResult); err != nil {
					cblog.With("component", "api").Warn("WatchApplications: failed to unmarshal event", "error", err, "line", dataLine)
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
			} else if !strings.HasPrefix(line, ":") {
				// Skip comment lines (starting with ":" ) but log unexpected lines
				cblog.With("component", "api").Debug("WatchApplications: skipping non-data line", "line", line)
			}
		}
	}
}

// SyncOptions represents options for syncing an application
// SyncResourceTarget represents a specific resource to sync
type SyncResourceTarget struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type SyncOptions struct {
	Prune        bool                 `json:"prune,omitempty"`
	DryRun       bool                 `json:"dryRun,omitempty"`
	Force        bool                 `json:"force,omitempty"`
	AppNamespace string               `json:"appNamespace,omitempty"`
	Resources    []SyncResourceTarget `json:"resources,omitempty"`
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

	// Extract ApplicationSet from ownerReferences
	for _, ref := range argoApp.Metadata.OwnerReferences {
		if ref.Kind == "ApplicationSet" {
			app.ApplicationSet = &ref.Name
			break
		}
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

// ResourceStatus holds sync/health status for a managed resource (from Application.status.resources[])
type ResourceStatus struct {
	Group     string          `json:"group"`
	Kind      string          `json:"kind"`
	Name      string          `json:"name"`
	Namespace string          `json:"namespace,omitempty"`
	Status    string          `json:"status"` // Sync status: "Synced", "OutOfSync"
	Version   string          `json:"version"`
	Health    *ResourceHealth `json:"health,omitempty"`
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
	streamResp, err := s.client.Stream(ctx, path)
	if err != nil {
		cblog.With("component", "api").Error("Failed to start resource tree watch", "err", err, "app", appName)
		return fmt.Errorf("failed to start resource tree watch: %w", err)
	}
	
	// Create AccumulatingSSEReader for reliable event processing
	sseConfig := DefaultSSEConfig()
	sseReader := NewAccumulatingSSEReader(streamResp.Body, sseConfig)
	defer sseReader.Close()
	cblog.With("component", "api").Debug("Resource tree stream established", "app", appName)

	cblog.With("component", "api").Debug("WatchResourceTree: configured AccumulatingSSEReader",
		"app", appName,
		"initialBuffer", sseConfig.InitialBuffer,
		"maxBuffer", sseConfig.MaxBuffer,
		"maxAccumulated", sseConfig.MaxAccumulated)
	
	eventCount := 0
	
	for {
		if ctx.Err() != nil {
			cblog.With("component", "api").Debug("Context cancelled, stopping tree watch", "app", appName)
			return ctx.Err()
		}
		
		// Read next SSE event
		eventData, err := sseReader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				// Stream ended normally
				cblog.With("component", "api").Debug("WatchResourceTree: stream ended normally", "app", appName)
				return nil
			}
			
			// Check for event too large error
			if errors.Is(err, ErrEventTooLarge) {
				metrics := sseReader.Metrics()
				cblog.With("component", "api").Error("WatchResourceTree: SSE event too large",
					"app", appName,
					"maxEventSize", metrics.MaxEventSize,
					"bufferResizes", metrics.BufferResizes,
					"error", err)
				return fmt.Errorf("resource tree SSE event exceeds maximum size: %w", err)
			}
			
			return fmt.Errorf("error reading SSE event: %w", err)
		}
		
		if len(eventData) == 0 {
			continue
		}
		
		// Process SSE event lines
		lines := strings.Split(string(eventData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || line == ":" {
				// Skip empty lines and keep-alive messages
				continue
			}

			// SSE format: lines starting with "data: " contain the JSON payload
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
	}
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

// RefreshOptions specifies options for refreshing an application
type RefreshOptions struct {
	Hard         bool    // If true, performs hard refresh (invalidates cache)
	AppNamespace *string // Optional app namespace for multi-tenant clusters
}

// RefreshApplication triggers a refresh for the specified application.
// Normal refresh compares with git; hard refresh invalidates the manifest cache.
func (s *ApplicationService) RefreshApplication(ctx context.Context, name string, opts *RefreshOptions) error {
	if name == "" {
		return fmt.Errorf("application name is required")
	}

	params := url.Values{}
	if opts != nil && opts.Hard {
		params.Set("refresh", "hard")
	} else {
		params.Set("refresh", "true")
	}
	if opts != nil && opts.AppNamespace != nil && *opts.AppNamespace != "" {
		params.Set("appNamespace", *opts.AppNamespace)
	}

	endpoint := fmt.Sprintf("/api/v1/applications/%s?%s", url.PathEscape(name), params.Encode())

	_, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to refresh application %s: %w", name, err)
	}

	return nil
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

// DeleteRequest represents a request to delete an application
type DeleteRequest struct {
	AppName           string
	AppNamespace      *string
	Cascade           bool
	PropagationPolicy string
}

// DeleteApplication deletes an application from ArgoCD
func (s *ApplicationService) DeleteApplication(ctx context.Context, req DeleteRequest) error {
	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}

	// Build the endpoint path
	endpoint := fmt.Sprintf("/api/v1/applications/%s", url.PathEscape(req.AppName))

	// Build query parameters
	params := url.Values{}

	// Set cascade parameter
	params.Set("cascade", fmt.Sprintf("%t", req.Cascade))

	// Set propagation policy if specified
	if req.PropagationPolicy != "" {
		params.Set("propagationPolicy", req.PropagationPolicy)
	}

	// Add app namespace if specified
	if req.AppNamespace != nil && *req.AppNamespace != "" {
		params.Set("appNamespace", *req.AppNamespace)
	}

	// Add query parameters to endpoint if any
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	// Perform the DELETE request
	_, err := s.client.Delete(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete application %s: %w", req.AppName, err)
	}

	return nil
}

// DeleteResourceRequest represents a request to delete a resource from an application
type DeleteResourceRequest struct {
	AppName      string
	AppNamespace *string
	ResourceName string
	Kind         string
	Namespace    string
	Version      string
	Group        string
	Orphan       bool
	Force        bool
}

// DeleteResource deletes a resource from the cluster via ArgoCD
func (s *ApplicationService) DeleteResource(ctx context.Context, req DeleteResourceRequest) error {
	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}
	if req.Kind == "" || req.ResourceName == "" {
		return fmt.Errorf("resource kind and name are required")
	}

	// Build the endpoint path
	endpoint := fmt.Sprintf("/api/v1/applications/%s/resource", url.PathEscape(req.AppName))

	// Build query parameters
	params := url.Values{}
	params.Set("resourceName", req.ResourceName)
	params.Set("kind", req.Kind)
	if req.Namespace != "" {
		params.Set("namespace", req.Namespace)
	}
	if req.Version != "" {
		params.Set("version", req.Version)
	}
	if req.Group != "" {
		params.Set("group", req.Group)
	}
	if req.Orphan {
		params.Set("orphan", "true")
	}
	if req.Force {
		params.Set("force", "true")
	}
	if req.AppNamespace != nil && *req.AppNamespace != "" {
		params.Set("appNamespace", *req.AppNamespace)
	}

	endpoint += "?" + params.Encode()

	// Perform the DELETE request
	_, err := s.client.Delete(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("failed to delete resource %s/%s: %w", req.Kind, req.ResourceName, err)
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

// RunResourceAction executes a resource action via ArgoCD's resource actions API v2
// This is used for actions like promote, abort, pause, etc. on Argo Rollouts
func (s *ApplicationService) RunResourceAction(ctx context.Context, req ResourceActionRequest) error {
	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}
	if req.ResourceName == "" {
		return fmt.Errorf("resource name is required")
	}
	if req.Kind == "" {
		return fmt.Errorf("resource kind is required")
	}
	if req.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Build the endpoint path for v2 resource actions API
	endpoint := fmt.Sprintf("/api/v1/applications/%s/resource/actions/v2", url.PathEscape(req.AppName))

	// Build the request body (v2 API uses JSON body instead of query params)
	body := map[string]interface{}{
		"name":         req.AppName,
		"resourceName": req.ResourceName,
		"kind":         req.Kind,
		"action":       req.Action,
	}

	if req.Namespace != "" {
		body["namespace"] = req.Namespace
	}
	if req.Group != "" {
		body["group"] = req.Group
	}
	if req.Version != "" {
		body["version"] = req.Version
	}
	if req.AppNamespace != nil && *req.AppNamespace != "" {
		body["appNamespace"] = *req.AppNamespace
	}

	_, err := s.client.Post(ctx, endpoint, body)
	if err != nil {
		return fmt.Errorf("failed to run action %s on %s/%s: %w", req.Action, req.Kind, req.ResourceName, err)
	}

	return nil
}

// ListResourceActions retrieves available actions for a specific resource
func (s *ApplicationService) ListResourceActions(ctx context.Context, params ListResourceActionsParams) ([]string, error) {
	if params.AppName == "" {
		return nil, fmt.Errorf("application name is required")
	}
	if params.ResourceName == "" {
		return nil, fmt.Errorf("resource name is required")
	}
	if params.Kind == "" {
		return nil, fmt.Errorf("resource kind is required")
	}

	// Build the endpoint path
	endpoint := fmt.Sprintf("/api/v1/applications/%s/resource/actions", url.PathEscape(params.AppName))

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("resourceName", params.ResourceName)
	queryParams.Set("kind", params.Kind)
	if params.Namespace != "" {
		queryParams.Set("namespace", params.Namespace)
	}
	if params.Group != "" {
		queryParams.Set("group", params.Group)
	}
	if params.Version != "" {
		queryParams.Set("version", params.Version)
	}
	if params.AppNamespace != nil && *params.AppNamespace != "" {
		queryParams.Set("appNamespace", *params.AppNamespace)
	}

	endpoint += "?" + queryParams.Encode()

	resp, err := s.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to list actions for %s/%s: %w", params.Kind, params.ResourceName, err)
	}

	// Parse the response - ArgoCD returns { "actions": [...] }
	var result struct {
		Actions []struct {
			Name     string `json:"name"`
			Disabled bool   `json:"disabled"`
		} `json:"actions"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse actions response: %w", err)
	}

	// Extract enabled action names
	var actions []string
	for _, action := range result.Actions {
		if !action.Disabled {
			actions = append(actions, action.Name)
		}
	}

	return actions, nil
}

// ListResourceActionsParams contains parameters for listing resource actions
type ListResourceActionsParams struct {
	AppName      string
	AppNamespace *string
	ResourceName string
	Namespace    string
	Kind         string
	Group        string
	Version      string
}
