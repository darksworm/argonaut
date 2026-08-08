package main

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// clearFooterNoticeMsg removes an expired transient footer hint.
type clearFooterNoticeMsg struct{ notice string }

// showFooterNotice puts a brief hint in the status bar — status messages
// only reach the log file, so user-facing refusals must go through here.
func (m *Model) showFooterNotice(text string) tea.Cmd {
	m.state.UI.FooterNotice = text
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearFooterNoticeMsg{notice: text}
	})
}

// rollbackMetaDueMsg fires when the metadata-fetch debounce elapses.
type rollbackMetaDueMsg struct {
	switchEpoch int
	loadSeq     int
}

// rollbackDiffNoChangesMsg reports a revision diff with no changes. It must
// not leave the rollback view (a NoDiff modal dismisses to the apps list) —
// it becomes a notice in the detail pane instead.
type rollbackDiffNoChangesMsg struct {
	switchEpoch int
	revision    string
}

func (m *Model) handleRollbackDiffNoChanges(msg rollbackDiffNoChangesMsg) {
	rb := m.state.Rollback
	if rb == nil || msg.switchEpoch != m.switchEpoch {
		return
	}
	rb.Notice = "No differences vs " + msg.revision
}

// scheduleRollbackMetaFetch arms the metadata debounce for the given load.
func (m *Model) scheduleRollbackMetaFetch(loadSeq int) tea.Cmd {
	epoch := m.switchEpoch
	return tea.Tick(paneFetchDebounce, func(time.Time) tea.Msg {
		return rollbackMetaDueMsg{switchEpoch: epoch, loadSeq: loadSeq}
	})
}

// retargetRollbackPane reacts to a cursor move: the detail pane is a lens
// over the selection, so a debounced metadata fetch is armed for the new row.
func (m *Model) retargetRollbackPane() tea.Cmd {
	rb := m.state.Rollback
	if rb == nil {
		return nil
	}
	rb.LoadSeq++
	rb.DetailOffset = 0 // the pane shows a new row — scroll back to its top
	rb.Notice = ""
	return m.scheduleRollbackMetaFetch(rb.LoadSeq)
}

// handleRollbackMetaDue fetches metadata for the row under the cursor once
// the debounce elapses, unless a newer cursor move superseded this tick.
func (m *Model) handleRollbackMetaDue(msg rollbackMetaDueMsg) tea.Cmd {
	rb := m.state.Rollback
	if rb == nil || msg.switchEpoch != m.switchEpoch || msg.loadSeq != rb.LoadSeq {
		return nil
	}
	return m.fetchVisibleRollbackMetadata()
}

// rollbackVisibleWindow is the row range the list shows for the given
// cursor position — the fetcher and the renderer must agree on it.
func rollbackVisibleWindow(selectedIdx, total, capacity int) (start, end int) {
	capacity = max(1, capacity)
	start = min(max(0, selectedIdx-capacity/2), max(0, total-capacity))
	return start, min(start+capacity, total)
}

// rollbackRowForMetadataReply resolves a metadata reply to the row it was
// fetched for, or nil when the reply is stale: another server context,
// another app's session, or a history that has been reloaded since.
func (m *Model) rollbackRowForMetadataReply(switchEpoch int, appName, revision string, rowIndex int) *model.RollbackRow {
	rb := m.state.Rollback
	if rb == nil || switchEpoch != m.switchEpoch || rb.AppName != appName {
		return nil
	}
	if rowIndex < 0 || rowIndex >= len(rb.Rows) || rb.Rows[rowIndex].Revision != revision {
		return nil
	}
	return &rb.Rows[rowIndex]
}

// fetchVisibleRollbackMetadata loads git metadata for every visible row
// that hasn't loaded (or failed) yet — the list shows each row's commit
// subject, so the whole window is needed, not just the cursor row.
func (m *Model) fetchVisibleRollbackMetadata() tea.Cmd {
	rb := m.state.Rollback
	if rb == nil {
		return nil
	}
	start, end := rollbackVisibleWindow(rb.SelectedIdx, len(rb.Rows), m.rollbackPageSize())
	var cmds []tea.Cmd
	for i := start; i < end; i++ {
		row := rb.Rows[i]
		if row.Author == nil && row.MetaError == nil {
			cmds = append(cmds, m.loadRevisionMetadata(rb.AppName, i, row.Revision, rb.AppNamespace))
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}
