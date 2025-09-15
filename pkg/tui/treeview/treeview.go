package treeview

import (
    "fmt"
    "sort"
    "strings"

    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/charmbracelet/lipgloss/v2"
    "github.com/darksworm/argonaut/pkg/api"
)

// TreeView wraps a simple interactive tree for displaying ArgoCD resource trees.
// It intentionally keeps state minimal and integrates with Bubble Tea.
type TreeView struct {
    Model       tea.Model // kept to satisfy requested shape; this instance is itself a tea.Model
    SelectedUID string

    width  int
    height int

    nodesByUID map[string]*treeNode
    roots      []*treeNode
    expanded   map[string]bool
    order      []*treeNode // visible nodes in DFS order based on expanded
    selIdx     int

    appName   string
    appHealth string
    appSync   string
}

type treeNode struct {
    uid       string
    kind      string
    name      string
    namespace string
    status    string
    health    string
    parent    *treeNode
    children  []*treeNode
}

// Color styles consistent with existing TUI
var (
    colorGreen  = lipgloss.Color("10")
    colorYellow = lipgloss.Color("11")
    colorRed    = lipgloss.Color("9")
    colorGray   = lipgloss.Color("8")
    colorWhite  = lipgloss.Color("15")
    selectBG    = lipgloss.Color("13")
)

func statusStyle(s string) lipgloss.Style {
    switch strings.ToLower(s) {
    case "healthy", "running", "synced":
        return lipgloss.NewStyle().Foreground(colorGreen)
    case "progressing", "pending":
        return lipgloss.NewStyle().Foreground(colorYellow)
    case "degraded", "error", "crashloop":
        return lipgloss.NewStyle().Foreground(colorRed)
    default:
        return lipgloss.NewStyle().Foreground(colorGray)
    }
}

// NewTreeView creates a new tree view instance
func NewTreeView(width, height int) *TreeView {
    tv := &TreeView{
        width:      width,
        height:     height,
        nodesByUID: make(map[string]*treeNode),
        expanded:   make(map[string]bool),
        selIdx:     0,
    }
    tv.Model = tv // self
    return tv
}

// Init implements tea.Model; no async startup required
func (v *TreeView) Init() tea.Cmd { return nil }

// SetData converts api.ResourceTree to internal nodes and builds adjacency
func (v *TreeView) SetData(tree *api.ResourceTree) {
    v.nodesByUID = make(map[string]*treeNode)
    v.roots = nil
    v.expanded = make(map[string]bool)
    v.order = nil
    v.selIdx = 0
    v.SelectedUID = ""

    // First pass: create nodes
    for _, n := range tree.Nodes {
        ns := ""
        if n.Namespace != nil {
            ns = *n.Namespace
        }
        health := ""
        if n.Health != nil && n.Health.Status != nil {
            health = *n.Health.Status
        }
        tn := &treeNode{
            uid:    n.UID,
            kind:   n.Kind,
            name:   n.Name,
            status: n.Status,
            health: health,
            namespace: ns,
        }
        v.nodesByUID[n.UID] = tn
    }

    // Second pass: set parents/children (use first matching ParentRef uid if present)
    // Some resources may have multiple ParentRefs; attach to each parent for visibility.
    for _, n := range tree.Nodes {
        child := v.nodesByUID[n.UID]
        if len(n.ParentRefs) == 0 {
            continue
        }
        for _, pref := range n.ParentRefs {
            if p, ok := v.nodesByUID[pref.UID]; ok {
                child.parent = p // last parent wins for back-navigation
                p.children = append(p.children, child)
            }
        }
    }

    // Roots = nodes with no parent in the set (temporary, will attach under app root)
    tempRoots := make([]*treeNode, 0)
    for _, node := range v.nodesByUID {
        if node.parent == nil {
            tempRoots = append(tempRoots, node)
        }
    }
    // Sort roots and children by kind/name for stable display
    sortNodes := func(list []*treeNode) {
        sort.Slice(list, func(i, j int) bool {
            if list[i].kind == list[j].kind {
                return list[i].name < list[j].name
            }
            return list[i].kind < list[j].kind
        })
    }
    sortNodes(tempRoots)
    for _, n := range v.nodesByUID {
        if len(n.children) > 0 {
            sortNodes(n.children)
        }
    }

    // Create synthetic application root and attach temp roots under it
    root := &treeNode{
        uid:    "__app_root__",
        kind:   "Application",
        name:   v.appName,
        status: v.appHealth,
        health: v.appHealth,
    }
    for _, r := range tempRoots {
        r.parent = root
        root.children = append(root.children, r)
    }
    v.nodesByUID[root.uid] = root
    v.roots = []*treeNode{root}

    // Expand everything by default (root and all descendants)
    for uid := range v.nodesByUID {
        v.expanded[uid] = true
    }
    v.rebuildOrder()
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
            if v.selIdx > 0 { v.selIdx-- }
        case "down", "j":
            if v.selIdx < len(v.order)-1 { v.selIdx++ }
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
                    if pidx >= 0 { v.selIdx = pidx }
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
        if x == n { return i }
    }
    return -1
}

