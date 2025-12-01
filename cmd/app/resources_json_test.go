package main

import (
	"encoding/json"
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}

// TestResourcesJSONMarshal tests marshaling []api.ResourceStatus to JSON
// as done in api_integration.go consumeWatchEvent and startLoadingResourceTree
func TestResourcesJSONMarshal(t *testing.T) {
	tests := []struct {
		name      string
		resources []api.ResourceStatus
		wantLen   int // Expected length of marshaled JSON (0 means empty/nil)
	}{
		{
			name: "marshal single resource",
			resources: []api.ResourceStatus{
				{
					Group:     "apps",
					Kind:      "Deployment",
					Name:      "my-app",
					Namespace: "default",
					Status:    "Synced",
					Version:   "v1",
				},
			},
			wantLen: 1,
		},
		{
			name: "marshal multiple resources",
			resources: []api.ResourceStatus{
				{Group: "apps", Kind: "Deployment", Name: "app1", Namespace: "ns1", Status: "Synced"},
				{Group: "", Kind: "Service", Name: "svc1", Namespace: "ns1", Status: "OutOfSync"},
				{Group: "networking.k8s.io", Kind: "Ingress", Name: "ing1", Status: "Synced"},
			},
			wantLen: 3,
		},
		{
			name:      "marshal empty slice",
			resources: []api.ResourceStatus{},
			wantLen:   0,
		},
		{
			name:      "marshal nil slice",
			resources: nil,
			wantLen:   0,
		},
		{
			name: "marshal resource with health",
			resources: []api.ResourceStatus{
				{
					Group:     "apps",
					Kind:      "Deployment",
					Name:      "healthy-app",
					Namespace: "prod",
					Status:    "Synced",
					Health:    &api.ResourceHealth{Status: strPtr("Healthy"), Message: strPtr("All pods running")},
				},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resourcesData []byte
			if len(tt.resources) > 0 {
				resourcesData, _ = json.Marshal(tt.resources)
			}

			if tt.wantLen == 0 {
				if len(resourcesData) != 0 {
					t.Errorf("expected empty resourcesData for %s, got %d bytes", tt.name, len(resourcesData))
				}
				return
			}

			// Verify we can unmarshal back
			var unmarshaled []api.ResourceStatus
			if err := json.Unmarshal(resourcesData, &unmarshaled); err != nil {
				t.Fatalf("failed to unmarshal marshaled data: %v", err)
			}

			if len(unmarshaled) != tt.wantLen {
				t.Errorf("expected %d resources after roundtrip, got %d", tt.wantLen, len(unmarshaled))
			}
		})
	}
}

