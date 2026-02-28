package model

import (
	"time"
)

// View represents the current view in the navigation hierarchy
type View string

const (
	ViewClusters        View = "clusters"
	ViewNamespaces      View = "namespaces"
	ViewProjects        View = "projects"
	ViewApps            View = "apps"
	ViewTree            View = "tree"
	ViewApplicationSets View = "applicationsets"
)

// Mode represents the current application mode
type Mode string

const (
	ModeNormal                Mode = "normal"
	ModeLoading               Mode = "loading"
	ModeSearch                Mode = "search"
	ModeCommand               Mode = "command"
	ModeTheme                 Mode = "theme"
	ModeHelp                  Mode = "help"
	ModeConfirmSync           Mode = "confirm-sync"
	ModeRollback              Mode = "rollback"
	ModeConfirmAppDelete      Mode = "confirm-app-delete"
	ModeConfirmResourceDelete Mode = "confirm-resource-delete"
	ModeExternal              Mode = "external"
	ModeDiff                  Mode = "diff"
	ModeAuthRequired          Mode = "auth-required"
	ModeRulerLine             Mode = "rulerline"
	ModeError                 Mode = "error"
	ModeConnectionError       Mode = "connection-error"
	ModeCoreDetected          Mode = "core-detected"
	ModeUpgrade               Mode = "upgrade"
	ModeUpgradeError          Mode = "upgrade-error"
	ModeUpgradeSuccess        Mode = "upgrade-success"
	ModeNoDiff                Mode = "no-diff"
	ModeK9sContextSelect      Mode = "k9s-context-select"
	ModeK9sError              Mode = "k9s-error"
	ModeConfirmResourceSync   Mode = "confirm-resource-sync"
	ModeDefaultViewWarning    Mode = "default-view-warning"
)

// App represents an ArgoCD application
type App struct {
	Name           string     `json:"name"`
	Sync           string     `json:"sync"`
	Health         string     `json:"health"`
	LastSyncAt     *time.Time `json:"lastSyncAt,omitempty"`
	Project        *string    `json:"project,omitempty"`
	ClusterID      *string    `json:"clusterId,omitempty"`
	ClusterLabel   *string    `json:"clusterLabel,omitempty"`
	Namespace      *string    `json:"namespace,omitempty"`
	AppNamespace   *string    `json:"appNamespace,omitempty"`
	ApplicationSet *string    `json:"applicationSet,omitempty"`
}

// SortKey returns the values used for semantic ordering of apps.
func (a App) SortKey() SortKey {
	return SortKey{Health: a.Health, Sync: a.Sync, Name: a.Name}
}

// Server represents an ArgoCD server configuration
type Server struct {
	BaseURL         string `json:"baseUrl"`
	Token           string `json:"token"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
	Insecure        bool   `json:"insecure,omitempty"`
	GrpcWebRootPath string `json:"grpcWebRootPath,omitempty"`
}

// TerminalState represents terminal dimensions
type TerminalState struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// Helper methods for set operations using map[string]bool

// NewStringSet creates a new string set
func NewStringSet() map[string]bool {
	return make(map[string]bool)
}

// StringSetFromSlice creates a string set from a slice
func StringSetFromSlice(items []string) map[string]bool {
	set := make(map[string]bool)
	for _, item := range items {
		set[item] = true
	}
	return set
}

// AddToStringSet adds an item to a string set
func AddToStringSet(set map[string]bool, item string) map[string]bool {
	if set == nil {
		set = make(map[string]bool)
	}
	set[item] = true
	return set
}

// RemoveFromStringSet removes an item from a string set
func RemoveFromStringSet(set map[string]bool, item string) map[string]bool {
	if set != nil {
		delete(set, item)
	}
	return set
}

// HasInStringSet checks if an item exists in a string set
func HasInStringSet(set map[string]bool, item string) bool {
	return set != nil && set[item]
}

// ResourceNode represents a Kubernetes resource in the ArgoCD application tree
type ResourceNode struct {
	Kind           string          `json:"kind"`
	Name           string          `json:"name"`
	Namespace      *string         `json:"namespace,omitempty"`
	Version        string          `json:"version"`
	Group          string          `json:"group"`
	UID            string          `json:"uid"`
	Health         *ResourceHealth `json:"health,omitempty"`
	Status         string          `json:"status"`
	NetworkingInfo *NetworkingInfo `json:"networkingInfo,omitempty"`
	ResourceRef    ResourceRef     `json:"resourceRef"`
	ParentRefs     []ResourceRef   `json:"parentRefs,omitempty"`
	Info           []ResourceInfo  `json:"info,omitempty"`
	CreatedAt      *time.Time      `json:"createdAt,omitempty"`
}

// ResourceHealth represents the health status of a resource
type ResourceHealth struct {
	Status  *string `json:"status,omitempty"`
	Message *string `json:"message,omitempty"`
}

// NetworkingInfo represents networking information for a resource
type NetworkingInfo struct {
	TargetLabels map[string]string `json:"targetLabels,omitempty"`
	TargetRefs   []ResourceRef     `json:"targetRefs,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Ingress      []IngressInfo     `json:"ingress,omitempty"`
}

