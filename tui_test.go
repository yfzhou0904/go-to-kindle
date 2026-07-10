package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputViewKeepsClipboardActionMinimalAndContextual(t *testing.T) {
	m := initialModel()
	m.checkboxFocused = 1
	view := m.View()

	if !strings.Contains(view, "Read Markdown from clipboard") {
		t.Fatal("expected clipboard action")
	}
	if strings.Contains(view, "Use browser") {
		t.Fatal("browser option should be hidden while clipboard action is focused")
	}
	if strings.Contains(view, "Markdown mode") || strings.Contains(view, "Markdown editor") {
		t.Fatal("unexpected extra Markdown controls")
	}
}

func TestInputNavigationIncludesClipboardAction(t *testing.T) {
	m := initialModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(model)
	if got.checkboxFocused != 1 {
		t.Fatalf("expected clipboard action focused after Tab, got %d", got.checkboxFocused)
	}
}