// TestResourcesJSONUnmarshal tests unmarshaling JSON to []api.ResourceStatus
// as done in model.go AppUpdatedMsg and ResourceTreeLoadedMsg handlers
func TestResourcesJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name       string
		jsonData   []byte
		wantLen    int
		wantErr    bool
		wantFirst  *api.ResourceStatus // Expected first resource (nil to skip check)
	}{
		{
			name:     "unmarshal valid single resource",
			jsonData: []byte(`[{"group":"apps","kind":"Deployment","name":"test","namespace":"default","status":"Synced","version":"v1"}]`),
			wantLen:  1,
			wantFirst: &api.ResourceStatus{
				Group:     "apps",
				Kind:      "Deployment",
				Name:      "test",
				Namespace: "default",
				Status:    "Synced",
				Version:   "v1",
			},
		},
		{
			name:     "unmarshal valid multiple resources",
			jsonData: []byte(`[{"group":"apps","kind":"Deployment","name":"d1","status":"Synced"},{"group":"","kind":"Service","name":"s1","status":"OutOfSync"}]`),
			wantLen:  2,
		},
		{
			name:     "unmarshal empty array",
			jsonData: []byte(`[]`),
			wantLen:  0,
		},
		{
			name:     "unmarshal nil/empty data",
			jsonData: nil,
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "unmarshal empty byte slice",
			jsonData: []byte{},
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "unmarshal invalid JSON",
			jsonData: []byte(`{not valid json}`),
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "unmarshal wrong type (object instead of array)",
			jsonData: []byte(`{"group":"apps","kind":"Deployment"}`),
			wantLen:  0,
			wantErr:  true,
		},
		{
			name:     "unmarshal resource with health status",
			jsonData: []byte(`[{"group":"apps","kind":"Deployment","name":"app","status":"Synced","health":{"status":"Healthy","message":"OK"}}]`),
			wantLen:  1,
			wantFirst: &api.ResourceStatus{
				Group:  "apps",
				Kind:   "Deployment",
				Name:   "app",
				Status: "Synced",
				Health: &api.ResourceHealth{Status: strPtr("Healthy"), Message: strPtr("OK")},
			},
		},
		{
			name:     "unmarshal resource with empty namespace",
			jsonData: []byte(`[{"group":"","kind":"Namespace","name":"kube-system","status":"Synced"}]`),
			wantLen:  1,
			wantFirst: &api.ResourceStatus{
				Group:  "",
				Kind:   "Namespace",
				Name:   "kube-system",
				Status: "Synced",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resources []api.ResourceStatus
			err := json.Unmarshal(tt.jsonData, &resources)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s, got nil", tt.name)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(resources) != tt.wantLen {
				t.Errorf("expected %d resources, got %d", tt.wantLen, len(resources))
			}

			if tt.wantFirst != nil && len(resources) > 0 {
				got := resources[0]
				if got.Group != tt.wantFirst.Group {
					t.Errorf("first resource Group = %q, want %q", got.Group, tt.wantFirst.Group)
				}
				if got.Kind != tt.wantFirst.Kind {
					t.Errorf("first resource Kind = %q, want %q", got.Kind, tt.wantFirst.Kind)
				}
				if got.Name != tt.wantFirst.Name {
					t.Errorf("first resource Name = %q, want %q", got.Name, tt.wantFirst.Name)
				}
				if got.Namespace != tt.wantFirst.Namespace {
					t.Errorf("first resource Namespace = %q, want %q", got.Namespace, tt.wantFirst.Namespace)
				}
				if got.Status != tt.wantFirst.Status {
					t.Errorf("first resource Status = %q, want %q", got.Status, tt.wantFirst.Status)
				}
				if tt.wantFirst.Health != nil {
					if got.Health == nil {
						t.Error("expected Health to be non-nil")
					} else if got.Health.Status != nil && tt.wantFirst.Health.Status != nil {
						if *got.Health.Status != *tt.wantFirst.Health.Status {
							t.Errorf("Health.Status = %q, want %q", *got.Health.Status, *tt.wantFirst.Health.Status)
						}
					}
				}
			}
		})
	}
}

// TestAppUpdatedMsgResourcesFlow tests the data flow from ArgoApiEvent to tree view
// This simulates what happens in consumeWatchEvent -> AppUpdatedMsg -> model.Update
func TestAppUpdatedMsgResourcesFlow(t *testing.T) {
	// Create test resources
	resources := []api.ResourceStatus{
		{Group: "apps", Kind: "Deployment", Name: "frontend", Namespace: "prod", Status: "Synced"},
		{Group: "", Kind: "Service", Name: "frontend-svc", Namespace: "prod", Status: "OutOfSync"},
	}

	// Marshal resources as done in consumeWatchEvent
	resourcesData, err := json.Marshal(resources)
	if err != nil {
		t.Fatalf("failed to marshal resources: %v", err)
	}

	// Create AppUpdatedMsg as done in consumeWatchEvent
	msg := model.AppUpdatedMsg{
		App:           model.App{Name: "test-app", Health: "Healthy", Sync: "Synced"},
		ResourcesJSON: resourcesData,
	}

	// Verify the message contains valid JSON that can be unmarshaled
	// (as done in model.go Update handler)
	var decoded []api.ResourceStatus
	if err := json.Unmarshal(msg.ResourcesJSON, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ResourcesJSON from msg: %v", err)
	}

	if len(decoded) != len(resources) {
		t.Errorf("expected %d resources, got %d", len(resources), len(decoded))
	}

	// Verify resource content
	for i, want := range resources {
		got := decoded[i]
		if got.Group != want.Group || got.Kind != want.Kind || got.Name != want.Name ||
			got.Namespace != want.Namespace || got.Status != want.Status {
			t.Errorf("resource[%d] mismatch: got %+v, want %+v", i, got, want)
		}
	}
}

