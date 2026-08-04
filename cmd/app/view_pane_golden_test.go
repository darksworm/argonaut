package main

import (
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
)

// buildPaneGoldenModel returns a model in tree view at the given size with a
// deterministic clock and a small demo tree.
func buildPaneGoldenModel(cols, rows int) *Model {
	m := buildBaseModel(cols, rows)
	m.state.Navigation.View = model.ViewTree
	fixedNow := time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)
	m.now = func() time.Time { return fixedNow }
	m.treeView.SetClock(m.now)

	ns := "demo"
	degraded := "Degraded"
	healthy := "Healthy"
	m.state.Apps = []model.App{{Name: "demo-app", Sync: "Synced", Health: "Degraded"}}
	m.treeView.SetAppMeta("demo-app", "Degraded", "Synced")
	m.treeView.UpsertAppTree("demo-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "svc-1", Kind: "Service", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "deploy-1", Kind: "Deployment", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &degraded}},
		{UID: "pod-1", Kind: "Pod", Name: "web-6f7d9b-x4k2m", Namespace: &ns, Health: &api.ResourceHealth{Status: &degraded},
			ParentRefs: []api.ResourceRef{{UID: "deploy-1"}}},
	}})
	return m
}

func samplePodEvents() []model.ResourceEvent {
	return []model.ResourceEvent{
		{
			Type:     "Warning",
			Reason:   "BackOff",
			Message:  "Back-off restarting failed container web in pod web-6f7d9b-x4k2m_demo(a1b2c3d)",
			Count:    412,
			LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		},
		{
			Type:     "Normal",
			Reason:   "Pulled",
			Message:  "Container image \"web:1.4.2\" already present on machine",
			Count:    413,
			LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		},
		{
			Type:     "Warning",
			Reason:   "Unhealthy",
			Message:  "Readiness probe failed: Get \"http://10.0.3.12:8080/healthz\": connection refused",
			Count:    38,
			LastSeen: time.Date(2026, 8, 4, 11, 57, 0, 0, time.UTC),
		},
		{
			Type:     "Normal",
			Reason:   "Scheduled",
			Message:  "Successfully assigned demo/web-6f7d9b-x4k2m to node-2",
			Count:    1,
			LastSeen: time.Date(2026, 8, 4, 11, 2, 0, 0, time.UTC),
		},
	}
}

func openGoldenEventsPane(m *Model) {
	m.state.Mode = model.ModeEvents
	m.state.Events = &model.EventsState{
		Target: model.EventsTarget{
			AppName:  "demo-app",
			Resource: model.EventsResource{Kind: "Pod", Namespace: "demo", Name: "web-6f7d9b-x4k2m", UID: "pod-1"},
		},
		ResourceStatus: &model.ResourceStatusSummary{
			Health:        "Degraded",
			Sync:          "OutOfSync",
			HealthMessage: "containers with unready status: [web]",
			CreatedAt:     time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		},
		Details: &model.SyncStatusDetails{
			Phase: "Failed",
			Resources: []model.SyncResourceResult{
				{Kind: "Pod", Namespace: "demo", Name: "web-6f7d9b-x4k2m", Status: "SyncFailed", Message: "pod spec invalid"},
			},
		},
		Items: samplePodEvents(),
	}
}

func TestGolden_EventsPane_SideBySide(t *testing.T) {
	m := buildPaneGoldenModel(100, 24)
	openGoldenEventsPane(m)
	compareWithGolden(t, "pane_events_side_100x24", stripANSI(m.renderMainLayout()))
}

func TestGolden_EventsPane_BottomAt80x24(t *testing.T) {
	m := buildPaneGoldenModel(80, 24)
	openGoldenEventsPane(m)
	compareWithGolden(t, "pane_events_bottom_80x24", stripANSI(m.renderMainLayout()))
}

func TestGolden_EventsPane_ScrolledShowsBothMarkers(t *testing.T) {
	// Short terminal so the events overflow the pane in both directions
	m := buildPaneGoldenModel(100, 14)
	openGoldenEventsPane(m)
	m.state.Events.Offset = 4
	compareWithGolden(t, "pane_events_scrolled", stripANSI(m.renderMainLayout()))
}

