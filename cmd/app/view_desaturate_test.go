package main

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
)

// Regression tests for desaturateANSI — see fix for the bug where lines
// containing any background-color escape were preserved wholesale,
// leaving popup-adjacent rows (e.g. the selected tree row's prefix and
// the status bar around the styled :upgrade/:changelog token)
// undimmed when a modal was overlaid.

func containsDimFG(s string) bool {
	// Match the dim-color foreground escape produced by lipgloss for
	// dimColor (palette index 8). Lipgloss may emit `\x1b[38;5;8m` or
	// `\x1b[90m` depending on profile; check both.
	return strings.Contains(s, "\x1b[38;5;8m") ||
		strings.Contains(s, "\x1b[90m")
}

func TestDesaturateANSI_PlainLineDimmed(t *testing.T) {
	out := desaturateANSI("hello world")
	if !containsDimFG(out) {
		t.Fatalf("expected plain line to be dimmed, got %q", out)
	}
	if !strings.Contains(out, "hello world") {
		t.Fatalf("expected text to be preserved, got %q", out)
	}
}

func TestDesaturateANSI_OnlyForegroundSegmentDimmed(t *testing.T) {
	// A line styled with only foreground color (no bg) should be dimmed
	// regardless of any embedded fg escapes.
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("warning")
	out := desaturateANSI(yellow)
	if !containsDimFG(out) {
		t.Fatalf("expected fg-only segment to be dimmed, got %q", out)
	}
	if strings.Contains(out, "\x1b[38;5;11m") || strings.Contains(out, "\x1b[33m") {
		t.Fatalf("expected original yellow fg to be stripped, got %q", out)
	}
}

func TestDesaturateANSI_BackgroundSegmentPreserved(t *testing.T) {
	bg := lipgloss.NewStyle().
		Background(lipgloss.Color("13")).
		Foreground(lipgloss.Color("0")).
		Render(":upgrade")
	out := desaturateANSI(bg)
	if !bgColorRE.MatchString(out) {
		t.Fatalf("expected bg-styled segment to retain its background, got %q", out)
	}
}

func TestDesaturateANSI_MixedSegmentsOnSameLine(t *testing.T) {
	// Simulates a status-bar-style line: plain text, then a styled
	// :upgrade token with a background, then more plain text. Before the
	// fix, the entire line was preserved unmodified because it contained
	// any bg color anywhere.
	plain := "Ready • "
	styled := lipgloss.NewStyle().
		Background(lipgloss.Color("13")).
		Foreground(lipgloss.Color("15")).
		Render(":upgrade")
	tail := " • 1/12"
	line := plain + styled + tail

	out := desaturateANSI(line)

	// The bg-styled :upgrade token must still carry a background.
	if !bgColorRE.MatchString(out) {
		t.Fatalf("expected styled :upgrade token to keep its background, got %q", out)
	}
	// The plain prefix and suffix must now be dim — meaning the dim
	// foreground escape appears in the output.
	if !containsDimFG(out) {
		t.Fatalf("expected non-bg portions to be dimmed, got %q", out)
	}
	// And the original textual content must still be intact.
	for _, want := range []string{"Ready", ":upgrade", "1/12"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q to survive desaturation, got %q", want, out)
		}
	}
}

func TestDesaturateANSI_TreeRowPrefixDimmedSelectionPreserved(t *testing.T) {
	// Mirrors the desaturate-mode tree row layout from
	// pkg/tui/treeview/treeview.go: a foreground-only prefix segment
	// followed by background-bearing segments for the kind/name/status.
	prefix := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render("  ├─ ")
	bg := lipgloss.NewStyle().Background(lipgloss.Color("13")).Foreground(lipgloss.Color("0"))
	kind := bg.Render("Pod")
	name := bg.Render(" [my-pod]")
	line := prefix + kind + name

	out := desaturateANSI(line)

	if !bgColorRE.MatchString(out) {
		t.Fatalf("expected selected row's bg highlight to survive, got %q", out)
	}
	if !containsDimFG(out) {
		t.Fatalf("expected non-bg prefix to be dimmed, got %q", out)
	}
	if !strings.Contains(out, "├─") || !strings.Contains(out, "[my-pod]") {
		t.Fatalf("expected text content to be preserved, got %q", out)
	}
}

