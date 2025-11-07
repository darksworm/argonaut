package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

func (m *Model) renderBanner() string {
	// If the terminal is short, collapse the header into 1–2 lines
	if m.state.Terminal.Rows <= 22 {
		return m.renderCompactBanner()
	}

	isNarrow := m.state.Terminal.Cols <= 100
	if isNarrow {
		// Float the small badge to the right of the first context line to save vertical space.
		ctx := m.renderContextBlock(true)
		ctxLines := strings.Split(ctx, "\n")
		first := ""
		rest := ""
		if len(ctxLines) > 0 {
			first = ctxLines[0]
			if len(ctxLines) > 1 {
				rest = strings.Join(ctxLines[1:], "\n")
			}
		}
		total := max(0, m.state.Terminal.Cols-2)
		// Width-based decision to show app version in badge
		withVersion := m.state.Terminal.Cols >= 72
		top := joinWithRightAlignment(first, m.renderSmallBadge(false, withVersion)+" ", total)
		if rest != "" {
			return top + "\n" + rest
		}
		return top
	}

	left := m.renderContextBlock(false)
	right := m.renderAsciiLogo()
	leftLines := strings.Count(left, "\n") + 1
	rightLines := strings.Count(right, "\n") + 1
	if leftLines < rightLines {
		left = strings.Repeat("\n", rightLines-leftLines) + left
	}
	if rightLines < leftLines {
		right = strings.Repeat("\n", leftLines-rightLines) + right
	}
	total := max(0, m.state.Terminal.Cols-2)
	return joinWithRightAlignment(left, right, total)
}

// renderSmallBadge renders the compact badge used in narrow terminals.
func (m *Model) renderSmallBadge(grayscale bool, withVersion bool) string {
	st := lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)
	if grayscale {
		st = st.Background(lipgloss.Color("243")).Foreground(ensureContrastingForeground(lipgloss.Color("243"), lipgloss.Color("16")))
	} else {
		st = st.Background(cyanBright).Foreground(textOnInfo)
	}
	text := "Argonaut"
	if withVersion {
		text += " " + appVersion
	}
	return st.Render(text)
}

// renderCompactBanner produces a 1–2 line banner optimized for low terminal height.
// Right-aligned shows the small badge; left-aligned shows a breadcrumb
// (ctx > cls > ns > proj). If it doesn't fit on 1 line beside the badge, the
// breadcrumb wraps to a second line; if it still doesn't fit, the badge is
// hidden; then the breadcrumb drops context, then cluster, then namespace
// until it fits.
func (m *Model) renderCompactBanner() string {
	total := max(0, m.state.Terminal.Cols-2)

	host := "—"
	if m.state.Server != nil {
		host = hostFromURL(m.state.Server.BaseURL)
	}
	cls := scopeToText(m.state.Selections.ScopeClusters)
	ns := scopeToText(m.state.Selections.ScopeNamespaces)
	pr := scopeToText(m.state.Selections.ScopeProjects)

	parts := []string{host, cls, ns, pr}
	// Build breadcrumb tokens with dim separators
	sep := " " + lipgloss.NewStyle().Foreground(dimColor).Render(">") + " "
	joinParts := func(items []string) string {
		// drop empty/placeholder
		xs := make([]string, 0, len(items))
		for _, s := range items {
			if strings.TrimSpace(s) != "" && s != "—" {
				xs = append(xs, s)
			}
		}
		if len(xs) == 0 {
			xs = []string{"—"}
		}
		return strings.Join(xs, sep)
	}

	// Tiny width: don't show version in badge
	withVersion := m.state.Terminal.Cols >= 72
	badge := m.renderSmallBadge(false, withVersion)
	rb := badge
	if rb != "" {
		rb += " "
	}

	// Try to fit full breadcrumb alongside badge on line 1
	line1 := joinParts(parts)
	// Always add one space of left padding to align with content box
	left1 := " " + line1
	if lipgloss.Width(left1)+lipgloss.Width(rb) <= total {
		return joinWithRightAlignment(left1, rb, total)
	}

	// If doesn't fit, move some tokens to line 2 (keep badge on line 1)
	// Try host+cls on line 1; ns+proj on line 2
	left1 = " " + joinParts(parts[:2])
	left2 := " " + joinParts(parts[2:])
	if lipgloss.Width(left1)+lipgloss.Width(rb) <= total && lipgloss.Width(left2) <= total {
		top := joinWithRightAlignment(left1, rb, total)
		bottom := joinWithRightAlignment(left2, "", total)
		return top + "\n" + bottom
	}

	// Drop the badge and try full breadcrumb on two lines
	rb = ""
	left1 = " " + joinParts(parts[:2])
	left2 = " " + joinParts(parts[2:])
	if lipgloss.Width(left1) <= total && lipgloss.Width(left2) <= total {
		return left1 + "\n" + left2
	}

	// Drop context, then cluster, then namespace until it fits in two lines
	dropOrders := [][]int{{0}, {1}, {2}}
	for _, toDrop := range dropOrders {
		w := make([]string, 0, len(parts))
		drop := make(map[int]bool)
		for _, di := range toDrop {
			drop[di] = true
		}
		for i, s := range parts {
			if !drop[i] {
				w = append(w, s)
			}
		}
		left1 = " " + joinParts(w[:min(2, len(w))])
		if len(w) > 2 {
			left2 = " " + joinParts(w[2:])
		} else {
			left2 = ""
		}
		if lipgloss.Width(left1) <= total && (left2 == "" || lipgloss.Width(left2) <= total) {
			if left2 == "" {
				return left1
			}
			return left1 + "\n" + left2
		}
	}
	// Fallback: just show whatever fits on one line
	return clipAnsiToWidth(" "+joinParts(parts), total)
}

func (m *Model) renderContextBlock(isNarrow bool) string {
	if m.state.Server == nil {
		return ""
	}
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

func (m *Model) renderAsciiLogo() string {
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
	if len(set) == 0 {
		return "—"
	}
	vals := make([]string, 0, len(set))
	for k := range set {
		vals = append(vals, k)
	}
	sortStrings(vals)
	return strings.Join(vals, ",")
}

func hostFromURL(s string) string {
	if s == "" {
		return "—"
	}
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		return u.Host
	}
	return s
}

func joinWithRightAlignment(left, right string, totalWidth int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	n := len(leftLines)
	if len(rightLines) > n {
		n = len(rightLines)
	}
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
		if filler < 1 {
			filler = 1
		}
		out = append(out, l+strings.Repeat(" ", filler)+r)
	}
	return strings.Join(out, "\n")
}
