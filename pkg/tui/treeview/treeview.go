package treeview

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/api"
	model "github.com/darksworm/argonaut/pkg/model"
	pkgsort "github.com/darksworm/argonaut/pkg/sort"
	"github.com/darksworm/argonaut/pkg/theme"
)

// TreeView wraps a simple interactive tree for displaying ArgoCD resource trees.
// It intentionally keeps state minimal and integrates with Bubble Tea.
type TreeView struct {
	Model       tea.Model // kept to satisfy requested shape; this instance is itself a tea.Model
	SelectedUID string

	width  int
	height int

	nodesByUID map[string]*treeNode
	// Per-app bookkeeping so we can upsert multiple apps into a single view
	nodesByApp map[string][]string // appName -> list of node keys
	rootByApp  map[string]*treeNode
	roots      []*treeNode
	expanded   map[string]bool
	order      []*treeNode // visible nodes in DFS order based on expanded
	selIdx     int

	// Deprecated single-app fields; kept for compatibility in legacy SetData path
	appName   string
	appHealth string
	appSync   string
	// Multi-app metadata for synthetic roots
	appMeta map[string]struct{ health, sync string }

	// Theme colors
	palette theme.Palette

	// Filter/search state
	filterQuery  string // Current search query (empty = no filter)
	matchIndices []int  // Indices in 'order' that match the query
	currentMatch int    // Current position in matchIndices (for n/N navigation)

	// Multi-selection state for resource deletion
	selectedUIDs map[string]bool // UIDs of selected resources

	// Desaturate mode: when true, only highlight selected items (not cursor),
	// and scope highlights to just the resource text (not tree prefixes or padding)
	desaturateMode bool

	// Flash mode: when true, all rows are highlighted with success color (refresh feedback)
	flashAll bool

	// Sort configuration for ordering siblings in the tree
	sortConfig *model.SortConfig
}

// ResourceSelection represents a selected resource for deletion
type ResourceSelection struct {
	AppName   string
	Group     string
	Version   string
	Kind      string
	Namespace string
	Name      string
	Status    string // Sync status (e.g., "Synced", "OutOfSync", "Missing")
	Health    string // Health status (e.g., "Healthy", "Degraded", "Missing")
}

// IsMissing returns true if the resource has Missing status or health
func (r ResourceSelection) IsMissing() bool {
	status := strings.TrimSpace(strings.ToLower(r.Status))
	health := strings.TrimSpace(strings.ToLower(r.Health))
	return status == "missing" || health == "missing"
}

type treeNode struct {
	uid       string
	group     string
	version   string
	kind      string
	name      string
	namespace string
	status    string
	health    string
	parent    *treeNode
	children  []*treeNode
}

// SortKey satisfies pkgsort.Sortable.
func (n *treeNode) SortKey() model.SortKey {
	return model.SortKey{Health: n.health, Sync: n.status, Kind: n.kind, Name: n.name}
}

// sortNodeChildren sorts a sibling list using the current sort config via pkgsort.Sort.
// A nil sortConfig resolves to the default (kind, name) ascending order.
func (v *TreeView) sortNodeChildren(list []*treeNode) {
	cfg := model.SortConfig{Field: model.SortFieldName, Direction: model.SortAsc}
	if v.sortConfig != nil {
		cfg = *v.sortConfig
	}
	pkgsort.Sort(list, cfg)
}

// SetSort applies a sort configuration to the tree view, re-sorting all sibling groups.
func (v *TreeView) SetSort(config model.SortConfig) {
	v.sortConfig = &config
	v.sortNodeChildren(v.roots)
	for _, node := range v.nodesByUID {
		if len(node.children) > 0 {
			v.sortNodeChildren(node.children)
		}
	}
	v.rebuildOrder()
}

// statusStyle returns a lipgloss style for the given status using theme colors
func (v *TreeView) statusStyle(s string) lipgloss.Style {
	switch strings.ToLower(s) {
	case "healthy", "running", "synced":
		return lipgloss.NewStyle().Foreground(v.palette.Success)
	case "progressing", "pending":
		return lipgloss.NewStyle().Foreground(v.palette.Progress)
	case "degraded", "error", "crashloop":
		return lipgloss.NewStyle().Foreground(v.palette.Danger)
	default:
		return lipgloss.NewStyle().Foreground(v.palette.Unknown)
	}
}

