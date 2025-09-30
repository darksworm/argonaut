package main

import "github.com/charmbracelet/lipgloss/v2"

// Layout dimension constants used across view rendering functions
const (
	// Border lines (top and bottom) for bordered content
	layoutBorderLines = 2

	// Table header lines (currently no header rows)
	layoutTableHeaderLines = 0

	// Tag line (currently unused)
	layoutTagLine = 0

	// Status line at bottom of screen
	layoutStatusLines = 1

	// Margin line between header and content
	layoutMarginTopLines = 1
)

// Color mappings for the application
var (
	// Core UI element colors
	magentaBright = lipgloss.Color("13") // Selection highlight
	yellowBright  = lipgloss.Color("11") // Headers, warnings
	dimColor      = lipgloss.Color("8")  // Dimmed text
	cyanBright    = lipgloss.Color("14") // Cyan accents, links
	whiteBright   = lipgloss.Color("15") // Bright white text
	blueBright    = lipgloss.Color("12") // Blue for info logs

	// Status colors for sync and health states
	syncedColor    = lipgloss.Color("10") // Green for Synced/Healthy
	outOfSyncColor = lipgloss.Color("9")  // Red for OutOfSync/Degraded
	progressColor  = lipgloss.Color("11") // Yellow for Progressing
	unknownColor   = lipgloss.Color("8")  // Dim for Unknown

	// Additional colors for modals and special cases
	black              = lipgloss.Color("0")   // Black background
	white              = lipgloss.Color("15")  // White (alias for whiteBright)
	redColor           = lipgloss.Color("9")   // Red (alias for outOfSyncColor)
	grayDesaturated    = lipgloss.Color("245") // Light gray for desaturated overlays
	grayInactiveButton = lipgloss.Color("238") // Dark gray for inactive buttons
	grayButtonDisabled = lipgloss.Color("236") // Lighter gray for disabled buttons

	// UI-specific colors
	grayBorder      = lipgloss.Color("240") // Border gray for tables and inputs
	grayPrompt      = lipgloss.Color("7")   // Prompt text gray
	pinkSpinner     = lipgloss.Color("205") // Pink for spinner
	yellowTable     = lipgloss.Color("229") // Yellow for table headers
	blueTable       = lipgloss.Color("57")  // Blue background for table headers
	grayBadgeBg     = lipgloss.Color("243") // Gray background for badges
	blackBadgeFg    = lipgloss.Color("16")  // Black foreground for badges
	grayServerLabel = lipgloss.Color("240") // Gray for server label background
)
