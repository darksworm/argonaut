package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

// buildRollbackGoldenModel returns a model in the rollback view at the given
// size with a deterministic clock and a four-entry history: the current
// deployment, the selected row (full metadata), one still loading, one failed.
func buildRollbackGoldenModel(cols, rows int) *Model {
	m := buildBaseModel(cols, rows)
	fixedNow := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	m.now = func() time.Time { return fixedNow }

	ts := func(daysAgo int, hour int) *time.Time {
		t := time.Date(2026, 8, 8-daysAgo, hour, 2, 0, 0, time.UTC)
		return &t
	}
	author := "Jane Doe"
	authorDate := time.Date(2026, 8, 7, 13, 58, 0, 0, time.UTC)
	message := "fix: suggest and validate :events arguments"
	metaError := "rpc error: commit not found"
	started := time.Date(2026, 8, 7, 14, 1, 46, 0, time.UTC)

	m.state.Mode = model.ModeRollback
	m.state.Rollback = &model.RollbackState{
		AppName:         "demo-app",
		CurrentRevision: "a1b2c3d4e5f6",
		SelectedIdx:     1,
		Mode:            "list",
		Watch:           true,
		AutoSyncEnabled: true,
		Rows: []model.RollbackRow{
			{ID: 30, Revision: "a1b2c3d4e5f6", DeployedAt: ts(0, 10), InitiatedBy: "jane.doe",
				Author: &author, Date: &authorDate, Message: &message},
			{ID: 29, Revision: "1122334455aa", DeployedAt: ts(1, 14), DeployStartedAt: &started, Automated: true,
				Author: &author, Date: &authorDate, Message: &message,
				Source: &model.RollbackSource{RepoURL: "https://github.com/corp/example-apps", Path: "apps/demo", TargetRevision: "main"}},
			{ID: 28, Revision: "deadbeef0011", DeployedAt: ts(3, 9), Automated: true},
			{ID: 27, Revision: "5566778899bb", DeployedAt: ts(4, 16), InitiatedBy: "bob.smith", MetaError: &metaError},
		},
	}
	return m
}

func TestGolden_RollbackPane_100x24(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	compareWithGolden(t, "rollback_pane_100x24", stripANSI(m.renderRollbackLayout()))
}

func TestGolden_RollbackPane_80x24(t *testing.T) {
	m := buildRollbackGoldenModel(80, 24)
	compareWithGolden(t, "rollback_pane_80x24", stripANSI(m.renderRollbackLayout()))
}

// One golden pins every non-browse pane state: history loading, load error,
// empty history, metadata loading, metadata error.
func TestGolden_RollbackPane_States(t *testing.T) {
	var out []string
	render := func(mutate func(m *Model)) {
		m := buildRollbackGoldenModel(100, 16)
		mutate(m)
		out = append(out, stripANSI(m.renderRollbackLayout()))
	}

	render(func(m *Model) { m.state.Rollback.Loading = true })
	render(func(m *Model) { m.state.Rollback.Error = "connection refused" })
	render(func(m *Model) { m.state.Rollback.Rows = nil })
	render(func(m *Model) { m.state.Rollback.SelectedIdx = 2 }) // metadata still loading
	render(func(m *Model) { m.state.Rollback.SelectedIdx = 3 }) // metadata failed

	compareWithGolden(t, "rollback_pane_states", strings.Join(out, "\n\n"))
}

func TestGolden_RollbackPane_Confirm(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	m.state.Rollback.Mode = "confirm"
	compareWithGolden(t, "rollback_pane_confirm", stripANSI(m.renderRollbackLayout()))
}

// The newest entry is the current deployment: confirming it is a redeploy,
// and with auto-sync off there is no warning.
func TestGolden_RollbackPane_ConfirmRedeploy(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	m.state.Rollback.Mode = "confirm"
	m.state.Rollback.SelectedIdx = 0
	m.state.Rollback.AutoSyncEnabled = false
	compareWithGolden(t, "rollback_pane_confirm_redeploy", stripANSI(m.renderRollbackLayout()))
}