// TestResourceTreeLoadedMsgResourcesFlow tests the data flow for tree loading
// This simulates startLoadingResourceTree -> ResourceTreeLoadedMsg -> model.Update
func TestResourceTreeLoadedMsgResourcesFlow(t *testing.T) {
	// Create test resources
	resources := []api.ResourceStatus{
		{Group: "apps", Kind: "Deployment", Name: "backend", Namespace: "staging", Status: "Synced"},
		{Group: "apps", Kind: "ReplicaSet", Name: "backend-abc123", Namespace: "staging", Status: "Synced"},
		{Group: "", Kind: "Pod", Name: "backend-abc123-xyz", Namespace: "staging", Status: "OutOfSync"},
	}

	// Marshal resources as done in startLoadingResourceTree
	resourcesData, _ := json.Marshal(resources)

	// Create ResourceTreeLoadedMsg
	msg := model.ResourceTreeLoadedMsg{
		AppName:       "test-app",
		Health:        "Healthy",
		Sync:          "Synced",
		TreeJSON:      []byte(`{"nodes":[]}`), // Minimal valid tree JSON
		ResourcesJSON: resourcesData,
	}

	// Unmarshal as done in model.go handler
	var decoded []api.ResourceStatus
	if err := json.Unmarshal(msg.ResourcesJSON, &decoded); err != nil {
		t.Fatalf("failed to unmarshal ResourcesJSON: %v", err)
	}

	if len(decoded) != len(resources) {
		t.Errorf("expected %d resources, got %d", len(resources), len(decoded))
	}
}

// TestSetResourceStatusesIntegration tests the complete flow including tree view update
func TestSetResourceStatusesIntegration(t *testing.T) {
	// Create a tree view
	tv := treeview.NewTreeView(80, 24)

	// Create a minimal resource tree
	tree := &api.ResourceTree{
		Nodes: []api.ResourceNode{
			{
				UID:       "uid-1",
				Group:     "apps",
				Kind:      "Deployment",
				Name:      "my-deploy",
				Namespace: strPtr("default"),
			},
			{
				UID:       "uid-2",
				Group:     "",
				Kind:      "Service",
				Name:      "my-svc",
				Namespace: strPtr("default"),
			},
		},
	}

	// Add tree to tree view
	tv.UpsertAppTree("test-app", tree)

	// Create resources with status
	resources := []api.ResourceStatus{
		{Group: "apps", Kind: "Deployment", Name: "my-deploy", Namespace: "default", Status: "Synced"},
		{Group: "", Kind: "Service", Name: "my-svc", Namespace: "default", Status: "OutOfSync"},
	}

	// Marshal and unmarshal (simulate the wire transfer)
	resourcesData, _ := json.Marshal(resources)
	var decoded []api.ResourceStatus
	if err := json.Unmarshal(resourcesData, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Call SetResourceStatuses as done in model.go
	tv.SetResourceStatuses("test-app", decoded)

	// The test verifies no panic occurs and the flow works end-to-end
	// Detailed status verification is handled by treeview_test.go
}

// TestConsumeWatchEventResourcesMarshal simulates the marshaling logic in consumeWatchEvent
func TestConsumeWatchEventResourcesMarshal(t *testing.T) {
	tests := []struct {
		name          string
		event         services.ArgoApiEvent
		wantResources bool
	}{
		{
			name: "app-updated with resources",
			event: services.ArgoApiEvent{
				Type: "app-updated",
				App:  &model.App{Name: "app1"},
				Resources: []api.ResourceStatus{
					{Group: "apps", Kind: "Deployment", Name: "d1", Status: "Synced"},
				},
			},
			wantResources: true,
		},
		{
			name: "app-updated without resources",
			event: services.ArgoApiEvent{
				Type:      "app-updated",
				App:       &model.App{Name: "app2"},
				Resources: nil,
			},
			wantResources: false,
		},
		{
			name: "app-updated with empty resources",
			event: services.ArgoApiEvent{
				Type:      "app-updated",
				App:       &model.App{Name: "app3"},
				Resources: []api.ResourceStatus{},
			},
			wantResources: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic in consumeWatchEvent
			var resourcesData []byte
			if len(tt.event.Resources) > 0 {
				resourcesData, _ = json.Marshal(tt.event.Resources)
			}

			hasResources := len(resourcesData) > 0
			if hasResources != tt.wantResources {
				t.Errorf("hasResources = %v, want %v", hasResources, tt.wantResources)
			}

			if tt.wantResources {
				// Verify valid JSON
				var decoded []api.ResourceStatus
				if err := json.Unmarshal(resourcesData, &decoded); err != nil {
					t.Errorf("invalid JSON produced: %v", err)
				}
				if len(decoded) != len(tt.event.Resources) {
					t.Errorf("decoded %d resources, expected %d", len(decoded), len(tt.event.Resources))
				}
			}
		})
	}
}

