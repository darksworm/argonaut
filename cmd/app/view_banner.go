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
		// Add a one-space right margin after the badge to balance with left padding
		top := joinWithRightAlignment(first, m.renderSmallBadge(false, true)+" ", total)
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

	// Determine if we are in very small mode (tight width/height).
	tiny := m.state.Terminal.Rows <= 18 || m.state.Terminal.Cols <= 60
	badge := m.renderSmallBadge(false, !tiny) // hide version in tiny mode
	badgeW := lipgloss.Width(badge)
	sep := "  "

	// Helper to try fit tokens into 2 lines (first with badge on right)
	tryFit := func(tok []string, avail1 int, avail2 int) (l1, l2 []string, ok bool) {
		w := 0
		for i, t := range tok {
			tw := lipgloss.Width(t)
			add := tw
			if i > 0 {
				add += lipgloss.Width(sep)
			}
			if w+add <= avail1 || len(l1) == 0 { // ensure at least first token gets in
				if i > 0 {
					w += lipgloss.Width(sep)
				}
				l1 = append(l1, t)
				w += tw
			} else {
				l2 = append(l2, t)
			}
		}
		// Check second line fits as a whole (no clipping)
		line2 := strings.Join(l2, sep)
		if lipgloss.Width(line2) <= avail2 {
			return l1, l2, true
		}
		return l1, l2, false
	}

	// Start with breadcrumb tokens in order host > cls > ns > proj
	avail1 := total - badgeW - 1
	if avail1 < 8 {
		avail1 = total // if badge too wide, allow fill and we may drop it below
	}
	line1Tokens, line2Tokens, ok := tryFit(tokens, avail1, total)
	if !ok {
		// Drop the badge and try again
		badge = ""
		badgeW = 0
		avail1 = total
		line1Tokens, line2Tokens, ok = tryFit(tokens, avail1, total)
	}
	if !ok {
		// Drop tokens progressively: ctx (host), then cls, then ns
		dropOrder := []int{0, 1, 2}
		// Build working copy of tokens
		work := append([]string{}, tokens...)
		for _, di := range dropOrder {
			if di < len(work) {
				// remove element at di
				work = append(work[:di], work[di+1:]...)
				line1Tokens, line2Tokens, ok = tryFit(work, avail1, total)
				if ok {
					tokens = work
					break
				}
			}
		}
		if !ok {
			// As a last resort, keep only the last element (project) if any
			if len(work) > 0 {
				tokens = work[len(work)-1:]
			} else {
				tokens = nil
			}
			line1Tokens, line2Tokens, _ = tryFit(tokens, avail1, total)
		}
	}

	left1 := strings.Join(line1Tokens, sep)
	// Add one space padding on the left to align with the main content box
	// Add a right margin outside the badge so the colored badge doesn't touch the edge
	rb := badge
	if rb != "" {
		rb += " "
	}
	top := joinWithRightAlignment(" "+left1, rb, total)
	if len(line2Tokens) == 0 {
		return top
	}
	left2 := strings.Join(line2Tokens, sep)
	bottom := joinWithRightAlignment(" "+left2, "", total)
	return top + "\n" + bottom
}

// renderSmallBadge renders the compact badge used in narrow terminals.
func (m *Model) renderSmallBadge(grayscale bool, withVersion bool) string {
	st := lipgloss.NewStyle().
		Bold(true).
		PaddingLeft(1).
		PaddingRight(1)
	if grayscale {
		st = st.Background(shadeBG).Foreground(whiteBright)
	} else {
		st = st.Background(cyanBright).Foreground(whiteBright)
	}
	text := "Argonaut"
	if withVersion {
		text += " " + appVersion
	}
	return st.Render(text)
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
