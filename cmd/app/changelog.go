package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

const changelogURL = "https://raw.githubusercontent.com/darksworm/argonaut/refs/heads/main/CHANGELOG.md"

// fetchChangelog fetches the changelog from GitHub
func (m *Model) fetchChangelog() tea.Cmd {
	return func() tea.Msg {
		logger := cblog.With("component", "changelog")
		logger.Info("Fetching changelog")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(changelogURL)
		if err != nil {
			logger.Error("Failed to fetch changelog", "err", err)
			return model.ChangelogLoadedMsg{
				Content: "",
				Error:   fmt.Errorf("network error: %w", err),
			}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			logger.Error("Changelog fetch returned non-200", "status", resp.StatusCode)
			return model.ChangelogLoadedMsg{
				Content: "",
				Error:   fmt.Errorf("HTTP %d - changelog unavailable", resp.StatusCode),
			}
		}

		// Limit read to 1MB to prevent memory issues
		limitedReader := io.LimitReader(resp.Body, 1*1024*1024)
		body, err := io.ReadAll(limitedReader)
		if err != nil {
			logger.Error("Failed to read changelog", "err", err)
			return model.ChangelogLoadedMsg{
				Content: "",
				Error:   fmt.Errorf("read error: %w", err),
			}
		}

		logger.Info("Changelog fetched successfully", "size", len(body))
		return model.ChangelogLoadedMsg{
			Content: string(body),
			Error:   nil,
		}
	}
}

// FormatChangelog converts markdown changelog to styled terminal output
func FormatChangelog(content string) string {
	var result strings.Builder
	lines := strings.Split(content, "\n")

	// Styles matching app theme
	h1Style := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	h2Style := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	h3Style := lipgloss.NewStyle().Foreground(syncedColor).Bold(true)
	bulletStyle := lipgloss.NewStyle().Foreground(cyanBright)
	linkStyle := lipgloss.NewStyle().Foreground(magentaBright).Underline(true)
	textStyle := lipgloss.NewStyle().Foreground(whiteBright)

	// Regex patterns
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	codeRe := regexp.MustCompile("`([^`]+)`")
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)

	prevWasEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		var formatted string

		switch {
		case strings.HasPrefix(trimmed, "# "):
			// Main title (# Changelog)
			title := strings.TrimPrefix(trimmed, "# ")
			formatted = h1Style.Render(title)
			prevWasEmpty = false
		case strings.HasPrefix(trimmed, "## "):
			// Version header (## [2.7.0])
			version := strings.TrimPrefix(trimmed, "## ")
			// Strip link markdown from version header
			version = linkRe.ReplaceAllString(version, "$1")
			formatted = "\n" + h2Style.Render(version)
			prevWasEmpty = false
		case strings.HasPrefix(trimmed, "### "):
			// Section header (### Features, ### Bug Fixes)
			section := strings.TrimPrefix(trimmed, "### ")
			formatted = "  " + h3Style.Render(section)
			prevWasEmpty = false
		case strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- "):
			// List item
			item := trimmed[2:]
			item = processInlineFormatting(item, linkRe, codeRe, boldRe, linkStyle, textStyle)
			bullet := bulletStyle.Render("•")
			formatted = "    " + bullet + " " + item
			prevWasEmpty = false
		case trimmed == "":
			// Skip consecutive empty lines
			if prevWasEmpty {
				continue
			}
			formatted = ""
			prevWasEmpty = true
		default:
			// Regular text
			formatted = processInlineFormatting(line, linkRe, codeRe, boldRe, linkStyle, textStyle)
			prevWasEmpty = false
		}

		result.WriteString(formatted + "\n")
	}

	return result.String()
}

// processInlineFormatting applies inline markdown formatting
func processInlineFormatting(text string, linkRe, codeRe, boldRe *regexp.Regexp, linkStyle, textStyle lipgloss.Style) string {
	// Replace links: [text](url) -> styled text (hide URL since not clickable)
	text = linkRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		if len(parts) >= 2 {
			return linkStyle.Render(parts[1])
		}
		return match
	})

	// Replace bold: **text** -> styled text
	text = boldRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := boldRe.FindStringSubmatch(match)
		if len(parts) >= 2 {
			return textStyle.Bold(true).Render(parts[1])
		}
		return match
	})

	// Replace inline code: `code` -> styled code
	codeStyle := lipgloss.NewStyle().Background(mutedBG).Foreground(whiteBright)
	text = codeRe.ReplaceAllStringFunc(text, func(match string) string {
		parts := codeRe.FindStringSubmatch(match)
		if len(parts) >= 2 {
			return codeStyle.Render(parts[1])
		}
		return match
	})

	return text
}