func TestDesaturateANSI_PreservesNewlines(t *testing.T) {
	in := "alpha\nbeta\ngamma"
	out := desaturateANSI(in)
	if got := strings.Count(out, "\n"); got != 2 {
		t.Fatalf("expected 2 newlines preserved, got %d in %q", got, out)
	}
}

// TestRender_AppsList_SyncModal_DesaturatesNonSelected verifies the
// scenario the user reported: with a confirm-sync modal up and one app
// selected for sync, only the selected app keeps its bg highlight.
// Other rows — including the cursor row — must not retain a bright
// selection-style background, and per-cell status colors (Synced /
// Healthy) on inactive rows must be dimmed.
func TestRender_AppsList_SyncModal_DesaturatesNonSelected(t *testing.T) {
	m := buildTestModelWithApps(100, 30)
	// Cursor on app-a (index 0); app-b (index 1) is the one selected
	// for sync — mirrors the "test-prod selected, test-dev at cursor"
	// case the user reported.
	m.state.Navigation.SelectedIdx = 0
	m.state.Selections.SelectedApps = map[string]bool{"app-b": true}
	m.state.Mode = model.ModeConfirmSync

	if !m.willDesaturateBase() {
		t.Fatalf("expected willDesaturateBase to be true under ModeConfirmSync")
	}

	// Render the list view and apply the same desaturation pipeline the
	// real overlay path uses (view_layout.go calls desaturateANSI on
	// the assembled baseView).
	listOut := m.renderListView(10)
	desaturated := desaturateANSI(listOut)

	// Sanity: the cursor row (app-a) must NOT carry the selectedStyle
	// bg color (magentaBright = 13 / bright magenta) after desaturation.
	// We split the output into lines and inspect the line containing
	// "app-a" — it should have no bg color codes, since the cursor
	// highlight was suppressed.
	for _, line := range strings.Split(desaturated, "\n") {
		if !strings.Contains(line, "app-a") {
			continue
		}
		if bgColorRE.MatchString(line) {
			t.Fatalf("cursor row (app-a) should not retain bg under desaturating modal: %q", line)
		}
	}

	// The selected row (app-b) MUST still carry a bg highlight so the
	// user can see what's being synced.
	foundSelectedHighlight := false
	for _, line := range strings.Split(desaturated, "\n") {
		if strings.Contains(line, "app-b") && bgColorRE.MatchString(line) {
			foundSelectedHighlight = true
			break
		}
	}
	if !foundSelectedHighlight {
		t.Fatalf("selected row (app-b) should retain bg highlight; output=\n%s", desaturated)
	}

	// Inactive rows render Sync/Health with fg-only color. After
	// desaturation those segments must be dimmed — i.e. the original
	// status fg colors (green for Synced/Healthy, red for OutOfSync,
	// yellow for Progressing) must not appear on the cursor row.
	// Check that the line containing the unrelated cursor row (app-a)
	// no longer carries the green fg used for Synced/Healthy.
	for _, line := range strings.Split(desaturated, "\n") {
		if !strings.Contains(line, "app-a") {
			continue
		}
		// 256-color green (10) and ANSI bright green
		for _, fg := range []string{"\x1b[38;5;10m", "\x1b[92m"} {
			if strings.Contains(line, fg) {
				t.Fatalf("cursor row should not retain status fg %q after desaturation: %q", fg, line)
			}
		}
	}
}

// TestClipAnsiToWidth_ClosesOpenSGR is the regression for the "cursor
// highlight bleeds onto the next visual line" bug. When a styled tree
// row exceeds the panel width and gets clipped, the clipped portion can
// end mid-styled-run with the bg still active. Without an appended
// reset, the bg "leaks" into whatever the terminal renders after.
func TestClipAnsiToWidth_ClosesOpenSGR(t *testing.T) {
	bg := lipgloss.NewStyle().Background(lipgloss.Color("13")).Render("highlighted row content beyond panel width")
	clipped := clipAnsiToWidth(bg, 10)
	if !strings.HasSuffix(clipped, "\x1b[m") && !strings.HasSuffix(clipped, "\x1b[0m") {
		t.Fatalf("clipped output must end with an SGR reset to prevent style bleed; got %q", clipped)
	}
}

func TestClipAnsiToWidth_PlainStringUntouched(t *testing.T) {
	out := clipAnsiToWidth("hello world", 5)
	if out != "hello" {
		t.Fatalf("expected exact clip to %q, got %q", "hello", out)
	}
}

