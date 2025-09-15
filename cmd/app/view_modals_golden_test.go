package main

import (
    "testing"
    "time"

    "github.com/darksworm/argonaut/pkg/model"
)

func buildBaseModel(cols, rows int) *Model {
    m := NewModel()
    m.ready = true
    m.state.Terminal.Cols = cols
    m.state.Terminal.Rows = rows
    // Provide server + version so banner/context is stable
    m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
    m.state.APIVersion = "v2.10.3"
    return m
}

func TestGolden_HelpModal(t *testing.T) {
    m := buildBaseModel(100, 30)
    m.state.Mode = model.ModeHelp
    out := stripANSI(m.renderHelpModal())
    compareWithGolden(t, "modal_help", out)
}

func TestGolden_DiffLoadingSpinner(t *testing.T) {
    m := buildBaseModel(80, 24)
    // Use spinner initial frame deterministically
    out := stripANSI(m.renderDiffLoadingSpinner())
    compareWithGolden(t, "modal_diff_loading_spinner", out)
}

func TestGolden_SyncLoadingModal(t *testing.T) {
    m := buildBaseModel(80, 24)
    out := stripANSI(m.renderSyncLoadingModal())
    compareWithGolden(t, "modal_sync_loading", out)
}

func TestGolden_InitialLoadingModal(t *testing.T) {
    m := buildBaseModel(100, 30)
    out := stripANSI(m.renderInitialLoadingModal())
    compareWithGolden(t, "modal_initial_loading", out)
}

func sampleRollbackState() *model.RollbackState {
    // Fixed timestamps for determinism
    ts := time.Date(2024, 7, 1, 12, 34, 0, 0, time.UTC)
    author := "Jane Doe"
    msg := "Refactor and optimize"
    rev1 := "a1b2c3d4e5f6"
    rev2 := "112233445566"
    rev3 := "deadbeefcafebabe"
    rows := []model.RollbackRow{
        {ID: 30, Revision: rev1, DeployedAt: &ts, Author: &author, Message: &msg},
        {ID: 29, Revision: rev2, DeployedAt: &ts},
        {ID: 28, Revision: rev3},
    }
    cur := "cafebabedeadbeef"
    return &model.RollbackState{
        AppName:         "demo-app",
        Rows:            rows,
        SelectedIdx:     0,
        CurrentRevision: cur,
        Loading:         false,
        Error:           "",
        Mode:            "list",
        Prune:           false,
        Watch:           false,
        DryRun:          false,
        ConfirmSelected: 1,
    }
}

func TestGolden_RollbackModal_List(t *testing.T) {
    m := buildBaseModel(100, 32)
    app := "demo-app"
    m.state.Modals.RollbackAppName = &app
    m.state.Rollback = sampleRollbackState()
    out := stripANSI(m.renderRollbackModal())
    compareWithGolden(t, "modal_rollback_list", out)
}

func TestGolden_RollbackModal_Confirm(t *testing.T) {
    m := buildBaseModel(100, 32)
    app := "demo-app"
    m.state.Modals.RollbackAppName = &app
    rb := sampleRollbackState()
    rb.Mode = "confirm"
    rb.ConfirmSelected = 0
    m.state.Rollback = rb
    out := stripANSI(m.renderRollbackModal())
    compareWithGolden(t, "modal_rollback_confirm", out)
}

func TestGolden_Banner(t *testing.T) {
    m := buildBaseModel(120, 24)
    // Add some scope selections to exercise context lines
    m.state.Selections.AddCluster("prod")
    m.state.Selections.AddNamespace("payments")
    m.state.Selections.AddProject("billing")
    out := stripANSI(m.renderBanner())
    compareWithGolden(t, "banner", out)
}

func TestGolden_FullScreenLayout_Sample(t *testing.T) {
    m := buildBaseModel(80, 20)
    header := "Header Title"
    body := "Sample body line 1\nSample body line 2"
    status := "Status footer"
    out := stripANSI(m.renderFullScreenView(header, body, status, true))
    compareWithGolden(t, "layout_fullscreen_sample", out)
}

