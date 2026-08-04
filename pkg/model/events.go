package model

import "time"

// ResourceEvent is a Kubernetes event scoped to an application or one of its
// resources, normalized for display (timestamp and count fallbacks resolved).
type ResourceEvent struct {
	Type     string    `json:"type"` // "Normal" or "Warning"
	Reason   string    `json:"reason"`
	Message  string    `json:"message"`
	Count    int       `json:"count"`
	LastSeen time.Time `json:"lastSeen"`
}

// EventsResource identifies one resource within an application's tree.
// The zero value means "the application itself" (application-level events).
type EventsResource struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
}

// EventsTarget identifies whose events the pane shows. Value-only fields so
// targets compare with == for async message gating ("" = unset namespace).
type EventsTarget struct {
	AppName      string         `json:"appName"`
	AppNamespace string         `json:"appNamespace"`
	Resource     EventsResource `json:"resource"`
}

// EventsState holds the state of the events side pane; its presence on
// AppState is the "pane is open" signal. LoadSeq identifies the fetch this
// pane is waiting for — a reopened pane shares epoch and target with the
// load it superseded, so those alone cannot gate late completions.
type EventsState struct {
	Target  EventsTarget    `json:"target"`
	Items   []ResourceEvent `json:"items"`
	Offset  int             `json:"offset"`
	Loading bool            `json:"loading"`
	Error   string          `json:"error"`
	// Notice explains why this target can have no events (e.g. the resource
	// is Missing from the cluster) — informational, unlike Error
	Notice  string `json:"notice,omitempty"`
	LoadSeq int    `json:"-"`
}

// SyncStatusTarget identifies the application whose sync status the pane shows.
type SyncStatusTarget struct {
	AppName      string `json:"appName"`
	AppNamespace string `json:"appNamespace"`
}

// SyncStatusState holds the state of the sync-status side pane; its presence
// on AppState is the "pane is open" signal. See EventsState for LoadSeq.
type SyncStatusState struct {
	Target  SyncStatusTarget   `json:"target"`
	Details *SyncStatusDetails `json:"details"` // nil after load = never synced
	Offset  int                `json:"offset"`
	Loading bool               `json:"loading"`
	Error   string             `json:"error"`
	LoadSeq int                `json:"-"`
}