// NewTreeView creates a new tree view instance
func NewTreeView(width, height int) *TreeView {
	tv := &TreeView{
		width:        width,
		height:       height,
		nodesByUID:   make(map[string]*treeNode),
		nodesByApp:   make(map[string][]string),
		rootByApp:    make(map[string]*treeNode),
		expanded:     make(map[string]bool),
		selIdx:       0,
		appMeta:      make(map[string]struct{ health, sync string }),
		palette:      theme.Default(), // Start with default theme
		selectedUIDs: make(map[string]bool),
	}
	tv.Model = tv // self
	return tv
}

// Init implements tea.Model; no async startup required
func (v *TreeView) Init() tea.Cmd { return nil }

// ApplyTheme updates the tree view's color palette
func (v *TreeView) ApplyTheme(palette theme.Palette) {
	v.palette = palette
}

// SetData converts api.ResourceTree to internal nodes and builds adjacency
func (v *TreeView) SetData(tree *api.ResourceTree) {
	// Legacy single-app path: reset state and insert once under v.appName
	v.nodesByUID = make(map[string]*treeNode)
	v.nodesByApp = make(map[string][]string)
	v.rootByApp = make(map[string]*treeNode)
	v.roots = nil
	v.expanded = make(map[string]bool)
	v.order = nil
	v.selIdx = 0
	v.SelectedUID = ""
	v.UpsertAppTree(v.appName, tree)
}

// UpsertAppTree replaces/adds a single application's tree under a synthetic root
func (v *TreeView) UpsertAppTree(appName string, tree *api.ResourceTree) {
	// Remove existing app entries
	if keys, ok := v.nodesByApp[appName]; ok {
		for _, k := range keys {
			delete(v.nodesByUID, k)
			delete(v.expanded, k)
		}
		delete(v.nodesByApp, appName)
	}
	if oldRoot, ok := v.rootByApp[appName]; ok {
		idx := -1
		for i, r := range v.roots {
			if r == oldRoot {
				idx = i
				break
			}
		}
		if idx >= 0 {
			v.roots = append(v.roots[:idx], v.roots[idx+1:]...)
		}
		delete(v.rootByApp, appName)
	}

	// Key scoping to avoid collisions across apps
	makeKey := func(uid string) string { return appName + "::" + uid }

	// First pass: build nodes for this app
	nodesLocal := make(map[string]*treeNode)
	appKeys := make([]string, 0, len(tree.Nodes)+1)
	for _, n := range tree.Nodes {
		ns := ""
		if n.Namespace != nil {
			ns = *n.Namespace
		}
		health := ""
		if n.Health != nil && n.Health.Status != nil {
			health = *n.Health.Status
		}
		key := makeKey(n.UID)
		tn := &treeNode{uid: key, group: n.Group, version: n.Version, kind: n.Kind, name: n.Name, status: n.Status, health: health, namespace: ns}
		v.nodesByUID[key] = tn
		nodesLocal[key] = tn
		appKeys = append(appKeys, key)
	}

	// Second pass: parent/child links in this app
	for _, n := range tree.Nodes {
		ckey := makeKey(n.UID)
		child := nodesLocal[ckey]
		if child == nil {
			continue
		}
		for _, pref := range n.ParentRefs {
			pkey := makeKey(pref.UID)
			if p, ok := nodesLocal[pkey]; ok {
				child.parent = p
				p.children = append(p.children, child)
			}
		}
	}

	// Collect roots for this app
	tempRoots := make([]*treeNode, 0)
	for _, node := range nodesLocal {
		if node.parent == nil {
			tempRoots = append(tempRoots, node)
		}
	}

	// Filter out Application nodes that would duplicate the synthetic root.
	// ArgoCD's resource tree stream sometimes includes the Application CR itself.
	// Promote any children of the filtered node to roots.
	filtered := make([]*treeNode, 0, len(tempRoots))
	for _, node := range tempRoots {
		if node.kind == "Application" && node.name == appName {
			for _, child := range node.children {
				child.parent = nil
				filtered = append(filtered, child)
			}
			delete(v.nodesByUID, node.uid)
			continue
		}
		filtered = append(filtered, node)
	}
	tempRoots = filtered

	// Sort roots and children according to current sort config
	v.sortNodeChildren(tempRoots)
	for _, n := range nodesLocal {
		if len(n.children) > 0 {
			v.sortNodeChildren(n.children)
		}
	}

	// Synthetic application root for this app
	meta := v.appMeta[appName]
	rootKey := makeKey("__app_root__")
	root := &treeNode{uid: rootKey, kind: "Application", name: appName, status: meta.sync, health: meta.health}
	for _, r := range tempRoots {
		r.parent = root
		root.children = append(root.children, r)
	}
	v.nodesByUID[rootKey] = root
	v.rootByApp[appName] = root
	v.roots = append(v.roots, root)
	appKeys = append(appKeys, rootKey)
	v.nodesByApp[appName] = appKeys

	// Expand newly added nodes
	for _, k := range appKeys {
		v.expanded[k] = true
	}

	// Stable root ordering by app name
	sort.SliceStable(v.roots, func(i, j int) bool { return v.roots[i].name < v.roots[j].name })
	v.rebuildOrder()
}

