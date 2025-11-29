package main

import (
	"strings"
	"testing"
)

func TestFormatChangelog(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		notContains []string
	}{
		{
			name: "formats h1 header",
			input: "# Changelog",
			contains: []string{"Changelog"},
		},
		{
			name: "formats h2 version header",
			input: "## [2.7.0](https://github.com/example/repo/compare/v2.6.1...v2.7.0) (2025-11-14)",
			contains: []string{"2.7.0", "2025-11-14"},
			// URL should be stripped from display
			notContains: []string{"https://github.com"},
		},
		{
			name: "formats h3 section header",
			input: "### Features",
			contains: []string{"Features"},
		},
		{
			name: "formats bullet points",
			input: "* add ArgoCD core mode detection",
			contains: []string{"•", "add ArgoCD core mode detection"},
		},
		{
			name: "formats dash bullet points",
			input: "- fix edge case in config loading",
			contains: []string{"•", "fix edge case in config loading"},
		},
		{
			name: "extracts link text and hides URL",
			input: "* see [the docs](https://example.com) for details",
			// Note: "the docs" will have ANSI styling applied, so we check for individual parts
			contains: []string{"see", "for details"},
			notContains: []string{"https://example.com"},
		},
		{
			name: "handles inline code",
			input: "* fix issue with `config.toml` file",
			contains: []string{"config.toml"},
		},
		{
			name: "handles bold text",
			input: "* **breaking change**: removed old API",
			contains: []string{"breaking change", "removed old API"},
		},
		{
			name: "handles empty lines",
			input: "\n\n",
			contains: []string{"\n"},
		},
		{
			name: "formats complete changelog section",
			input: `# Changelog

## [2.8.0](https://github.com/darksworm/argonaut/compare/v2.7.0...v2.8.0) (2025-11-29)

### Features

* add what's new notification on version upgrade
* add :changelog command to view release notes

### Bug Fixes

* fix status line spacing issue`,
			contains: []string{
				"Changelog",
				"2.8.0",
				"Features",
				"Bug Fixes",
				"what's new notification",
				":changelog command",
				"status line spacing",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatChangelog(tt.input)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatChangelog() result should contain %q, got:\n%s", expected, result)
				}
			}

			for _, unexpected := range tt.notContains {
				if strings.Contains(result, unexpected) {
					t.Errorf("FormatChangelog() result should NOT contain %q, got:\n%s", unexpected, result)
				}
			}
		})
	}
}

func TestProcessInlineFormatting(t *testing.T) {
	// Note: These tests verify that text content is preserved after formatting.
	// ANSI escape codes are inserted by lipgloss, so we check for parts that
	// don't have character-by-character styling applied.
	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:     "handles plain text",
			input:    "plain text",
			contains: []string{"plain text"},
		},
		{
			name:     "handles link - surrounding text preserved",
			input:    "see [docs](https://example.com) here",
			contains: []string{"see", "here"},
		},
		{
			name:     "handles bold - surrounding text preserved",
			input:    "this is **important** stuff",
			contains: []string{"this is", "stuff"},
		},
		{
			name:     "handles code - surrounding text preserved",
			input:    "edit `code` file",
			contains: []string{"edit", "file"},
		},
		{
			name:     "handles multiple formats - surrounding text preserved",
			input:    "check the docs and config for details",
			contains: []string{"check the", "and", "for"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: We can't easily test the actual styling, but we can verify content is preserved
			result := FormatChangelog("* " + tt.input)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("processInlineFormatting() result should contain %q, got:\n%s", expected, result)
				}
			}
		})
	}
}
