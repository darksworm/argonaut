package api

import (
	"testing"
	"time"
)

func TestConvertToApp_WithApplicationSet(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: struct {
			Name            string           `json:"name"`
			Namespace       string           `json:"namespace,omitempty"`
			OwnerReferences []OwnerReference `json:"ownerReferences,omitempty"`
		}{
			Name:      "test-app",
			Namespace: "argocd",
			OwnerReferences: []OwnerReference{
				{
					APIVersion: "argoproj.io/v1alpha1",
					Kind:       "ApplicationSet",
					Name:       "my-appset",
					UID:        "12345",
				},
			},
		},
		Status: struct {
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
		}{
			Sync: struct {
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
			}{Status: "Synced"},
			Health: struct {
				Status  string `json:"status,omitempty"`
				Message string `json:"message,omitempty"`
			}{Status: "Healthy"},
		},
	}

	app := svc.ConvertToApp(argoApp)

	if app.Name != "test-app" {
		t.Errorf("Expected name 'test-app', got %s", app.Name)
	}

	if app.ApplicationSet == nil {
		t.Fatal("Expected ApplicationSet to be set")
	}

	if *app.ApplicationSet != "my-appset" {
		t.Errorf("Expected ApplicationSet 'my-appset', got %s", *app.ApplicationSet)
	}
}

func TestConvertToApp_WithoutApplicationSet(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: struct {
			Name            string           `json:"name"`
			Namespace       string           `json:"namespace,omitempty"`
			OwnerReferences []OwnerReference `json:"ownerReferences,omitempty"`
		}{
			Name:            "standalone-app",
			Namespace:       "argocd",
			OwnerReferences: nil,
		},
		Status: struct {
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
		}{
			Sync: struct {
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
			}{Status: "Synced"},
			Health: struct {
				Status  string `json:"status,omitempty"`
				Message string `json:"message,omitempty"`
			}{Status: "Healthy"},
		},
	}

	app := svc.ConvertToApp(argoApp)

	if app.ApplicationSet != nil {
		t.Errorf("Expected ApplicationSet to be nil for standalone app, got %v", *app.ApplicationSet)
	}
}

func TestConvertToApp_WithOtherOwnerReference(t *testing.T) {
	svc := &ApplicationService{}

	// Test that apps with non-ApplicationSet owner references don't get an ApplicationSet field
	argoApp := ArgoApplication{
		Metadata: struct {
			Name            string           `json:"name"`
			Namespace       string           `json:"namespace,omitempty"`
			OwnerReferences []OwnerReference `json:"ownerReferences,omitempty"`
		}{
			Name:      "app-with-other-owner",
			Namespace: "argocd",
			OwnerReferences: []OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "ConfigMap", // Not an ApplicationSet
					Name:       "some-configmap",
					UID:        "67890",
				},
			},
		},
		Status: struct {
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
		}{
			Sync: struct {
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
			}{Status: "Synced"},
			Health: struct {
				Status  string `json:"status,omitempty"`
				Message string `json:"message,omitempty"`
			}{Status: "Healthy"},
		},
	}

	app := svc.ConvertToApp(argoApp)

	if app.ApplicationSet != nil {
		t.Errorf("Expected ApplicationSet to be nil for app with non-ApplicationSet owner, got %v", *app.ApplicationSet)
	}
}
