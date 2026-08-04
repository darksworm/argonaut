package api

import (
	"reflect"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestConvertToApp_WithOperationState_BuildsSyncOpSummary(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			OperationState: OperationState{
				Phase:      "Failed",
				StartedAt:  time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
				FinishedAt: time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
				Operation: Operation{
					Sync:        &SyncOperation{Revision: "HEAD"},
					InitiatedBy: OperationInitiator{Username: "alice"},
				},
				SyncResult: &SyncOperationResult{Revision: "a1b2c3d"},
			},
		},
	}

	app := svc.ConvertToApp(argoApp)

	want := &model.SyncOpSummary{
		Phase:       "Failed",
		StartedAt:   time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
		Revision:    "a1b2c3d",
		InitiatedBy: "alice",
		Automated:   false,
	}
	if !reflect.DeepEqual(app.SyncOp, want) {
		t.Errorf("expected SyncOp %+v, got %+v", want, app.SyncOp)
	}
}

func TestConvertToApp_NeverSynced_HasNoSyncOpSummary(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{Name: "fresh-app"},
	}

	app := svc.ConvertToApp(argoApp)

	if app.SyncOp != nil {
		t.Errorf("expected no SyncOp for a never-synced app, got %+v", app.SyncOp)
	}
}

func TestConvertToApp_RunningSync_FallsBackToRequestedRevision(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			OperationState: OperationState{
				Phase:     "Running",
				StartedAt: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
				Operation: Operation{
					Sync:        &SyncOperation{Revision: "main"},
					InitiatedBy: OperationInitiator{Automated: true},
				},
			},
		},
	}

	app := svc.ConvertToApp(argoApp)

	if app.SyncOp == nil {
		t.Fatal("expected a SyncOp for a running operation")
	}
	if app.SyncOp.Revision != "main" {
		t.Errorf("expected revision to fall back to the requested revision 'main', got %q", app.SyncOp.Revision)
	}
	if !app.SyncOp.Automated {
		t.Error("expected Automated to be true for an automated sync")
	}
	if !app.SyncOp.FinishedAt.IsZero() {
		t.Errorf("expected zero FinishedAt while running, got %v", app.SyncOp.FinishedAt)
	}
}

func TestConvertOperationState_FailedSync_CarriesResourceResults(t *testing.T) {
	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			OperationState: OperationState{
				Phase:      "Failed",
				Message:    "one or more objects failed to apply",
				StartedAt:  time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
				FinishedAt: time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
				Operation: Operation{
					Sync:        &SyncOperation{Revision: "HEAD"},
					InitiatedBy: OperationInitiator{Username: "alice"},
				},
				SyncResult: &SyncOperationResult{
					Revision: "a1b2c3d",
					Resources: []ResourceResult{
						{
							Kind:      "Service",
							Namespace: "demo",
							Name:      "web",
							Status:    "Synced",
							Message:   "service/web unchanged",
						},
						{
							Kind:      "Deployment",
							Namespace: "demo",
							Name:      "web",
							Status:    "SyncFailed",
							Message:   "Deployment.apps \"web\" is invalid",
						},
					},
				},
			},
		},
	}

	details := ConvertOperationState(argoApp)

	want := &model.SyncStatusDetails{
		Phase:       "Failed",
		Message:     "one or more objects failed to apply",
		StartedAt:   time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
		Revision:    "a1b2c3d",
		InitiatedBy: "alice",
		Automated:   false,
		Resources: []model.SyncResourceResult{
			{Kind: "Service", Namespace: "demo", Name: "web", Status: "Synced", Message: "service/web unchanged"},
			{Kind: "Deployment", Namespace: "demo", Name: "web", Status: "SyncFailed", Message: "Deployment.apps \"web\" is invalid"},
		},
	}
	if !reflect.DeepEqual(details, want) {
		t.Errorf("expected %+v, got %+v", want, details)
	}
}

func TestConvertOperationState_HookResource_ShowsHookPhase(t *testing.T) {
	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			OperationState: OperationState{
				Phase: "Succeeded",
				SyncResult: &SyncOperationResult{
					Resources: []ResourceResult{{
						Kind:      "Job",
						Namespace: "demo",
						Name:      "db-migrate",
						HookType:  "PreSync",
						HookPhase: "Succeeded",
					}},
				},
			},
		},
	}

	details := ConvertOperationState(argoApp)

	if details.Resources[0].Status != "Succeeded" {
		t.Errorf("expected hook resource to show its hook phase 'Succeeded', got %q", details.Resources[0].Status)
	}
}

func TestConvertToApp_WithApplicationSet(t *testing.T) {
	svc := &ApplicationService{}

	argoApp := ArgoApplication{
		Metadata: ApplicationMetadata{
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
		Status: ApplicationStatus{
			Sync:   SyncStatus{Status: "Synced"},
			Health: HealthStatus{Status: "Healthy"},
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
		Metadata: ApplicationMetadata{
			Name:            "standalone-app",
			Namespace:       "argocd",
			OwnerReferences: nil,
		},
		Status: ApplicationStatus{
			Sync:   SyncStatus{Status: "Synced"},
			Health: HealthStatus{Status: "Healthy"},
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
		Metadata: ApplicationMetadata{
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
		Status: ApplicationStatus{
			Sync:   SyncStatus{Status: "Synced"},
			Health: HealthStatus{Status: "Healthy"},
		},
	}

	app := svc.ConvertToApp(argoApp)

	if app.ApplicationSet != nil {
		t.Errorf("Expected ApplicationSet to be nil for app with non-ApplicationSet owner, got %v", *app.ApplicationSet)
	}
}
