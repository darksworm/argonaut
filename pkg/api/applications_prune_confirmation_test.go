package api

import "testing"

func runningSyncOf(resources []ResourceStatus) ArgoApplication {
	return ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			Resources: resources,
			OperationState: OperationState{
				Phase:     "Running",
				Operation: Operation{Sync: &SyncOperation{Prune: true}},
			},
		},
	}
}

func TestConvertOperationState_SyncWaitingOnPruneConfirmation_SaysWhyItIsStuck(t *testing.T) {
	app := runningSyncOf([]ResourceStatus{
		{Kind: "ConfigMap", Name: "settings"},
		{Kind: "Namespace", Name: "legacy", RequiresDeletionConfirmation: true},
	})

	details := ConvertOperationState(app)

	if !details.AwaitingPruneConfirmation {
		t.Error("expected a sync held on prune confirmation to be reported as such")
	}
}

func TestConvertOperationState_RunningSyncWithNothingToConfirm_IsJustRunning(t *testing.T) {
	app := runningSyncOf([]ResourceStatus{{Kind: "ConfigMap", Name: "settings"}})

	details := ConvertOperationState(app)

	if details.AwaitingPruneConfirmation {
		t.Error("expected no confirmation notice when no resource requires one")
	}
}

func TestConvertOperationState_NonSyncOperationIsNotWaitingOnPruneConfirmation(t *testing.T) {
	app := runningSyncOf([]ResourceStatus{{Kind: "Namespace", Name: "legacy", RequiresDeletionConfirmation: true}})
	app.Status.OperationState.Operation.Sync = nil

	details := ConvertOperationState(app)

	if details.AwaitingPruneConfirmation {
		t.Error("expected a non-sync operation not to report waiting on prune confirmation")
	}
}

func TestConvertOperationState_SyncWithoutPruneIsNotWaitingOnPruneConfirmation(t *testing.T) {
	app := runningSyncOf([]ResourceStatus{{Kind: "Namespace", Name: "legacy", RequiresDeletionConfirmation: true}})
	app.Status.OperationState.Operation.Sync.Prune = false

	details := ConvertOperationState(app)

	if details.AwaitingPruneConfirmation {
		t.Error("expected a sync with pruning disabled not to report waiting on prune confirmation")
	}
}

func TestConvertOperationState_FinishedSync_IsNotWaitingOnAnything(t *testing.T) {
	app := runningSyncOf([]ResourceStatus{{Kind: "Namespace", Name: "legacy", RequiresDeletionConfirmation: true}})
	app.Status.OperationState.Phase = "Succeeded"

	details := ConvertOperationState(app)

	if details.AwaitingPruneConfirmation {
		t.Error("expected a finished sync not to report waiting on confirmation")
	}
}
