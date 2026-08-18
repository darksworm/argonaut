package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// captureSyncStrategy runs a sync and returns the strategy the server received.
func captureSyncStrategy(t *testing.T, opts *SyncOptions) map[string]interface{} {
	t.Helper()

	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding sync request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})
	if err := svc.SyncApplication(context.Background(), "test-app", opts); err != nil {
		t.Fatalf("SyncApplication: %v", err)
	}

	strategy, ok := body["strategy"].(map[string]interface{})
	if !ok {
		return nil
	}
	return strategy
}

func TestSyncApplication_Force_KeepsHookStrategySoSyncHooksStillRun(t *testing.T) {
	strategy := captureSyncStrategy(t, &SyncOptions{Force: true})

	if strategy == nil {
		t.Fatal("expected a strategy for a forced sync, got none")
	}
	if _, applyOnly := strategy["apply"]; applyOnly {
		t.Error("forced sync sent strategy.apply, which skips sync hooks")
	}
	hook, ok := strategy["hook"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected strategy.hook, got %v", strategy)
	}
	if hook["force"] != true {
		t.Errorf("expected hook.force=true, got %v", hook["force"])
	}
}

func TestSyncApplication_WithoutForce_SendsNoStrategy(t *testing.T) {
	if strategy := captureSyncStrategy(t, &SyncOptions{}); strategy != nil {
		t.Errorf("expected no strategy when force is off, got %v", strategy)
	}
}

func TestSyncApplication_DryRun_IsSentSoArgoCDAppliesNothing(t *testing.T) {
	var body map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding sync request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})
	if err := svc.SyncApplication(context.Background(), "test-app", &SyncOptions{DryRun: true}); err != nil {
		t.Fatalf("SyncApplication: %v", err)
	}

	if body["dryRun"] != true {
		t.Errorf("expected dryRun=true in the request body, got %v", body)
	}
}