// SetResourceStatuses updates sync status for nodes matching the given resources.
// Resources are matched by (group, kind, namespace, name).
func (v *TreeView) SetResourceStatuses(appName string, resources []api.ResourceStatus) {
	// Build lookup by (group, kind, namespace, name)
	statusByKey := make(map[string]string)
	for _, r := range resources {
		key := fmt.Sprintf("%s/%s/%s/%s", r.Group, r.Kind, r.Namespace, r.Name)
		statusByKey[key] = r.Status
	}

	// Update nodes for this app
	if keys, ok := v.nodesByApp[appName]; ok {
		for _, nodeKey := range keys {
			if node, ok := v.nodesByUID[nodeKey]; ok {
				lookupKey := fmt.Sprintf("%s/%s/%s/%s", node.group, node.kind, node.namespace, node.name)
				if status, found := statusByKey[lookupKey]; found {
					node.status = status
				}
			}
		}
	}

	// If sorting by sync or health status, re-sort to reflect updated values
	if v.sortConfig != nil && (v.sortConfig.Field == model.SortFieldSync || v.sortConfig.Field == model.SortFieldHealth)   {
		v.sortNodeChildren(v.roots)
		for _, node := range v.nodesByUID {
			if len(node.children) > 0 {
				v.sortNodeChildren(node.children)
			}
		}
		v.rebuildOrder()
	}
}

func (v *TreeView) rebuildOrder() {
	v.order = v.order[:0]
	var walk func(n *treeNode, depth int)
	walk = func(n *treeNode, depth int) {
		v.order = append(v.order, n)
		if v.expanded[n.uid] {
			for _, c := range n.children {
				walk(c, depth+1)
			}
		}
	}
	for _, r := range v.roots {
		walk(r, 0)
	}
	// Clamp selection
	if v.selIdx >= len(v.order) {
		v.selIdx = max(0, len(v.order)-1)
	}
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		v.SelectedUID = v.order[v.selIdx].uid
	}
}

func (v *TreeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		switch m.String() {
		case "up", "k":
			if v.selIdx > 0 {
				v.selIdx--
			}
		case "down", "j":
			if v.selIdx < len(v.order)-1 {
				v.selIdx++
			}
		case "left", "h":
			if v.selIdx >= 0 && v.selIdx < len(v.order) {
				cur := v.order[v.selIdx]
				if v.expanded[cur.uid] && len(cur.children) > 0 {
					// collapse
					v.expanded[cur.uid] = false
					v.rebuildOrder()
				} else if cur.parent != nil {
					// go to parent
					// move selection up until we reach parent index
					pidx := v.indexOf(cur.parent)
					if pidx >= 0 {
						v.selIdx = pidx
					}
				}
			}
		case "right", "l", "enter":
			if v.selIdx >= 0 && v.selIdx < len(v.order) {
				cur := v.order[v.selIdx]
				if len(cur.children) > 0 {
					v.expanded[cur.uid] = true
					v.rebuildOrder()
				}
			}
		}
		if v.selIdx >= 0 && v.selIdx < len(v.order) {
			v.SelectedUID = v.order[v.selIdx].uid
		}
	case tea.WindowSizeMsg:
		v.width = m.Width
		v.height = m.Height
	}
	return v, nil
}

