package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/a9s/go-app/pkg/model"
)

// ArgoApplication represents an ArgoCD application from the API
type ArgoApplication struct {
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace,omitempty"`
	} `json:"metadata"`
	Spec struct {
		Project string `json:"project,omitempty"`
		Source  struct {
			RepoURL        string `json:"repoURL,omitempty"`
			Path           string `json:"path,omitempty"`
			TargetRevision string `json:"targetRevision,omitempty"`
		} `json:"source"`
		Destination struct {
			Name      string `json:"name,omitempty"`
			Server    string `json:"server,omitempty"`
			Namespace string `json:"namespace,omitempty"`
		} `json:"destination"`
	} `json:"spec"`
	Status struct {
		Sync struct {
			Status     string    `json:"status,omitempty"`
			ComparedTo struct {
				Source struct {
					RepoURL        string `json:"repoURL,omitempty"`
					Path           string `json:"path,omitempty"`
					TargetRevision string `json:"targetRevision,omitempty"`
				} `json:"source"`
			} `json:"comparedTo"`
			Revision string `json:"revision,omitempty"`
		} `json:"sync"`
		Health struct {
			Status  string `json:"status,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"health"`
		OperationState struct {
			Phase     string    `json:"phase,omitempty"`
			StartedAt time.Time `json:"startedAt,omitempty"`
			FinishedAt time.Time `json:"finishedAt,omitempty"`
		} `json:"operationState,omitempty"`
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
    var withItems struct{
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
            if app.Sync == "" { app.Sync = "Unknown" }
            if app.Health == "" { app.Health = "Unknown" }
        }

        apps = append(apps, app)
    }

    return apps, nil
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
	stream, err := s.client.Stream(ctx, "/api/v1/stream/applications")
	if err != nil {
		return fmt.Errorf("failed to start watch stream: %w", err)
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var eventResult WatchEventResult
		if err := json.Unmarshal([]byte(line), &eventResult); err != nil {
			// Skip malformed lines
			continue
		}

		select {
		case eventChan <- eventResult.Result:
		case <-ctx.Done():
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
