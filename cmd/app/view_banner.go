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
		top := joinWithRightAlignment(first, m.renderSmallBadge(false), total)
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

// renderCompactBanner produces a 1–2 line banner optimized for low terminal height.
// Right-aligned it shows: ctx/cls/ns/proj details and the small badge (logo).
// If details don't fit on one line beside the badge, they wrap to a second line
// while the badge remains on the first line.
func (m *Model) renderCompactBanner() string {
	total := max(0, m.state.Terminal.Cols-2)

	// Build compact details tokens
	host := "—"
	if m.state.Server != nil {
		host = hostFromURL(m.state.Server.BaseURL)
	}
	cls := scopeToText(m.state.Selections.ScopeClusters)
	ns := scopeToText(m.state.Selections.ScopeNamespaces)
	pr := scopeToText(m.state.Selections.ScopeProjects)

	lbl := lipgloss.NewStyle().Foreground(whiteBright)
	val := lipgloss.NewStyle().Foreground(cyanBright)

	// tokens like: "ctx: host", "cls: ...", etc.
	tokens := []string{
		lbl.Render("ctx:") + " " + val.Render(host),
		lbl.Render("cls:") + " " + val.Render(cls),
		lbl.Render("ns:") + " " + val.Render(ns),
		lbl.Render("proj:") + " " + val.Render(pr),
	}

	badge := m.renderSmallBadge(false)
	badgeW := lipgloss.Width(badge)
	sep := "  "

	// Fill as many tokens as fit on the first line next to the badge
	avail := total - badgeW - 1
	if avail < 10 {
		avail = total // if too tight, let tokens use full width; joinWithRightAlignment will push badge to edge
	}
	var line1Tokens, line2Tokens []string
	widthSoFar := 0
	for i, t := range tokens {
		tw := lipgloss.Width(t)
		extra := 0
		if i > 0 {
			extra = lipgloss.Width(sep)
		}
		if widthSoFar+extra+tw <= avail || len(line1Tokens) == 0 {
			if i > 0 {
				widthSoFar += extra
			}
			line1Tokens = append(line1Tokens, t)
			widthSoFar += tw
		} else {
			line2Tokens = append(line2Tokens, t)
		}
	}

	right1 := strings.Join(line1Tokens, sep)
	if right1 != "" {
		right1 += " "
	}
	right1 += badge

	top := joinWithRightAlignment("", right1, total)
	if len(line2Tokens) == 0 {
		return top
	}
	right2 := strings.Join(line2Tokens, sep)
	bottom := joinWithRightAlignment("", right2, total)
	return top + "\n" + bottom
}

// renderSmallBadge renders the compact badge used in narrow terminals.
func (m *Model) renderSmallBadge(grayscale bool) string {
	st := lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)
	if grayscale {
		st = st.Background(shadeBG).Foreground(whiteBright)
	} else {
		st = st.Background(cyanBright).Foreground(whiteBright)
	}
	return st.Render("Argonaut " + appVersion)
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