func (v *TreeView) indexOf(n *treeNode) int {
	for i, x := range v.order {
		if x == n {
			return i
		}
	}
	return -1
}

// Render returns the current string representation of the tree.
func (v *TreeView) Render() string {
    if len(v.order) == 0 {
        return "(no resources)"
    }
    var b strings.Builder
    parentMap := make(map[*treeNode]*treeNode)
    for _, n := range v.nodesByUID {
        for _, c := range n.children {
            parentMap[c] = n
        }
    }
    for i, n := range v.order {
        if n.parent == nil && i > 0 {
            b.WriteString("\n")
        }
        // Build ancestry stack
        stack := make([]*treeNode, 0)
        pp := n.parent
        for pp != nil {
            stack = append(stack, pp)
            pp = pp.parent
        }
        // reverse stack
        for l, r := 0, len(stack)-1; l < r; l, r = l+1, r-1 {
            stack[l], stack[r] = stack[r], stack[l]
        }
        var prefixParts []string
        for _, anc := range stack {
            if anc.parent == nil {
                continue
            }
            siblings := anc.parent.children
            last := len(siblings) > 0 && siblings[len(siblings)-1] == anc
            if last {
                prefixParts = append(prefixParts, "    ")
            } else {
                prefixParts = append(prefixParts, "│   ")
            }
        }
        conn := ""
        if n.parent != nil {
            siblings := n.parent.children
            if len(siblings) > 0 && siblings[len(siblings)-1] == n {
                conn = "└── "
            } else {
                conn = "├── "
            }
        }
        prefix := strings.Join(prefixParts, "") + conn
        disc := ""
        if len(n.children) > 0 && !v.expanded[n.uid] {
            disc = "▸ "
        }

        prefixStyled := lipgloss.NewStyle().Foreground(v.palette.Text).Render(prefix + disc)
        label := v.renderLabel(n)
        line := prefixStyled + label
        if len(n.children) > 0 && !v.expanded[n.uid] {
            hidden := countDescendants(n)
            if hidden > 0 {
                hint := lipgloss.NewStyle().Foreground(v.palette.Dim).Render(fmt.Sprintf(" (+%d)", hidden))
                line += hint
            }
        }
        isMatch := v.filterQuery != "" && v.isMatchIndex(i)
        isSelected := v.selectedUIDs[n.uid]
        isCursor := i == v.selIdx

        // Flash mode: all rows get success color background (refresh feedback)
        if v.flashAll {
            name := n.name
            if n.namespace != "" {
                name = fmt.Sprintf("%s/%s", n.namespace, n.name)
            }
            flashBG := v.palette.Success
            bgStyle := lipgloss.NewStyle().Background(flashBG)
            ps := lipgloss.NewStyle().Foreground(v.palette.Text).Background(flashBG).Render(prefix + disc)
            ks := lipgloss.NewStyle().Foreground(v.palette.Text).Background(flashBG).Render(n.kind)
            ns := lipgloss.NewStyle().Foreground(v.palette.DarkBG).Background(flashBG).Render("[" + name + "]")
            st := v.renderStatusPartWithBG(n, flashBG)
            sp := bgStyle.Render(" ")
            line = ps + ks + sp + ns + sp + st
            line = padRightWithBG(line, v.innerWidth(), flashBG)
        } else if v.desaturateMode {
        // In desaturate mode: only highlight selected items, with scoped highlighting
        // In normal mode: highlight both cursor and selected items with full-line highlighting
            // Desaturate mode: only selected items get highlighted, and only the resource text
            if isSelected {
                name := n.name
                if n.namespace != "" {
                    name = fmt.Sprintf("%s/%s", n.namespace, n.name)
                }
                rowBG := v.palette.SelectedBG
                bgStyle := lipgloss.NewStyle().Background(rowBG)
                // Prefix rendered WITHOUT background (will be dimmed by desaturateANSI)
                ps := lipgloss.NewStyle().Foreground(v.palette.Text).Render(prefix + disc)
                // Only resource text (kind, name, status) gets background
                ks := lipgloss.NewStyle().Foreground(v.palette.Text).Background(rowBG).Render(n.kind)
                ns := lipgloss.NewStyle().Foreground(v.palette.DarkBG).Background(rowBG).Render("[" + name + "]")
                st := v.renderStatusPartWithBG(n, rowBG)
                sp := bgStyle.Render(" ")
                line = ps + ks + sp + ns + sp + st
                // NO padRightWithBG - don't extend highlight to full width
            }
            // else: cursor-only or regular line - keep default rendering (no special background)
        } else {
            // Normal mode: existing behavior (cursor and selection both get full-line highlighting)
            if isCursor || isSelected {
                name := n.name
                if n.namespace != "" {
                    name = fmt.Sprintf("%s/%s", n.namespace, n.name)
                }
                // Determine background color based on state
                var rowBG color.Color
                if isCursor && isSelected {
                    // Cursor on selected: distinct color to show both states
                    rowBG = v.palette.CursorSelectedBG
                } else if isMatch {
                    // Search match (cursor or selected): use info color
                    rowBG = v.palette.Info
                } else {
                    // Plain cursor or plain selected: use standard selection background
                    rowBG = v.palette.SelectedBG
                }
                bgStyle := lipgloss.NewStyle().Background(rowBG)
                ps := lipgloss.NewStyle().Foreground(v.palette.Text).Background(rowBG).Render(prefix + disc)
                ks := lipgloss.NewStyle().Foreground(v.palette.Text).Background(rowBG).Render(n.kind)
                ns := lipgloss.NewStyle().Foreground(v.palette.DarkBG).Background(rowBG).Render("[" + name + "]")
                st := v.renderStatusPartWithBG(n, rowBG)
                sp := bgStyle.Render(" ")
                line = ps + ks + sp + ns + sp + st
                line = padRightWithBG(line, v.innerWidth(), rowBG)
            } else if isMatch {
                // Non-selected, non-cursor match: highlight with warning background
                name := n.name
                if n.namespace != "" {
                    name = fmt.Sprintf("%s/%s", n.namespace, n.name)
                }
                matchBG := v.palette.Warning
                bgStyle := lipgloss.NewStyle().Background(matchBG)
                ps := lipgloss.NewStyle().Foreground(v.palette.Text).Background(matchBG).Render(prefix + disc)
                ks := lipgloss.NewStyle().Foreground(v.palette.DarkBG).Background(matchBG).Render(n.kind)
                ns := lipgloss.NewStyle().Foreground(v.palette.DarkBG).Background(matchBG).Render("[" + name + "]")
                st := v.renderStatusPartWithBG(n, matchBG)
                sp := bgStyle.Render(" ")
                line = ps + ks + sp + ns + sp + st
                line = padRightWithBG(line, v.innerWidth(), matchBG)
            }
        }
        b.WriteString(line)
        if i < len(v.order)-1 {
            b.WriteString("\n")
        }
    }
    return b.String()
}

