package model

import (
	"time"
)

// View represents the current view in the navigation hierarchy
type View string

const (
	ViewClusters   View = "clusters"
	ViewNamespaces View = "namespaces"
	ViewProjects   View = "projects"
	ViewApps       View = "apps"
)

// Mode represents the current application mode
type Mode string

const (
	ModeNormal       Mode = "normal"
	ModeLoading      Mode = "loading"
	ModeSearch       Mode = "search"
	ModeCommand      Mode = "command"
	ModeHelp         Mode = "help"
	ModeLicense      Mode = "license"
	ModeConfirmSync  Mode = "confirm-sync"
	ModeRollback     Mode = "rollback"
    ModeExternal     Mode = "external"
    ModeDiff         Mode = "diff"
    ModeDiffLoading  Mode = "diff-loading"
    ModeResources    Mode = "resources"
	ModeAuthRequired Mode = "auth-required"
	ModeRulerLine    Mode = "rulerline"
	ModeError        Mode = "error"
	ModeLogs         Mode = "logs"
)

// App represents an ArgoCD application
type App struct {
	Name         string     `json:"name"`
	Sync         string     `json:"sync"`
	Health       string     `json:"health"`
	LastSyncAt   *time.Time `json:"lastSyncAt,omitempty"`
	Project      *string    `json:"project,omitempty"`
	ClusterID    *string    `json:"clusterId,omitempty"`
	ClusterLabel *string    `json:"clusterLabel,omitempty"`
	Namespace    *string    `json:"namespace,omitempty"`
	AppNamespace *string    `json:"appNamespace,omitempty"`
}

// Server represents an ArgoCD server configuration
type Server struct {
	BaseURL   string `json:"baseUrl"`
	Token     string `json:"token"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Insecure  bool   `json:"insecure,omitempty"`
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

// StringSetToSlice converts a string set to a slice
func StringSetToSlice(set map[string]bool) []string {
	var result []string
	for key := range set {
		result = append(result, key)
	}
	return result
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
