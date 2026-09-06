package api

import "testing"

func appWithOperation(op Operation) ArgoApplication {
	return ArgoApplication{
		Metadata: ApplicationMetadata{Name: "demo-app"},
		Status: ApplicationStatus{
			OperationState: OperationState{Phase: "Succeeded", Operation: op},
		},
	}
}

func TestConvertOperationState_PlainSync_IsNotQualified(t *testing.T) {
	details := ConvertOperationState(appWithOperation(Operation{Sync: &SyncOperation{Revision: "HEAD"}}))

	if details.Operation != "Sync" {
		t.Errorf("expected operation %q, got %q", "Sync", details.Operation)
	}
	if details.OperationNote != "" {
		t.Errorf("expected no qualifier on a plain sync, got %q", details.OperationNote)
	}
}

func TestConvertOperationState_DryRun_SaysSoSoASucceededPhaseIsNotMisread(t *testing.T) {
	details := ConvertOperationState(appWithOperation(Operation{Sync: &SyncOperation{DryRun: true}}))

	if details.OperationNote != "dry run" {
		t.Errorf("expected the operation marked as a dry run, got %q", details.OperationNote)
	}
}

func TestConvertOperationState_ResourceSubset_IsMarkedPartial(t *testing.T) {
	op := Operation{Sync: &SyncOperation{Resources: []SyncResourceTarget{{Kind: "Deployment", Name: "api"}}}}

	details := ConvertOperationState(appWithOperation(op))

	if details.OperationNote != "partial" {
		t.Errorf("expected a resource-scoped sync marked partial, got %q", details.OperationNote)
	}
}

func TestConvertOperationState_DryRunOfASubset_CarriesBothQualifiers(t *testing.T) {
	op := Operation{Sync: &SyncOperation{
		DryRun:    true,
		Resources: []SyncResourceTarget{{Kind: "Deployment", Name: "api"}},
	}}

	details := ConvertOperationState(appWithOperation(op))

	if details.OperationNote != "dry run, partial" {
		t.Errorf("expected both qualifiers, got %q", details.OperationNote)
	}
}

func TestConvertOperationState_AppBeingDeleted_ReportsDeletionNotTheOldSync(t *testing.T) {
	app := appWithOperation(Operation{Sync: &SyncOperation{Revision: "HEAD"}})
	app.Metadata.DeletionTimestamp = "2026-08-18T10:00:00Z"

	details := ConvertOperationState(app)

	if details.Operation != "Deleting" {
		t.Errorf("expected a deleting app to report Deleting, got %q", details.Operation)
	}
}
