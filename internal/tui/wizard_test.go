package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/leonardotrapani/hyprvoice/internal/config"
)

func TestWizardMenuTransitionAppliesSize(t *testing.T) {
	cfg := &config.Config{}
	state := &wizardState{cfg: cfg}
	model := newWizardModel(state, newMenuScreen(state))

	updated, _ := model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	model = updated.(wizardModel)

	// move down to "Voice Model" item (index 1) which leads to a listScreen
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(wizardModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(wizardModel)

	listScreen, ok := model.screen.(*listScreen)
	if !ok {
		t.Fatalf("expected list screen after selection, got %T", model.screen)
	}
	if listScreen.list.Width() <= 0 || listScreen.list.Height() <= 0 {
		t.Fatalf("expected list size to be set, got width=%d height=%d", listScreen.list.Width(), listScreen.list.Height())
	}
}
