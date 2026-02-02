package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Base styles for hyprvoice TUI components
var (
	// Header style for titles and section headers
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	// Label style for form field labels
	StyleLabel = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	// Success style for positive feedback
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	// Error style for error messages
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Warning style for warnings
	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning)

	// Muted style for secondary text
	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Subtle style for hints and descriptions
	StyleSubtle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Italic(true)

	// Highlight style for selected/focused items
	StyleHighlight = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	// Selected style for chosen options
	StyleSelected = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// Box style for bordered containers
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSubtle).
			Padding(1, 2)

	// FocusedBox style for focused containers
	StyleFocusedBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)
)

const logoASCII = `
 _                            _          
| |__  _   _ _ __  _ ____   _(_) ___ ___ 
| '_ \| | | | '_ \| '__\ \ / / |/ __/ _ \
| | | | |_| | |_) | |   \ V /| | (_|  __/
|_| |_|\__, | .__/|_|    \_/ |_|\___\___|
       |___/|_|                          `

// Logo returns the hyprvoice ASCII art
func Logo() string {
	return StyleHeader.Render(strings.Trim(logoASCII, "\n"))
}

func LogoLines() []string {
	return strings.Split(strings.Trim(logoASCII, "\n"), "\n")
}