func (v *TreeView) View() tea.View {
    return tea.NewView(v.Render())
}

func (v *TreeView) renderLabel(n *treeNode) string {
	name := n.name
	if n.namespace != "" {
		name = fmt.Sprintf("%s/%s", n.namespace, n.name)
	}
	st := v.renderStatusPart(n)
	// Only the bracketed name should be gray/dim
	nameStyled := lipgloss.NewStyle().Foreground(v.palette.Dim).Render("[" + name + "]")
	kindStyled := lipgloss.NewStyle().Foreground(v.palette.Text).Render(n.kind)
	return fmt.Sprintf("%s %s %s", kindStyled, nameStyled, st)
}

// renderStatusPart returns styled status string showing health and/or sync status
func (v *TreeView) renderStatusPart(n *treeNode) string {
	health := n.health
	sync := n.status

	// Both present and different: show both
	if health != "" && sync != "" && !strings.EqualFold(health, sync) {
		healthStyled := v.statusStyle(health).Render(health)
		syncStyled := v.statusStyle(sync).Render(sync)
		return fmt.Sprintf("(%s, %s)", healthStyled, syncStyled)
	}
	// Only health present (or both same)
	if health != "" {
		return v.statusStyle(health).Render(fmt.Sprintf("(%s)", health))
	}
	// Only sync present
	if sync != "" {
		return v.statusStyle(sync).Render(fmt.Sprintf("(%s)", sync))
	}
	return ""
}

