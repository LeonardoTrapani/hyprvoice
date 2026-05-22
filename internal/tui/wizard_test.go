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

func TestInjectionScreenPreservesConfiguredBackendOrder(t *testing.T) {
	cfg := &config.Config{
		Injection: config.InjectionConfig{
			Backends: []string{"dotool", "wtype", "clipboard"},
		},
	}
	state := &wizardState{cfg: cfg}

	next := newInjectionScreen(state, func() screen { return nil })
	screen, ok := next.(*multiSelectScreen)
	if !ok {
		t.Fatalf("expected multi select screen, got %T", next)
	}
	items := listToToggleItems(screen.list.Items())

	wantOrder := []string{"dotool", "wtype", "clipboard", "ydotool"}
	if len(items) != len(wantOrder) {
		t.Fatalf("items length = %d, want %d", len(items), len(wantOrder))
	}
	for i, want := range wantOrder {
		if items[i].value != want {
			t.Fatalf("item %d value = %q, want %q", i, items[i].value, want)
		}
	}

	screen.onSubmit(items)
	wantSaved := []string{"dotool", "wtype", "clipboard"}
	if len(cfg.Injection.Backends) != len(wantSaved) {
		t.Fatalf("saved backends length = %d, want %d", len(cfg.Injection.Backends), len(wantSaved))
	}
	for i, want := range wantSaved {
		if cfg.Injection.Backends[i] != want {
			t.Fatalf("saved backend %d = %q, want %q", i, cfg.Injection.Backends[i], want)
		}
	}
}
