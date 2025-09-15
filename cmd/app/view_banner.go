package main

import (
    "fmt"
    "net/url"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"
)

func (m Model) renderBanner() string {
    isNarrow := m.state.Terminal.Cols <= 100
    if isNarrow {
        badge := m.renderSmallBadge(false)
        var sections []string
        sections = append(sections, "")
        sections = append(sections, badge)
        sections = append(sections, "")
        ctx := m.renderContextBlock(true)
        sections = append(sections, ctx)
        sections = append(sections, "")
        return strings.Join(sections, "\n")
    }

    left := m.renderContextBlock(false)
    right := m.renderAsciiLogo()
    leftLines := strings.Count(left, "\n") + 1
    rightLines := strings.Count(right, "\n") + 1
    if leftLines < rightLines { left = strings.Repeat("\n", rightLines-leftLines) + left }
    if rightLines < leftLines { right = strings.Repeat("\n", leftLines-rightLines) + right }
    total := max(0, m.state.Terminal.Cols-2)
    return joinWithRightAlignment(left, right, total)
}

// renderSmallBadge renders the compact badge used in narrow terminals.
func (m Model) renderSmallBadge(grayscale bool) string {
    st := lipgloss.NewStyle().
        Bold(true).
        PaddingLeft(1).
        PaddingRight(1)
    if grayscale {
        st = st.Background(lipgloss.Color("243")).Foreground(lipgloss.Color("16"))
    } else {
        st = st.Background(cyanBright).Foreground(whiteBright)
    }
    return st.Render("Argonaut " + appVersion)
}

func (m Model) renderContextBlock(isNarrow bool) string {
    if m.state.Server == nil { return "" }
    label := lipgloss.NewStyle().Bold(true).Foreground(whiteBright)
    cyan := lipgloss.NewStyle().Foreground(cyanBright)
    green := lipgloss.NewStyle().Foreground(syncedColor)

    serverHost := hostFromURL(m.state.Server.BaseURL)
    clusterScope := scopeToText(m.state.Selections.ScopeClusters)
    namespaceScope := scopeToText(m.state.Selections.ScopeNamespaces)
    projectScope := scopeToText(m.state.Selections.ScopeProjects)

    var lines []string
    lines = append(lines, fmt.Sprintf("%s %s", label.Render("Context:"), cyan.Render(serverHost)))
    if clusterScope != "—" {
        lines = append(lines, fmt.Sprintf("%s %s", label.Render("Cluster:"), clusterScope))
    }
    if namespaceScope != "—" {
        lines = append(lines, fmt.Sprintf("%s %s", label.Render("Namespace:"), namespaceScope))
    }
    if projectScope != "—" {
        lines = append(lines, fmt.Sprintf("%s %s", label.Render("Project:"), projectScope))
    }
    if !isNarrow && m.state.APIVersion != "" {
        lines = append(lines, fmt.Sprintf("%s %s", label.Render("ArgoCD:"), green.Render(m.state.APIVersion)))
    }
    block := strings.Join(lines, "\n")
    return lipgloss.NewStyle().PaddingRight(2).Render(block)
}

func (m Model) renderAsciiLogo() string {
    cyan := lipgloss.NewStyle().Foreground(cyanBright)
    white := lipgloss.NewStyle().Foreground(whiteBright)
    dim := lipgloss.NewStyle().Foreground(dimColor)
    version := appVersion
    versionPadded := fmt.Sprintf("%13s", version)
    l1 := cyan.Render("   _____") + strings.Repeat(" ", 43) + white.Render(" __   ")
    l2 := cyan.Render("  /  _  \\_______  ____   ____") + white.Render("   ____ _____   __ ___/  |_ ")
    l3 := cyan.Render(" /  /_\\  \\_  __ \\/ ___\\ /  _ \\ ") + white.Render("/    \\\\__  \\ |  |  \\   __\\")
    l4 := cyan.Render(" /    |    \\  | \\/ /_/  >  <_> )  ") + white.Render(" |  \\/ __ \\|  |  /|  |  ")
    l5 := cyan.Render("\\____|__  /__|  \\___  / \\____/") + white.Render("|___|  (____  /____/ |__|  ")
    l6 := cyan.Render("        \\/     /_____/             ") + white.Render("\\/     \\/") + dim.Render(versionPadded)
    return strings.Join([]string{l1, l2, l3, l4, l5, l6}, "\n")
}

func scopeToText(set map[string]bool) string {
    if len(set) == 0 { return "—" }
    vals := make([]string, 0, len(set))
    for k := range set { vals = append(vals, k) }
    sortStrings(vals)
    return strings.Join(vals, ",")
}

func hostFromURL(s string) string {
    if s == "" { return "—" }
    if u, err := url.Parse(s); err == nil && u.Host != "" { return u.Host }
    return s
}

func joinWithRightAlignment(left, right string, totalWidth int) string {
    leftLines := strings.Split(left, "\n")
    rightLines := strings.Split(right, "\n")
    n := len(leftLines)
    if len(rightLines) > n { n = len(rightLines) }
    if len(leftLines) < n {
        pad := make([]string, n-len(leftLines))
        leftLines = append(pad, leftLines...)
    }
    if len(rightLines) < n {
        pad := make([]string, n-len(rightLines))
        rightLines = append(pad, rightLines...)
    }
    var out []string
    for i := 0; i < n; i++ {
        l := leftLines[i]
        r := rightLines[i]
        lw := lipgloss.Width(l)
        rw := lipgloss.Width(r)
        filler := totalWidth - lw - rw
        if filler < 1 { filler = 1 }
        out = append(out, l+strings.Repeat(" ", filler)+r)
    }
    return strings.Join(out, "\n")
}