// TestRender_AppsList_SyncModal_NoColorLeaks_RealTheme is the
// integration test for the user-reported "(Healthy) is still green /
// header still yellow / random app stays bright" cluster of bugs,
// exercised through the FULL pipeline a real session uses: the nord
// theme is applied (so SGR codes are truecolor like in production),
// and the assertion runs against the post-canvas-compose output.
//
// Earlier versions of this test ran without applying any theme,
// which meant `dimColor` resolved to the bare ANSI 8 / `\x1b[90m`
// path and never exercised the truecolor (`\x1b[38;2;R;G;Bm`)
// branches — masking any regex-or-segment bug that only triggers on
// truecolor SGRs.
func TestRender_AppsList_SyncModal_NoColorLeaks_RealTheme(t *testing.T) {
	palette, ok := theme.Get("nord")
	if !ok {
		t.Skip("nord theme not available")
	}
	applyTheme(palette)
	// Restore default theme afterwards so we don't leak global
	// theme state into other tests in the package.
	t.Cleanup(func() { applyTheme(theme.Default()) })

	m := buildTestModelWithApps(100, 30)
	m.applyThemeToModel()
	m.state.Navigation.SelectedIdx = 0
	m.state.Selections.SelectedApps = map[string]bool{"app-b": true}
	m.state.UI.RefreshFlashApps = map[string]bool{"app-c": true}
	m.state.Mode = model.ModeConfirmSync

	// Run the full render: this includes desaturateANSI AND
	// composeOverlay (lipgloss canvas). If the canvas were to revert
	// dimmed runs to the original colors, we'd see saturated codes
	// here — that hypothesis was disproved (lipgloss preserves the
	// SGR we emit) but this test gates the full pipeline regardless.
	out := m.renderMainLayout()

	// nord palette saturated truecolor codes that must NOT survive
	// outside of explicit modal/badge/selected-row regions:
	//   Success #a3be8c  → 163;190;140 (green)
	//   Danger  #bf616a  → 191;97;106  (red)
	//   Warning #ebcb8b  → 235;203;139 (yellow)
	saturatedSGRs := []string{
		"38;2;163;190;140", // green fg
		"38;2;191;97;106",  // red fg
		"38;2;235;203;139", // yellow fg
	}

	for i, line := range strings.Split(out, "\n") {
		plain := stripANSI(line)
		if strings.Contains(plain, "app-b") {
			continue // selected sync target keeps its highlight
		}
		// Skip the modal body — it's allowed to be vivid.
		if strings.Contains(plain, "Sync") && strings.Contains(plain, "?") {
			continue
		}
		if strings.Contains(plain, "Confirm") || strings.Contains(plain, "[Sync]") {
			continue
		}
		// The "Argonaut dev" badge keeps its bg/fg styling on purpose.
		if strings.Contains(plain, "Argonaut") {
			continue
		}
		for _, sgr := range saturatedSGRs {
			if strings.Contains(line, sgr) {
				t.Errorf("line %d carries saturated truecolor fg %q under sync modal: %q", i, sgr, line)
			}
		}
	}
}

// TestSegmentHasBgColor_TruecolorFgFalsePositives is the regression
// for the regex false-positive that left "(Healthy)" / "(OutOfSync)"
// colored under modals on themes whose status colors happen to encode
// an RGB byte in the 40-47 / 100-107 range (which the old
// `bgColorRE` regex confused with basic-bg / bright-bg SGR).
//
// nord Danger        #bf616a → 191;97;106  (B=106  → 106m)
// solarized Danger   #dc322f → 220;50;47   (B=47   → 47m)
// monokai Success    #a6e22e → 166;226;46  (B=46   → 46m)
// tokyo-storm Success#9ece6a → 158;206;106 (B=106  → 106m)
//
// All of these are foreground SGRs and must be classified as
// "fg-only" so desaturateANSI dims them rather than preserving them.
func TestSegmentHasBgColor_TruecolorFgFalsePositives(t *testing.T) {
	fgOnly := []string{
		"\x1b[38;2;191;97;106m",
		"\x1b[38;2;220;50;47m",
		"\x1b[38;2;166;226;46m",
		"\x1b[38;2;158;206;106m",
		"\x1b[38;2;76;86;106m",
		"\x1b[1;38;2;163;190;140m",
		"\x1b[33m", "\x1b[1;33m", "\x1b[38;5;10m",
	}
	for _, c := range fgOnly {
		if segmentHasBgColor(c) {
			t.Errorf("expected fg-only SGR to NOT be classified as bg: %q", c)
		}
	}
	bgPresent := []string{
		"\x1b[40m", "\x1b[42m", "\x1b[47m",
		"\x1b[100m", "\x1b[105m", "\x1b[107m",
		"\x1b[48;5;235m",
		"\x1b[48;2;30;30;80m",
		"\x1b[97;48;2;30;30;80m",
		"\x1b[38;2;255;255;255;48;2;30;30;80m",
	}
	for _, c := range bgPresent {
		if !segmentHasBgColor(c) {
			t.Errorf("expected bg SGR to be classified as bg: %q", c)
		}
	}
}

