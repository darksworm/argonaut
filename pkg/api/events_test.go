package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestListEvents_ApplicationLevel_FetchesAndNormalizes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/applications/demo-app/events" {
			t.Errorf("expected path /api/v1/applications/demo-app/events, got %s", r.URL.Path)
		}
		if len(r.URL.Query()) != 0 {
			t.Errorf("expected no query params for app-level events, got %v", r.URL.Query())
		}
		w.Write([]byte(`{"items":[{
			"type": "Warning",
			"reason": "BackOff",
			"message": "Back-off restarting failed container web",
			"count": 412,
			"firstTimestamp": "2026-08-04T10:00:00Z",
			"lastTimestamp": "2026-08-04T12:00:00Z"
		}]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []model.ResourceEvent{{
		Type:     "Warning",
		Reason:   "BackOff",
		Message:  "Back-off restarting failed container web",
		Count:    412,
		LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("expected %+v, got %+v", want, events)
	}
}

func TestListEvents_ResourceScoped_SendsAllResourceParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("resourceUID"); got != "abc-123" {
			t.Errorf("expected resourceUID=abc-123, got %q", got)
		}
		if got := q.Get("resourceName"); got != "web-6f7d9b-x4k2m" {
			t.Errorf("expected resourceName=web-6f7d9b-x4k2m, got %q", got)
		}
		if got := q.Get("resourceNamespace"); got != "demo" {
			t.Errorf("expected resourceNamespace=demo, got %q", got)
		}
		if got := q.Get("appNamespace"); got != "argocd" {
			t.Errorf("expected appNamespace=argocd, got %q", got)
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	_, err := svc.ListEvents(context.Background(), ListEventsParams{
		AppName:           "demo-app",
		AppNamespace:      "argocd",
		ResourceUID:       "abc-123",
		ResourceName:      "web-6f7d9b-x4k2m",
		ResourceNamespace: "demo",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// Cluster-scoped resources (Namespace, ClusterRole, …) have no namespace;
// the server accepts uid+name with an empty resourceNamespace (verified
// against a live Argo CD).
func TestListEvents_ClusterScopedResource_SendsEmptyNamespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("resourceUID"); got != "ns-uid" {
			t.Errorf("expected resourceUID=ns-uid, got %q", got)
		}
		if got := q.Get("resourceName"); got != "argonaut-demo" {
			t.Errorf("expected resourceName=argonaut-demo, got %q", got)
		}
		if !q.Has("resourceNamespace") || q.Get("resourceNamespace") != "" {
			t.Errorf("expected an empty resourceNamespace param, got %q", q.Get("resourceNamespace"))
		}
		w.Write([]byte(`{"items":[]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	_, err := svc.ListEvents(context.Background(), ListEventsParams{
		AppName:      "rollout-bluegreen",
		ResourceUID:  "ns-uid",
		ResourceName: "argonaut-demo",
	})
	if err != nil {
		t.Fatalf("expected cluster-scoped resource events to be fetchable, got %v", err)
	}
}

func TestListEvents_PartialResourceParams_FailsWithoutRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("expected no HTTP request for partial resource params, got %s %s", r.Method, r.URL)
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	_, err := svc.ListEvents(context.Background(), ListEventsParams{
		AppName:     "demo-app",
		ResourceUID: "abc-123", // name and namespace missing
	})
	if err == nil {
		t.Fatal("expected an error for partial resource params, got nil")
	}
}

func TestListEvents_EventsK8sIoStyle_FallsBackToEventTimeAndSeriesCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{
			"type": "Normal",
			"reason": "Pulled",
			"message": "Container image already present on machine",
			"eventTime": "2026-08-04T11:30:00Z",
			"series": {"count": 7}
		}]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []model.ResourceEvent{{
		Type:     "Normal",
		Reason:   "Pulled",
		Message:  "Container image already present on machine",
		Count:    7,
		LastSeen: time.Date(2026, 8, 4, 11, 30, 0, 0, time.UTC),
	}}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("expected %+v, got %+v", want, events)
	}
}

func TestListEvents_EventWithoutCountOrSeries_CountsAsOne(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{
			"type": "Normal",
			"reason": "Scheduled",
			"message": "Successfully assigned demo/web to node-2",
			"lastTimestamp": "2026-08-04T11:00:00Z"
		}]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Count != 1 {
		t.Errorf("expected count to default to 1, got %d", events[0].Count)
	}
}

func TestListEvents_BareArrayResponse_IsAccepted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{
			"type": "Warning",
			"reason": "Unhealthy",
			"message": "Readiness probe failed",
			"count": 38,
			"lastTimestamp": "2026-08-04T11:55:00Z"
		}]`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(events) != 1 || events[0].Reason != "Unhealthy" {
		t.Errorf("expected the Unhealthy event from a bare array, got %+v", events)
	}
}

func TestListEvents_ReturnsEventsNewestFirst(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[
			{"reason": "Scheduled", "lastTimestamp": "2026-08-04T10:00:00Z"},
			{"reason": "BackOff", "lastTimestamp": "2026-08-04T12:00:00Z"},
			{"reason": "Pulled", "eventTime": "2026-08-04T11:00:00Z"}
		]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var reasons []string
	for _, e := range events {
		reasons = append(reasons, e.Reason)
	}
	want := []string{"BackOff", "Pulled", "Scheduled"}
	if !reflect.DeepEqual(reasons, want) {
		t.Errorf("expected order %v, got %v", want, reasons)
	}
}

// Helm/controller errors embed newlines and tabs; a message must reach the
// renderer as a single line or it breaks the pane frame row accounting.
func TestListEvents_MultilineMessage_IsFlattenedToOneLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{
			"type": "Warning",
			"reason": "OperationCompleted",
			"message": "failed: line one\n\nError: bad yaml:\n\t12:30\texecuting template",
			"count": 1,
			"lastTimestamp": "2026-08-04T12:00:00Z"
		}]}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	events, err := svc.ListEvents(context.Background(), ListEventsParams{AppName: "demo-app"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "failed: line one Error: bad yaml: 12:30 executing template"
	if events[0].Message != want {
		t.Errorf("expected whitespace runs collapsed to single spaces,\nwant %q\ngot  %q", want, events[0].Message)
	}
}

func TestSortEventsNewestFirst_OrdersByLastSeenDescending(t *testing.T) {
	older := model.ResourceEvent{Reason: "Scheduled", LastSeen: time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)}
	newer := model.ResourceEvent{Reason: "BackOff", LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)}

	events := []model.ResourceEvent{older, newer}
	SortEventsNewestFirst(events)

	if events[0].Reason != "BackOff" {
		t.Errorf("expected newest event (BackOff) first, got %s", events[0].Reason)
	}
	if events[1].Reason != "Scheduled" {
		t.Errorf("expected oldest event (Scheduled) last, got %s", events[1].Reason)
	}
}