func TestGolden_EventsPane_LoadingEmptyError(t *testing.T) {
	m := buildPaneGoldenModel(100, 24)
	l := m.paneLayout(10)

	m.state.Events = &model.EventsState{
		Target: model.EventsTarget{
			AppName:  "demo-app",
			Resource: model.EventsResource{Kind: "Pod", Namespace: "demo", Name: "web-6f7d9b-x4k2m", UID: "pod-1"},
		},
		Loading: true,
	}
	out := stripANSI(m.renderSidePane(l))

	m.state.Events.Loading = false
	out += "\n\n" + stripANSI(m.renderSidePane(l))

	m.state.Events.Error = "connection refused: dial tcp 10.0.0.1:443"
	out += "\n\n" + stripANSI(m.renderSidePane(l))

	compareWithGolden(t, "pane_events_loading_empty_error", out)
}

func TestGolden_EventsPane_AppRow_StatusBlockAboveEvents(t *testing.T) {
	m := buildPaneGoldenModel(100, 30)
	m.state.Mode = model.ModeEvents
	m.state.Events = &model.EventsState{
		Target: model.EventsTarget{AppName: "demo-app"},
		Details: &model.SyncStatusDetails{
			Phase:       "Failed",
			Message:     "one or more objects failed to apply",
			StartedAt:   time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			FinishedAt:  time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
			Revision:    "a1b2c3d4e5f6789",
			InitiatedBy: "alice",
			Resources: []model.SyncResourceResult{
				{Kind: "Service", Namespace: "demo", Name: "web", Status: "Synced", Message: "service/web unchanged"},
				{Kind: "Deployment", Namespace: "demo", Name: "web", Status: "SyncFailed",
					Message: "Deployment.apps \"web\" is invalid: spec.template.spec.containers[0].image: Required value"},
			},
		},
		Items: []model.ResourceEvent{
			{
				Type:     "Warning",
				Reason:   "OperationCompleted",
				Message:  "Sync operation to a1b2c3d failed: one or more objects failed to apply",
				Count:    1,
				LastSeen: time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
			},
			{
				Type:     "Normal",
				Reason:   "OperationStarted",
				Message:  "admin initiated sync to HEAD",
				Count:    1,
				LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
			},
		},
	}
	compareWithGolden(t, "pane_approw_status_and_events_100x30", stripANSI(m.renderMainLayout()))
}

func TestGolden_EventsPane_AppRow_NeverSynced(t *testing.T) {
	m := buildPaneGoldenModel(100, 24)
	m.state.Mode = model.ModeEvents
	m.state.Events = &model.EventsState{
		Target: model.EventsTarget{AppName: "demo-app"},
		Items:  samplePodEvents()[3:], // just the Scheduled event
	}
	compareWithGolden(t, "pane_approw_never_synced", stripANSI(m.renderSidePane(m.paneLayout(12))))
}

func TestGolden_TreeView_SyncSummaryVariants(t *testing.T) {
	m := buildPaneGoldenModel(100, 24)
	// A second app whose sync is running under an automation policy
	m.treeView.SetAppMeta("other-app", "Healthy", "Synced")
	m.treeView.UpsertAppTree("other-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "svc-2", Kind: "Service", Name: "api"},
	}})

	m.treeView.SetAppSyncSummary("demo-app", &model.SyncOpSummary{
		Phase:       "Failed",
		StartedAt:   time.Date(2026, 8, 4, 11, 59, 54, 0, time.UTC),
		FinishedAt:  time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		Revision:    "a1b2c3d4e5f6789",
		InitiatedBy: "alice",
	})
	m.treeView.SetAppSyncSummary("other-app", &model.SyncOpSummary{
		Phase:     "Running",
		StartedAt: time.Date(2026, 8, 4, 12, 1, 30, 0, time.UTC),
		Automated: true,
	})

	compareWithGolden(t, "tree_view_sync_summary_variants", stripANSI(m.renderTreePanel(14, m.state.Terminal.Cols)))
}