// TestDesaturateANSI_AllPaletteColorsCollapseToDim is the
// exhaustive unit test: walk every shipped theme preset, render a
// string in every palette color as a plain foreground (no bg), run
// the result through desaturateANSI, and assert the surviving
// foreground SGR is identical to the dim foreground SGR for that
// theme. If any palette color escapes the dimmer (as nord Danger /
// monokai Success / solarized-light Danger did before the SGR-param
// parser fix), the test fails and pinpoints the offending
// preset+field.
func TestDesaturateANSI_AllPaletteColorsCollapseToDim(t *testing.T) {
	t.Cleanup(func() { applyTheme(theme.Default()) })

	// Pull the SGR run that wraps the rendered text: the sequence
	// from the first \x1b[ to its terminating m. Returns "" when the
	// rendered string has no styling.
	extractFgSGR := func(s string) string {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return ""
		}
		j := strings.Index(s[i:], "m")
		if j < 0 {
			return ""
		}
		return s[i : i+j+1]
	}

	type field struct {
		name string
		get  func(p theme.Palette) color.Color
	}
	fields := []field{
		{"Accent", func(p theme.Palette) color.Color { return p.Accent }},
		{"Warning", func(p theme.Palette) color.Color { return p.Warning }},
		{"Dim", func(p theme.Palette) color.Color { return p.Dim }},
		{"Success", func(p theme.Palette) color.Color { return p.Success }},
		{"Danger", func(p theme.Palette) color.Color { return p.Danger }},
		{"Progress", func(p theme.Palette) color.Color { return p.Progress }},
		{"Unknown", func(p theme.Palette) color.Color { return p.Unknown }},
		{"Info", func(p theme.Palette) color.Color { return p.Info }},
		{"Text", func(p theme.Palette) color.Color { return p.Text }},
		{"Gray", func(p theme.Palette) color.Color { return p.Gray }},
	}

	for _, themeName := range []string{
		"nord", "dracula", "solarized-light", "solarized-dark",
		"monokai", "tokyo-night", "gruvbox-dark", "gruvbox-light",
		"one-dark", "one-light",
	} {
		palette, ok := theme.Get(themeName)
		if !ok {
			continue
		}
		t.Run(themeName, func(t *testing.T) {
			applyTheme(palette)
			expected := extractFgSGR(lipgloss.NewStyle().Foreground(dimColor).Render("X"))
			if expected == "" {
				t.Fatalf("could not derive expected dim SGR for theme %s", themeName)
			}
			for _, f := range fields {
				c := f.get(palette)
				if c == nil {
					continue
				}
				styled := lipgloss.NewStyle().Foreground(c).Render("Healthy")
				out := desaturateANSI(styled)
				got := extractFgSGR(out)
				if got != expected {
					t.Errorf("theme=%s field=%s: fg-only color survived desaturation\n  input  = %q\n  output = %q\n  expected fg SGR = %q\n  got fg SGR      = %q",
						themeName, f.name, styled, out, expected, got)
				}
				// Sanity: visible text must be intact.
				if !strings.Contains(out, "Healthy") {
					t.Errorf("theme=%s field=%s: visible text lost: %q", themeName, f.name, out)
				}
			}
		})
	}
}

