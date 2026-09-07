package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// The pane can be opened before the resource tree has loaded, and until it
// has, the tree cannot name the app the cursor sits in.
func TestPaneFetch_WithoutAnAppName_FetchesNothing(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Events = &model.EventsState{
		Target:         model.EventsTarget{},
		Loading:        true,
		DetailsLoading: true,
		LoadSeq:        1,
	}

	if cmd := m.paneFetchCmds(); cmd != nil {
		t.Error("expected no request while the app is unknown: it would GET /api/v1/applications/ " +
			"and the server's 403 surfaces as a bare \"permission denied\"")
	}
}

func TestPaneFetch_WithAnAppName_Fetches(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Events = &model.EventsState{
		Target:         model.EventsTarget{AppName: "demo-app"},
		Loading:        true,
		DetailsLoading: true,
		LoadSeq:        1,
	}

	if cmd := m.paneFetchCmds(); cmd == nil {
		t.Error("expected the pane to fetch once the app is known")
	}
}

// The pane stays in its loading state so the already-armed refresh retries
// once the tree has named the app.
func TestPaneFetch_WithoutAnAppName_StaysLoadingSoTheRefreshRetries(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Events = &model.EventsState{
		Target:         model.EventsTarget{},
		Loading:        true,
		DetailsLoading: true,
		LoadSeq:        1,
	}

	m.paneFetchCmds()

	if !m.state.Events.Loading || !m.state.Events.DetailsLoading {
		t.Error("expected the pane to remain loading so the next refresh picks the app up")
	}
}

// The refresh is the retry the initial skip relies on, so it needs the same
// guard: until the tree names the app, it would fetch the unnamed application.
func TestPaneRefresh_WithoutAnAppName_FetchesNothing(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Events = &model.EventsState{
		Target:  model.EventsTarget{},
		LoadSeq: 1,
	}

	if cmd := m.paneRefreshCmds(); cmd != nil {
		t.Error("expected the scheduled refresh to skip an unnamed application")
	}
}

func TestPaneRefresh_WithAnAppName_Fetches(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Events = &model.EventsState{
		Target:  model.EventsTarget{AppName: "demo-app"},
		LoadSeq: 1,
	}

	if cmd := m.paneRefreshCmds(); cmd == nil {
		t.Error("expected the refresh to fetch once the app is known")
	}
}

// A pane opened before the resource tree arrives starts with an unnamed
// target. The tree load must name that existing pane and start its pending
// requests; waiting for a scheduled refresh is not sufficient when refresh is
// disabled.
func TestResourceTreeLoaded_NamesAndFetchesAnAlreadyOpenPane(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Events = &model.EventsState{
		Target:         model.EventsTarget{},
		Loading:        true,
		DetailsLoading: true,
		LoadSeq:        1,
	}

	teaModel, cmd := m.Update(model.ResourceTreeLoadedMsg{
		AppName: "test-app", SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	want := model.EventsTarget{AppName: "test-app", AppNamespace: "test-namespace"}
	if mm.state.Events.Target != want {
		t.Errorf("expected the loaded tree to name the pane target %+v, got %+v", want, mm.state.Events.Target)
	}
	if cmd == nil {
		t.Error("expected the named pane to start its pending fetches")
	}
}
