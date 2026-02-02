package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

// ConfigureResult holds the configuration result from the TUI.
type ConfigureResult struct {
	Config    *config.Config
	Cancelled bool
}

type screen interface {
	Init() tea.Cmd
	Update(tea.Msg) (screen, tea.Cmd)
	View() string
}

type wizardState struct {
	cfg        *config.Config
	onboarding bool
	cancelled  bool
	err        error
	result     *ConfigureResult
}

type optionItem struct {
	title    string
	desc     string
	value    string
	disabled bool
}

func (i optionItem) Title() string       { return i.title }
func (i optionItem) Description() string { return i.desc }
func (i optionItem) FilterValue() string {
	return strings.TrimSpace(i.title + " " + i.desc)
}

type toggleItem struct {
	title    string
	desc     string
	value    string
	selected bool
}

func (i toggleItem) Title() string {
	prefix := "[ ]"
	if i.selected {
		prefix = "[x]"
	}
	return prefix + " " + i.title
}

func (i toggleItem) Description() string { return i.desc }
func (i toggleItem) FilterValue() string {
	return strings.TrimSpace(i.title + " " + i.desc)
}

type modelOption struct {
	ID    string
	Title string
	Desc  string
}
