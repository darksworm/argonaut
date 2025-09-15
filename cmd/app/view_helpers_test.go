package main

import (
    "regexp"
    "testing"
)

// stripANSI removes ANSI escape sequences for stable assertions.
var ansiTestRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(s string) string {
    return ansiTestRE.ReplaceAllString(s, "")
}

func TestAbbreviateStatus(t *testing.T) {
    cases := map[string]string{
        "Synced":      "Sync",
        "OutOfSync":   "Out",
        "Healthy":     "OK",
        "Degraded":    "Bad",
        "Progressing": "Prog",
        "Unknown":     "?",
        "OKX":         "OKX", // <= 4 stays as is
        "Longer":      "Long", // truncated
    }
    for in, want := range cases {
        if got := abbreviateStatus(in); got != want {
            t.Fatalf("abbreviateStatus(%q) = %q, want %q", in, got, want)
        }
    }
}

func TestCalculateColumnWidths(t *testing.T) {
    // Narrow
    n, s, h := calculateColumnWidths(30)
    if n != 24 || s != 2 || h != 2 { // 30 - 2(sep) - 2 - 2 = 24
        t.Fatalf("narrow widths = (%d,%d,%d), want (24,2,2)", n, s, h)
    }

    // Wide
    n, s, h = calculateColumnWidths(80)
    if n != 51 || s != 12 || h != 15 { // 80 - 2(sep) - 12 - 15 = 51
        t.Fatalf("wide widths = (%d,%d,%d), want (51,12,15)", n, s, h)
    }
}

func TestClipAnsiToWidth_StripsAtRuneBoundary(t *testing.T) {
    styled := headerStyle.Render("HELLO") // has ANSI codes
    clipped := clipAnsiToWidth(styled, 3)
    plain := stripANSI(clipped)
    if plain != "HEL" {
        t.Fatalf("clipAnsiToWidth to 3 => %q, want %q", plain, "HEL")
    }
}

func TestWrapAnsiToWidth_Basic(t *testing.T) {
    lines := wrapAnsiToWidth("abcdef", 3)
    if len(lines) != 2 || lines[0] != "abc" || lines[1] != "def" {
        t.Fatalf("wrapAnsiToWidth = %#v, want [\"abc\", \"def\"]", lines)
    }
}

func TestTruncateString_Basic(t *testing.T) {
    if got := truncateString("abcdef", 4); got != "a..." {
        t.Fatalf("truncateString short = %q, want %q", got, "a...")
    }
    if got := truncateString("abc", 4); got != "abc" {
        t.Fatalf("truncateString no-op = %q, want %q", got, "abc")
    }
}