// renderStatusPartWithBG returns styled status string with a background color
func (v *TreeView) renderStatusPartWithBG(n *treeNode, bg color.Color) string {
	health := n.health
	sync := n.status

	bgStyle := lipgloss.NewStyle().Background(bg)

	// Both present and different: show both
	if health != "" && sync != "" && !strings.EqualFold(health, sync) {
		healthStyled := v.statusStyle(health).Background(bg).Render(health)
		syncStyled := v.statusStyle(sync).Background(bg).Render(sync)
		return bgStyle.Render("(") + healthStyled + bgStyle.Render(", ") + syncStyled + bgStyle.Render(")")
	}
	// Only health present (or both same)
	if health != "" {
		return bgStyle.Render("(") + v.statusStyle(health).Background(bg).Render(health) + bgStyle.Render(")")
	}
	// Only sync present
	if sync != "" {
		return bgStyle.Render("(") + v.statusStyle(sync).Background(bg).Render(sync) + bgStyle.Render(")")
	}
	return ""
}

func (v *TreeView) innerWidth() int {
	return v.width
}

func (v *TreeView) SetSize(width, height int) {
	v.width, v.height = width, height
}

// SetDesaturateMode enables or disables desaturate mode.
// When enabled, only selected items are highlighted (not the cursor),
// and highlights are scoped to just the resource text (not tree prefixes or padding).
func (v *TreeView) SetDesaturateMode(enabled bool) {
	v.desaturateMode = enabled
}

// SetFlashAll enables or disables flash mode.
// When enabled, all rows are highlighted with success color (refresh feedback).
func (v *TreeView) SetFlashAll(enabled bool) {
	v.flashAll = enabled
}

// Expose selected index for integration (optional)
func (v *TreeView) SelectedIndex() int { return v.selIdx }

// SetSelectedIndex sets the selected index directly with bounds checking.
// This is useful for external navigation (e.g., PageUp/PageDown from a ListNavigator).
func (v *TreeView) SetSelectedIndex(idx int) {
	if idx < 0 {
		idx = 0
	}
	if len(v.order) > 0 && idx >= len(v.order) {
		idx = len(v.order) - 1
	}
	v.selIdx = idx
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		v.SelectedUID = v.order[v.selIdx].uid
	}
}

// VisibleCount returns the number of currently visible nodes in DFS order.
func (v *TreeView) VisibleCount() int { return len(v.order) }

// SelectedLineIndex returns the index of the selected line in the rendered
// output, accounting for the blank separator lines inserted between app
// roots in View().
func (v *TreeView) SelectedLineIndex() int {
	if v.selIdx <= 0 || v.selIdx >= len(v.order) {
		if v.selIdx < 0 {
			return 0
		}
		return min(v.selIdx, max(0, len(v.order)-1))
	}
	gaps := 0
	for i := 1; i <= v.selIdx && i < len(v.order); i++ {
		if v.order[i].parent == nil {
			gaps++
		}
	}
	return v.selIdx + gaps
}

// VisibleLineCount returns the number of lines produced by View(), which is
// the number of visible nodes plus the number of blank separators (roots-1).
func (v *TreeView) VisibleLineCount() int {
	roots := 0
	for _, n := range v.order {
		if n.parent == nil {
			roots++
		}
	}
	if roots > 0 {
		roots--
	}
	return len(v.order) + roots
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func padRightWithBG(s string, width int, bg color.Color) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	padding := lipgloss.NewStyle().Background(bg).Render(strings.Repeat(" ", width-w))
	return s + padding
}