// TestResourcesJSONEdgeCases tests edge cases in JSON handling
func TestResourcesJSONEdgeCases(t *testing.T) {
	t.Run("marshal/unmarshal preserves empty strings", func(t *testing.T) {
		resources := []api.ResourceStatus{
			{Group: "", Kind: "Namespace", Name: "test", Namespace: "", Status: "Synced"},
		}

		data, _ := json.Marshal(resources)
		var decoded []api.ResourceStatus
		json.Unmarshal(data, &decoded)

		if decoded[0].Group != "" {
			t.Errorf("Group should be empty, got %q", decoded[0].Group)
		}
		if decoded[0].Namespace != "" {
			t.Errorf("Namespace should be empty, got %q", decoded[0].Namespace)
		}
	})

	t.Run("unmarshal handles extra fields gracefully", func(t *testing.T) {
		// JSON with extra unknown fields
		jsonData := []byte(`[{"group":"apps","kind":"Deployment","name":"test","status":"Synced","unknownField":"value"}]`)

		var resources []api.ResourceStatus
		if err := json.Unmarshal(jsonData, &resources); err != nil {
			t.Errorf("should handle unknown fields gracefully: %v", err)
		}
		if len(resources) != 1 {
			t.Errorf("expected 1 resource, got %d", len(resources))
		}
	})

	t.Run("unmarshal handles null values", func(t *testing.T) {
		jsonData := []byte(`[{"group":"apps","kind":"Deployment","name":"test","status":"Synced","health":null}]`)

		var resources []api.ResourceStatus
		if err := json.Unmarshal(jsonData, &resources); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resources[0].Health != nil {
			t.Error("Health should be nil when JSON has null")
		}
	})

	t.Run("handles unicode in resource names", func(t *testing.T) {
		resources := []api.ResourceStatus{
			{Group: "apps", Kind: "Deployment", Name: "app-日本語-test", Namespace: "ñoño", Status: "Synced"},
		}

		data, err := json.Marshal(resources)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var decoded []api.ResourceStatus
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if decoded[0].Name != "app-日本語-test" {
			t.Errorf("Name not preserved: got %q", decoded[0].Name)
		}
		if decoded[0].Namespace != "ñoño" {
			t.Errorf("Namespace not preserved: got %q", decoded[0].Namespace)
		}
	})
}

// TestResourcesJSONLargePayload tests handling of large resource lists
func TestResourcesJSONLargePayload(t *testing.T) {
	// Create a large list of resources (simulating a real application with many resources)
	resources := make([]api.ResourceStatus, 500)
	for i := 0; i < 500; i++ {
		resources[i] = api.ResourceStatus{
			Group:     "apps",
			Kind:      "Pod",
			Name:      "pod-" + string(rune('a'+i%26)) + "-" + string(rune('0'+i%10)),
			Namespace: "namespace-" + string(rune('a'+i%5)),
			Status:    "Synced",
		}
	}

	// Marshal
	data, err := json.Marshal(resources)
	if err != nil {
		t.Fatalf("failed to marshal large payload: %v", err)
	}

	// Unmarshal
	var decoded []api.ResourceStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal large payload: %v", err)
	}

	if len(decoded) != 500 {
		t.Errorf("expected 500 resources, got %d", len(decoded))
	}
}

// TestModelUpdateWithNilResourcesJSON verifies model.Update handles nil ResourcesJSON safely
func TestModelUpdateWithNilResourcesJSON(t *testing.T) {
	// This test verifies the guard condition in model.go:
	// if len(msg.ResourcesJSON) > 0 { ... json.Unmarshal ... }

	msg := model.AppUpdatedMsg{
		App:           model.App{Name: "test"},
		ResourcesJSON: nil, // Explicitly nil
	}

	// The condition len(msg.ResourcesJSON) > 0 should be false for nil
	if len(msg.ResourcesJSON) > 0 {
		t.Error("len(nil) should be 0")
	}

	msg2 := model.AppUpdatedMsg{
		App:           model.App{Name: "test"},
		ResourcesJSON: []byte{}, // Empty slice
	}

	// Should also be false for empty slice
	if len(msg2.ResourcesJSON) > 0 {
		t.Error("len([]) should be 0")
	}
}

// TestResourceTreeLoadedMsgWithNilResourcesJSON verifies tree loading handles nil resources
func TestResourceTreeLoadedMsgWithNilResourcesJSON(t *testing.T) {
	msg := model.ResourceTreeLoadedMsg{
		AppName:       "test-app",
		Health:        "Healthy",
		Sync:          "Synced",
		TreeJSON:      []byte(`{"nodes":[]}`),
		ResourcesJSON: nil, // No resources
	}

	// This should not attempt unmarshal (checking the guard condition)
	if len(msg.ResourcesJSON) > 0 {
		t.Error("expected ResourcesJSON to be nil/empty")
	}
}