// IngressInfo represents ingress information
type IngressInfo struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// ResourceRef represents a reference to a Kubernetes resource
type ResourceRef struct {
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
	Group     string  `json:"group"`
	Version   string  `json:"version"`
	UID       string  `json:"uid"`
}

// ResourceInfo represents additional information about a resource
type ResourceInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// RollbackRow represents a deployment history entry for rollback selection
type RollbackRow struct {
	ID         int        `json:"id"`         // Deployment ID
	Revision   string     `json:"revision"`   // Git SHA/revision
	DeployedAt *time.Time `json:"deployedAt"` // Deployment timestamp
	Author     *string    `json:"author"`     // Git author (loaded async)
	Date       *time.Time `json:"date"`       // Git commit date
	Message    *string    `json:"message"`    // Git commit message
	MetaError  *string    `json:"metaError"`  // Error loading metadata
}

// RollbackState holds the state for rollback operations
type RollbackState struct {
	AppName         string        `json:"appName"`         // App being rolled back
	AppNamespace    *string       `json:"appNamespace"`    // App namespace
	Rows            []RollbackRow `json:"rows"`            // Deployment history
	SelectedIdx     int           `json:"selectedIdx"`     // Currently selected row
	CurrentRevision string        `json:"currentRevision"` // Current deployment revision
	Loading         bool          `json:"loading"`         // Loading state
	Error           string        `json:"error"`           // Error message
	Mode            string        `json:"mode"`            // "list" or "confirm"
	Prune           bool          `json:"prune"`           // Prune option
	Watch           bool          `json:"watch"`           // Watch option after rollback
	DryRun          bool          `json:"dryRun"`          // Dry run option (not shown in confirm view)
	ConfirmSelected int           `json:"confirmSelected"` // 0 = Yes, 1 = No/Cancel
}

// RevisionMetadata represents git commit metadata for a revision
type RevisionMetadata struct {
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Tags    []string  `json:"tags,omitempty"`
}

// RollbackRequest represents a rollback API request
type RollbackRequest struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	DryRun       bool    `json:"dryRun,omitempty"`
	Prune        bool    `json:"prune,omitempty"`
	AppNamespace *string `json:"appNamespace,omitempty"`
}

// InstallMethod represents how argonaut was installed
type InstallMethod string

const (
	InstallMethodUnknown InstallMethod = "unknown"
	InstallMethodBrew    InstallMethod = "brew"
	InstallMethodAUR     InstallMethod = "aur"
	InstallMethodDocker  InstallMethod = "docker"
	InstallMethodManual  InstallMethod = "manual"
)

// UpdateInfo contains information about available updates
type UpdateInfo struct {
	Available           bool          `json:"available"`
	CurrentVersion      string        `json:"current_version"`
	LatestVersion       string        `json:"latest_version"`
	PublishedAt         time.Time     `json:"published_at"`
	InstallMethod       InstallMethod `json:"install_method"`
	DownloadURL         string        `json:"download_url,omitempty"`
	ChecksumURL         string        `json:"checksum_url,omitempty"`
	ChecksumSHA256      string        `json:"checksum_sha256,omitempty"`
	LastChecked         time.Time     `json:"last_checked"`
	CheckIntervalMin    int           `json:"check_interval_min"`
	NotificationShownAt *time.Time    `json:"notification_shown_at,omitempty"`
}

// ResourceDeleteTarget represents a resource to be deleted
type ResourceDeleteTarget struct {
	AppName   string `json:"appName"`
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"version"`
}

// ResourceSyncTarget represents a resource to be synced
type ResourceSyncTarget struct {
	AppName   string `json:"appName"`
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}
