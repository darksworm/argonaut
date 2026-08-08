package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestConvertDeploymentHistory_CarriesDeployMetadataAndSource(t *testing.T) {
	historyJSON := `[{
		"id": 30,
		"revision": "a1b2c3d4e5",
		"deployedAt": "2026-08-07T14:02:00Z",
		"deployStartedAt": "2026-08-07T14:01:46Z",
		"initiatedBy": {"username": "jane.doe"},
		"source": {"repoURL": "https://github.com/corp/example-apps", "path": "apps/demo", "targetRevision": "main"}
	}, {
		"id": 29,
		"revision": "11223344",
		"deployedAt": "2026-08-06T10:00:00Z",
		"initiatedBy": {"automated": true}
	}]`

	var history []DeploymentHistory
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		t.Fatalf("unmarshal history: %v", err)
	}

	rows := ConvertDeploymentHistoryToRollbackRows(history)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	manual := rows[0]
	wantStarted := time.Date(2026, 8, 7, 14, 1, 46, 0, time.UTC)
	if manual.DeployStartedAt == nil || !manual.DeployStartedAt.Equal(wantStarted) {
		t.Errorf("DeployStartedAt = %v, want %v", manual.DeployStartedAt, wantStarted)
	}
	if manual.InitiatedBy != "jane.doe" || manual.Automated {
		t.Errorf("initiator = (%q, automated=%v), want (jane.doe, false)", manual.InitiatedBy, manual.Automated)
	}
	wantSource := &model.RollbackSource{
		RepoURL:        "https://github.com/corp/example-apps",
		Path:           "apps/demo",
		TargetRevision: "main",
	}
	if manual.Source == nil || *manual.Source != *wantSource {
		t.Errorf("Source = %+v, want %+v", manual.Source, wantSource)
	}

	automated := rows[1]
	if automated.DeployStartedAt != nil {
		t.Errorf("DeployStartedAt = %v, want nil when absent", automated.DeployStartedAt)
	}
	if !automated.Automated || automated.InitiatedBy != "" {
		t.Errorf("initiator = (%q, automated=%v), want (\"\", true)", automated.InitiatedBy, automated.Automated)
	}
	if automated.Source != nil {
		t.Errorf("Source = %+v, want nil when absent", automated.Source)
	}
}

func TestGetApplicationManifests_FetchesRevisionManifestsForNamespacedApp(t *testing.T) {
	var gotPath, gotRevision, gotNamespace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotRevision = r.URL.Query().Get("revision")
		gotNamespace = r.URL.Query().Get("appNamespace")
		w.Write([]byte(`{"manifests": ["{\"kind\":\"Deployment\"}", "{\"kind\":\"Service\"}"]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})
	ns := "team a"
	manifests, err := svc.GetApplicationManifests(context.Background(), "my app", "deadbeef", &ns)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotPath != "/api/v1/applications/my%20app/manifests" {
		t.Errorf("path = %s, want /api/v1/applications/my%%20app/manifests", gotPath)
	}
	if gotRevision != "deadbeef" || gotNamespace != "team a" {
		t.Errorf("query = (revision=%q, appNamespace=%q), want (deadbeef, team a)", gotRevision, gotNamespace)
	}
	want := []string{`{"kind":"Deployment"}`, `{"kind":"Service"}`}
	if !reflect.DeepEqual(manifests, want) {
		t.Errorf("manifests = %v, want %v", manifests, want)
	}
}

func TestGetRevisionMetadata_EscapesNameAndRevision(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte(`{"author": "jane", "date": "2026-08-07T13:58:00Z", "message": "fix"}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})
	_, err := svc.GetRevisionMetadata(context.Background(), "my app", "release/v1", nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "/api/v1/applications/my%20app/revisions/release%2Fv1/metadata"
	if gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
}

func TestDisableAutoSync_SendsMergePatchNullingAutomatedPolicy(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})
	ns := "team-a"
	if err := svc.DisableAutoSync(context.Background(), "my app", &ns); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if gotMethod != "PATCH" || gotPath != "/api/v1/applications/my%20app" {
		t.Errorf("request = %s %s, want PATCH /api/v1/applications/my%%20app", gotMethod, gotPath)
	}
	if gotBody["name"] != "my app" || gotBody["patchType"] != "merge" || gotBody["appNamespace"] != "team-a" {
		t.Errorf("body = %v, want name/patchType/appNamespace set", gotBody)
	}
	patch, _ := gotBody["patch"].(string)
	var patchDoc map[string]map[string]any
	if err := json.Unmarshal([]byte(patch), &patchDoc); err != nil {
		t.Fatalf("patch is not valid JSON: %v (%q)", err, patch)
	}
	syncPolicy, ok := patchDoc["spec"]["syncPolicy"].(map[string]any)
	if !ok {
		t.Fatalf("patch = %q, want spec.syncPolicy object", patch)
	}
	if automated, present := syncPolicy["automated"]; !present || automated != nil {
		t.Errorf("patch spec.syncPolicy.automated = %v (present=%v), want explicit null", automated, present)
	}
}