// SetAppMeta sets the application metadata used for the synthetic top-level node
func (v *TreeView) SetAppMeta(name, health, sync string) {
	if v.appMeta == nil {
		v.appMeta = make(map[string]struct{ health, sync string })
	}
	v.appMeta[name] = struct{ health, sync string }{health: health, sync: sync}
	// Keep legacy fields for single-app compatibility
	v.appName = name
	v.appHealth = health
	v.appSync = sync
}

// countDescendants returns the number of nodes under n (deep)
func countDescendants(n *treeNode) int {
	if n == nil || len(n.children) == 0 {
		return 0
	}
	total := 0
	for _, c := range n.children {
		total++
		total += countDescendants(c)
	}
	return total
}

// SetFilter sets the filter query and rebuilds match indices
func (v *TreeView) SetFilter(query string) {
	v.filterQuery = strings.TrimSpace(query)
	v.rebuildMatches()
}

// ClearFilter clears the filter state
func (v *TreeView) ClearFilter() {
	v.filterQuery = ""
	v.matchIndices = nil
	v.currentMatch = 0
}

// GetFilter returns the current filter query
func (v *TreeView) GetFilter() string {
	return v.filterQuery
}

// MatchCount returns the number of matching nodes
func (v *TreeView) MatchCount() int {
	return len(v.matchIndices)
}

// CurrentMatchIndex returns the 1-based index of the current match (for display)
func (v *TreeView) CurrentMatchIndex() int {
	if len(v.matchIndices) == 0 {
		return 0
	}
	return v.currentMatch + 1
}

// NextMatch moves to the next matching node and returns true if moved
func (v *TreeView) NextMatch() bool {
	if len(v.matchIndices) == 0 {
		return false
	}
	v.currentMatch = (v.currentMatch + 1) % len(v.matchIndices)
	v.selIdx = v.matchIndices[v.currentMatch]
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		v.SelectedUID = v.order[v.selIdx].uid
	}
	return true
}

// PrevMatch moves to the previous matching node and returns true if moved
func (v *TreeView) PrevMatch() bool {
	if len(v.matchIndices) == 0 {
		return false
	}
	v.currentMatch--
	if v.currentMatch < 0 {
		v.currentMatch = len(v.matchIndices) - 1
	}
	v.selIdx = v.matchIndices[v.currentMatch]
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		v.SelectedUID = v.order[v.selIdx].uid
	}
	return true
}

// JumpToFirstMatch moves to the first match if any exist
func (v *TreeView) JumpToFirstMatch() bool {
	if len(v.matchIndices) == 0 {
		return false
	}
	v.currentMatch = 0
	v.selIdx = v.matchIndices[0]
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		v.SelectedUID = v.order[v.selIdx].uid
	}
	return true
}

// rebuildMatches scans the order slice and finds indices of matching nodes
func (v *TreeView) rebuildMatches() {
	v.matchIndices = nil
	if v.filterQuery == "" {
		v.currentMatch = 0
		return
	}
	query := strings.ToLower(v.filterQuery)
	for i, node := range v.order {
		if v.nodeMatchesQuery(node, query) {
			v.matchIndices = append(v.matchIndices, i)
		}
	}
	// Reset current match position
	v.currentMatch = 0
}

// nodeMatchesQuery checks if a node matches the search query (case-insensitive)
func (v *TreeView) nodeMatchesQuery(n *treeNode, query string) bool {
	// Match against kind, name, namespace, status, health
	if strings.Contains(strings.ToLower(n.kind), query) {
		return true
	}
	if strings.Contains(strings.ToLower(n.name), query) {
		return true
	}
	if strings.Contains(strings.ToLower(n.namespace), query) {
		return true
	}
	if strings.Contains(strings.ToLower(n.status), query) {
		return true
	}
	if strings.Contains(strings.ToLower(n.health), query) {
		return true
	}
	return false
}

// isMatchIndex checks if the given index is in the matchIndices slice
func (v *TreeView) isMatchIndex(idx int) bool {
	for _, mi := range v.matchIndices {
		if mi == idx {
			return true
		}
	}
	return false
}