func (v *TreeView) View() string {
    if len(v.order) == 0 {
        return "(no resources)"
    }
    // Render visible nodes with ASCII tree lines
    var b strings.Builder
    // Precompute depths and last-child flags for drawing
    parentMap := make(map[*treeNode]*treeNode)
    for _, n := range v.nodesByUID { for _, c := range n.children { parentMap[c] = n } }

    for i, n := range v.order {
        // Determine depth by walking parents
        depth := 0
        p := n.parent
        for p != nil { depth++; p = p.parent }
        // Determine prefix
        var prefixParts []string
        // Build ancestry stack to know if ancestor is last child
        stack := make([]*treeNode, 0)
        pp := n.parent
        for pp != nil { stack = append(stack, pp); pp = pp.parent }
        // reverse stack
        for l, r := 0, len(stack)-1; l < r; l, r = l+1, r-1 { stack[l], stack[r] = stack[r], stack[l] }
        for _, anc := range stack {
            // Skip adding a trunk for the synthetic root (no parent)
            if anc.parent == nil { continue }
            // is anc last among its siblings?
            last := false
            siblings := anc.parent.children
            if len(siblings) > 0 && siblings[len(siblings)-1] == anc {
                last = true
            }
            if last {
                prefixParts = append(prefixParts, "    ")
            } else {
                prefixParts = append(prefixParts, "│   ")
            }
        }
        // For this node, determine connector relative to its parent
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

        // Disclosure indicator for nodes with children
        disc := ""
        if len(n.children) > 0 {
            if v.expanded[n.uid] {
                disc = "▾ "
            } else {
                disc = "▸ "
            }
        }

        // Color non-bracket parts (tree lines, disclosure, kind) as bright white for clarity
        prefixStyled := lipgloss.NewStyle().Foreground(colorWhite).Render(prefix + disc)
        // Default line composition without selection background
        label := v.renderLabel(n)
        line := prefixStyled + label

        // If collapsed, hint how many items are hidden
        if len(n.children) > 0 && !v.expanded[n.uid] {
            hidden := countDescendants(n)
            if hidden > 0 {
                hint := lipgloss.NewStyle().Foreground(colorGray).Render(fmt.Sprintf(" (+%d)", hidden))
                line += hint
            }
        }
        if i == v.selIdx {
            // Rebuild the line so that the selection background covers the entire row,
            // including the kind text and the bracketed name/status segments.
            name := n.name
            if n.namespace != "" {
                name = fmt.Sprintf("%s/%s", n.namespace, n.name)
            }
            status := n.health
            if status == "" { status = n.status }

            ps := lipgloss.NewStyle().Foreground(colorWhite).Background(selectBG).Render(prefix + disc)
            ks := lipgloss.NewStyle().Foreground(colorWhite).Background(selectBG).Render(n.kind)
            ns := lipgloss.NewStyle().Foreground(colorGray).Background(selectBG).Render("[" + name + "]")
            st := statusStyle(status).Background(selectBG).Render(fmt.Sprintf("(%s)", status))
            line = ps + ks + " " + ns + " " + st
            line = padRight(line, v.innerWidth())
        }
        b.WriteString(line)
        if i < len(v.order)-1 { b.WriteString("\n") }
    }
    return b.String()
}

func (v *TreeView) renderLabel(n *treeNode) string {
    name := n.name
    if n.namespace != "" {
        name = fmt.Sprintf("%s/%s", n.namespace, n.name)
    }
    status := n.health
    if status == "" { status = n.status }
    st := statusStyle(status).Render(fmt.Sprintf("(%s)", status))
    // Only the bracketed name should be gray/dim
    nameStyled := lipgloss.NewStyle().Foreground(colorGray).Render("[" + name + "]")
    kindStyled := lipgloss.NewStyle().Foreground(colorWhite).Render(n.kind)
    return fmt.Sprintf("%s %s %s", kindStyled, nameStyled, st)
}

func (v *TreeView) innerWidth() int {
    if v.width <= 2 { return v.width }
    return v.width - 2
}

func (v *TreeView) SetSize(width, height int) {
    v.width, v.height = width, height
}

func max(a, b int) int { if a > b { return a }; return b }

// Expose selected index for integration (optional)
func (v *TreeView) SelectedIndex() int { return v.selIdx }

// VisibleCount returns the number of currently visible nodes in DFS order.
func (v *TreeView) VisibleCount() int { return len(v.order) }

func padRight(s string, width int) string {
    w := lipgloss.Width(s)
    if w >= width { return s }
    return s + strings.Repeat(" ", width-w)
}

// SetAppMeta sets the application metadata used for the synthetic top-level node
func (v *TreeView) SetAppMeta(name, health, sync string) {
    v.appName = name
    v.appHealth = health
    v.appSync = sync
}

// countDescendants returns the number of nodes under n (deep)
func countDescendants(n *treeNode) int {
    if n == nil || len(n.children) == 0 { return 0 }
    total := 0
    for _, c := range n.children {
        total++
        total += countDescendants(c)
    }
    return total
}
