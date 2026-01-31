package tui

import "github.com/charmbracelet/lipgloss"

// Color palette for hyprvoice TUI
// Using a modern, accessible color scheme
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple - main accent
	ColorSecondary = lipgloss.Color("#06B6D4") // Cyan - secondary accent

	// Status colors
	ColorSuccess = lipgloss.Color("#22C55E") // Green
	ColorError   = lipgloss.Color("#EF4444") // Red
	ColorWarning = lipgloss.Color("#F59E0B") // Amber

	// Text colors
	ColorText   = lipgloss.Color("#F8FAFC") // Bright white
	ColorMuted  = lipgloss.Color("#94A3B8") // Slate gray
	ColorSubtle = lipgloss.Color("#64748B") // Darker gray

	// Background colors
	ColorBg        = lipgloss.Color("#0F172A") // Dark slate
	ColorBgAlt     = lipgloss.Color("#1E293B") // Slightly lighter
	ColorHighlight = lipgloss.Color("#334155") // Selection highlight
)