// SelectedResource returns the group, kind, namespace, and name of the currently selected resource.
// Returns ok=false if no resource is selected or the selection is invalid.
func (v *TreeView) SelectedResource() (group, kind, namespace, name string, ok bool) {
	if v.selIdx < 0 || v.selIdx >= len(v.order) {
		return "", "", "", "", false
	}
	node := v.order[v.selIdx]
	if node == nil {
		return "", "", "", "", false
	}
	return node.group, node.kind, node.namespace, node.name, true
}

// GetAppName returns the name of the application being displayed.
func (v *TreeView) GetAppName() string {
	return v.appName
}

// ToggleSelection toggles selection for the current resource.
// Application nodes and Missing resources cannot be selected.
// Returns true if selection was toggled, false if the resource cannot be selected.
func (v *TreeView) ToggleSelection() bool {
	if v.selIdx < 0 || v.selIdx >= len(v.order) {
		return false
	}
	node := v.order[v.selIdx]
	// Don't allow selecting synthetic Application root nodes
	if node.kind == "Application" {
		return false
	}
	// Don't allow selecting Missing resources (already deleted)
	if v.nodeIsMissing(node) {
		return false
	}
	if v.selectedUIDs == nil {
		v.selectedUIDs = make(map[string]bool)
	}
	if v.selectedUIDs[node.uid] {
		delete(v.selectedUIDs, node.uid)
	} else {
		v.selectedUIDs[node.uid] = true
	}
	return true
}

// CurrentResourceIsMissing returns true if the current resource has "Missing" status or health.
func (v *TreeView) CurrentResourceIsMissing() bool {
	if v.selIdx < 0 || v.selIdx >= len(v.order) {
		return false
	}
	return v.nodeIsMissing(v.order[v.selIdx])
}

// nodeIsMissing checks if a node has Missing status or health (case-insensitive, with whitespace trimming)
func (v *TreeView) nodeIsMissing(node *treeNode) bool {
	if node == nil {
		return false
	}
	status := strings.TrimSpace(strings.ToLower(node.status))
	health := strings.TrimSpace(strings.ToLower(node.health))
	return status == "missing" || health == "missing"
}

// GetSelectedResources returns all selected resources.
// If no resources are explicitly selected, returns the current resource (if not an Application).
func (v *TreeView) GetSelectedResources() []ResourceSelection {
	var result []ResourceSelection

	// If explicit selections exist, return those
	if len(v.selectedUIDs) > 0 {
		for uid := range v.selectedUIDs {
			if node, ok := v.nodesByUID[uid]; ok {
				// Extract app name from uid (format: "appName::uid")
				appName := v.appName
				if idx := strings.Index(uid, "::"); idx > 0 {
					appName = uid[:idx]
				}
				result = append(result, ResourceSelection{
					AppName:   appName,
					Group:     node.group,
					Version:   node.version,
					Kind:      node.kind,
					Namespace: node.namespace,
					Name:      node.name,
					Status:    node.status,
					Health:    node.health,
				})
			}
		}
		return result
	}

	// No explicit selection - return current resource if valid
	if v.selIdx >= 0 && v.selIdx < len(v.order) {
		node := v.order[v.selIdx]
		if node.kind != "Application" {
			appName := v.appName
			if idx := strings.Index(node.uid, "::"); idx > 0 {
				appName = node.uid[:idx]
			}
			result = append(result, ResourceSelection{
				AppName:   appName,
				Group:     node.group,
				Version:   node.version,
				Kind:      node.kind,
				Namespace: node.namespace,
				Name:      node.name,
				Status:    node.status,
				Health:    node.health,
			})
		}
	}
	return result
}

// ClearSelection clears all resource selections.
func (v *TreeView) ClearSelection() {
	v.selectedUIDs = make(map[string]bool)
}

// HasSelection returns true if any resources are explicitly selected.
func (v *TreeView) HasSelection() bool {
	return len(v.selectedUIDs) > 0
}

// SelectionCount returns the number of explicitly selected resources.
func (v *TreeView) SelectionCount() int {
	return len(v.selectedUIDs)
}

// IsSelected returns true if the resource at the given index is selected.
func (v *TreeView) IsSelected(idx int) bool {
	if idx < 0 || idx >= len(v.order) {
		return false
	}
	return v.selectedUIDs[v.order[idx].uid]
}