// TestRender_TreeUnderModal_AllThemes_NoStatusColorLeak iterates
// every preset and asserts that when a modal is up over the tree
// view, no row carries a saturated truecolor fg matching the
// theme's Success/Danger/Warning hues. This catches future themes
// whose RGB bytes fall in the same trap as nord/monokai/etc.
func TestRender_TreeUnderModal_AllThemes_NoStatusColorLeak(t *testing.T) {
	t.Cleanup(func() { applyTheme(theme.Default()) })

	themes := []string{"nord", "dracula", "solarized-light", "solarized-dark", "monokai", "tokyo-night", "gruvbox-dark", "gruvbox-light", "one-dark", "one-light"}
	for _, name := range themes {
		palette, ok := theme.Get(name)
		if !ok {
			continue
		}
		t.Run(name, func(t *testing.T) {
			applyTheme(palette)
			m := buildBaseModel(120, 30)
			m.applyThemeToModel()
			m.state.Navigation.View = model.ViewTree
			m.state.Mode = model.ModeConfirmSync

			out := m.renderMainLayout()

			toSGR := func(c interface{ RGBA() (uint32, uint32, uint32, uint32) }) string {
				r, g, b, _ := c.RGBA()
				// Match the truecolor fg payload anywhere in an SGR run:
				// standalone (`\x1b[38;2;R;G;Bm`), preceded by attrs
				// (`\x1b[1;38;2;R;G;Bm`), or followed by another param
				// like a bg (`\x1b[38;2;R;G;B;48;2;...m`).
				return "38;2;" + itoa(int(r>>8)) + ";" + itoa(int(g>>8)) + ";" + itoa(int(b>>8))
			}
			banned := []string{}
			if palette.Success != nil {
				if c, ok := palette.Success.(interface {
					RGBA() (uint32, uint32, uint32, uint32)
				}); ok {
					banned = append(banned, toSGR(c))
				}
			}
			if palette.Danger != nil {
				if c, ok := palette.Danger.(interface {
					RGBA() (uint32, uint32, uint32, uint32)
				}); ok {
					banned = append(banned, toSGR(c))
				}
			}
			for _, line := range strings.Split(out, "\n") {
				plain := stripANSI(line)
				// Skip the "Argonaut" badge — it intentionally keeps
				// its bg/fg.
				if strings.Contains(plain, "Argonaut") {
					continue
				}
				for _, b := range banned {
					if strings.Contains(line, b) {
						t.Errorf("theme %s: tree base under modal still carries banned saturated SGR %q in line: %q", name, b, line)
					}
				}
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Original 16-color check, kept for the no-theme/baseline harness path.
func TestRender_AppsList_SyncModal_NoColorLeaks(t *testing.T) {
	m := buildTestModelWithApps(100, 30)
	// cursor on app-a, sync target = app-b, app-c is mid-flash from
	// a previous refresh.
	m.state.Navigation.SelectedIdx = 0
	m.state.Selections.SelectedApps = map[string]bool{"app-b": true}
	m.state.UI.RefreshFlashApps = map[string]bool{"app-c": true}
	m.state.Mode = model.ModeConfirmSync

	out := m.renderMainLayout()

	// Saturated status fg colors that must NOT appear on rows other
	// than the selected one. (Health/Sync use truecolor in real
	// themes; in the default test palette they fall back to the
	// 16-color set — we check both.)
	saturatedFG := []string{
		"\x1b[32m", "\x1b[92m", // green
		"\x1b[31m", "\x1b[91m", // red
		"\x1b[33m", "\x1b[93m", // yellow
		"\x1b[38;5;10m", "\x1b[38;5;9m", "\x1b[38;5;11m",
	}

	for i, line := range strings.Split(out, "\n") {
		plain := stripANSI(line)
		// Skip the modal lines (they're allowed to be vivid).
		if strings.Contains(plain, "Sync") && strings.Contains(plain, "?") {
			continue
		}
		if strings.Contains(plain, "Confirm") || strings.Contains(plain, "y/n") || strings.Contains(plain, "[Sync]") {
			continue
		}
		// The selected sync target keeps its bg highlight; that's OK.
		isSelectedRow := strings.Contains(plain, "app-b")
		if isSelectedRow {
			continue
		}
		for _, sgr := range saturatedFG {
			if strings.Contains(line, sgr) {
				t.Errorf("line %d carries saturated fg %q under sync modal: %q", i, sgr, line)
			}
		}
	}
}

func TestWillDesaturateBase_TogglesWithModalState(t *testing.T) {
	m := buildTestModelWithApps(80, 24)
	if m.willDesaturateBase() {
		t.Fatal("expected false in normal mode")
	}
	m.state.Mode = model.ModeConfirmSync
	if !m.willDesaturateBase() {
		t.Fatal("expected true under ModeConfirmSync")
	}
	m.state.Mode = model.ModeNormal
	m.state.Modals.ConfirmSyncLoading = true
	if !m.willDesaturateBase() {
		t.Fatal("expected true while ConfirmSyncLoading")
	}
}
