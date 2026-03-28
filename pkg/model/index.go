package model

import "sort"

// AppIndex holds pre-computed indices over the app list for O(1) lookups.
// Rebuilt every time m.state.Apps is mutated.
type AppIndex struct {
	// Pre-computed sorted unique dimension values (derived from ALL apps)
	Clusters        []string
	Namespaces      []string
	Projects        []string
	ApplicationSets []string

	// Reverse mappings: dimension value → app indices in the Apps slice
	ByCluster        map[string][]int
	ByNamespace      map[string][]int
	ByProject        map[string][]int
	ByApplicationSet map[string][]int

	// App name → index in the Apps slice for O(1) upsert/delete
	NameToIndex map[string]int

	// Total number of apps when the index was built
	Total int
}

// BuildAppIndex constructs an AppIndex from the given app slice.
// The caller must rebuild whenever the Apps slice is mutated.
func BuildAppIndex(apps []App) *AppIndex {
	idx := &AppIndex{
		ByCluster:        make(map[string][]int),
		ByNamespace:      make(map[string][]int),
		ByProject:        make(map[string][]int),
		ByApplicationSet: make(map[string][]int),
		NameToIndex:      make(map[string]int, len(apps)),
		Total:            len(apps),
	}

	clusterSet := make(map[string]bool)
	nsSet := make(map[string]bool)
	projSet := make(map[string]bool)
	appsetSet := make(map[string]bool)

	for i, app := range apps {
		idx.NameToIndex[app.Name] = i

		// Cluster
		cl := ""
		if app.ClusterLabel != nil {
			cl = *app.ClusterLabel
		}
		if cl != "" {
			clusterSet[cl] = true
			idx.ByCluster[cl] = append(idx.ByCluster[cl], i)
		}

		// Namespace
		ns := ""
		if app.Namespace != nil {
			ns = *app.Namespace
		}
		if ns != "" {
			nsSet[ns] = true
			idx.ByNamespace[ns] = append(idx.ByNamespace[ns], i)
		}

		// Project
		prj := ""
		if app.Project != nil {
			prj = *app.Project
		}
		if prj != "" {
			projSet[prj] = true
			idx.ByProject[prj] = append(idx.ByProject[prj], i)
		}

		// ApplicationSet
		if app.ApplicationSet != nil && *app.ApplicationSet != "" {
			as := *app.ApplicationSet
			appsetSet[as] = true
			idx.ByApplicationSet[as] = append(idx.ByApplicationSet[as], i)
		}
	}

	idx.Clusters = sortedKeys(clusterSet)
	idx.Namespaces = sortedKeys(nsSet)
	idx.Projects = sortedKeys(projSet)
	idx.ApplicationSets = sortedKeys(appsetSet)

	return idx
}

// sortedKeys extracts keys from a bool map and returns them sorted.
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ScopedNamespaces returns sorted unique namespaces for apps matching the cluster scope.
// If clusterScope is empty, returns all namespaces.
func (idx *AppIndex) ScopedNamespaces(apps []App, clusterScope map[string]bool) []string {
	if idx == nil {
		return nil
	}
	if len(clusterScope) == 0 {
		return idx.Namespaces
	}
	seen := make(map[string]bool)
	for cl, ok := range clusterScope {
		if !ok {
			continue
		}
		for _, i := range idx.ByCluster[cl] {
			if i < len(apps) {
				ns := ""
				if apps[i].Namespace != nil {
					ns = *apps[i].Namespace
				}
				if ns != "" {
					seen[ns] = true
				}
			}
		}
	}
	return sortedKeys(seen)
}

// ScopedProjects returns sorted unique projects from apps matching cluster+namespace scopes.
// Empty scopes are treated as "all".
func (idx *AppIndex) ScopedProjects(apps []App, clusterScope, nsScope map[string]bool) []string {
	if idx == nil {
		return nil
	}
	if len(clusterScope) == 0 && len(nsScope) == 0 {
		return idx.Projects
	}

	// Build a set of in-scope app indices using bitwise intersection
	inScope := idx.scopeFilter(clusterScope, nsScope, nil, nil)

	seen := make(map[string]bool)
	for _, i := range inScope {
		if i < len(apps) {
			prj := ""
			if apps[i].Project != nil {
				prj = *apps[i].Project
			}
			if prj != "" {
				seen[prj] = true
			}
		}
	}
	return sortedKeys(seen)
}

// ScopedApps returns the apps matching all active scope filters, preserving order.
func (idx *AppIndex) ScopedApps(apps []App, sel *SelectionState) []App {
	if idx == nil {
		return apps
	}
	hasClusters := len(sel.ScopeClusters) > 0
	hasNs := len(sel.ScopeNamespaces) > 0
	hasProjects := len(sel.ScopeProjects) > 0
	hasAppSets := len(sel.ScopeApplicationSets) > 0

	if !hasClusters && !hasNs && !hasProjects && !hasAppSets {
		return apps
	}

	indices := idx.scopeFilter(sel.ScopeClusters, sel.ScopeNamespaces, sel.ScopeProjects, sel.ScopeApplicationSets)
	result := make([]App, 0, len(indices))
	for _, i := range indices {
		if i < len(apps) {
			result = append(result, apps[i])
		}
	}
	return result
}

// scopeFilter returns ordered app indices matching all non-empty scope filters.
func (idx *AppIndex) scopeFilter(clusterScope, nsScope, projScope, appsetScope map[string]bool) []int {
	// Start with all indices as a bitset
	bits := make([]bool, idx.Total)
	for i := range bits {
		bits[i] = true
	}

	if len(clusterScope) > 0 {
		match := make([]bool, idx.Total)
		for cl, ok := range clusterScope {
			if ok {
				for _, i := range idx.ByCluster[cl] {
					match[i] = true
				}
			}
		}
		for i := range bits {
			bits[i] = bits[i] && match[i]
		}
	}

	if len(nsScope) > 0 {
		match := make([]bool, idx.Total)
		for ns, ok := range nsScope {
			if ok {
				for _, i := range idx.ByNamespace[ns] {
					match[i] = true
				}
			}
		}
		for i := range bits {
			bits[i] = bits[i] && match[i]
		}
	}

	if len(projScope) > 0 {
		match := make([]bool, idx.Total)
		for prj, ok := range projScope {
			if ok {
				for _, i := range idx.ByProject[prj] {
					match[i] = true
				}
			}
		}
		for i := range bits {
			bits[i] = bits[i] && match[i]
		}
	}

	if len(appsetScope) > 0 {
		match := make([]bool, idx.Total)
		for as, ok := range appsetScope {
			if ok {
				for _, i := range idx.ByApplicationSet[as] {
					match[i] = true
				}
			}
		}
		for i := range bits {
			bits[i] = bits[i] && match[i]
		}
	}

	result := make([]int, 0)
	for i, ok := range bits {
		if ok {
			result = append(result, i)
		}
	}
	return result
}
