package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

type wizardModel struct {
	state  *wizardState
	screen screen
	width  int
	height int
}

func newWizardModel(state *wizardState, start screen) wizardModel {
	return wizardModel{state: state, screen: start}
}

func (m wizardModel) Init() tea.Cmd {
	if m.screen == nil {
		return tea.Quit
	}
	return m.screen.Init()
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.state.cancelled = true
			m.state.result = &ConfigureResult{Cancelled: true}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	if m.screen == nil {
		return m, tea.Quit
	}

	next, cmd := m.screen.Update(msg)
	if next == nil {
		if m.state.result == nil {
			m.state.result = &ConfigureResult{Config: m.state.cfg, Cancelled: m.state.cancelled}
		}
		return m, tea.Quit
	}
	if next != m.screen {
		var sizeCmd tea.Cmd
		if m.width > 0 && m.height > 0 {
			updated, scmd := next.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			if updated == nil {
				if m.state.result == nil {
					m.state.result = &ConfigureResult{Config: m.state.cfg, Cancelled: m.state.cancelled}
				}
				return m, tea.Quit
			}
			next = updated
			sizeCmd = scmd
		}
		m.screen = next
		initCmd := m.screen.Init()
		return m, tea.Batch(cmd, sizeCmd, initCmd)
	}
	m.screen = next
	return m, cmd
}

func (m wizardModel) View() string {
	if m.screen == nil {
		return ""
	}
	return m.screen.View()
}

// Run starts the TUI configuration wizard.
// If onboarding is true, forces the guided wizard flow even if config exists.
func Run(existingConfig *config.Config, onboarding bool) (*ConfigureResult, error) {
	if existingConfig == nil {
		return nil, fmt.Errorf("config is required")
	}

	state := &wizardState{cfg: existingConfig}
	if onboarding || !hasUserChanges(existingConfig) {
		state.onboarding = true
	}

	var start screen
	if state.onboarding {
		start = newWelcomeScreen(state)
	} else {
		start = newMenuScreen(state)
	}

	model := newWizardModel(state, start)
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		return &ConfigureResult{Cancelled: true}, err
	}

	if state.err != nil {
		return &ConfigureResult{Cancelled: true}, state.err
	}
	if state.result == nil {
		state.result = &ConfigureResult{Config: existingConfig, Cancelled: state.cancelled}
	}
	return state.result, nil
}
