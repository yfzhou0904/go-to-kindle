package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	readability "github.com/go-shiori/go-readability"
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

func TestEditScreenDateContextDefaultsOnAndCanBeToggled(t *testing.T) {
	published := time.Date(2020, time.January, 2, 0, 0, 0, 0, time.UTC)
	m := initialModel()
	m.state = editScreen
	m.article = &readability.Article{Title: "Article", PublishedTime: &published}
	m.titleInput.SetValue("Article")

	if !m.includeDates || !strings.Contains(m.View(), "☑") || !strings.Contains(m.View(), "Published 2020/1/2") {
		t.Fatal("date context should default on with a preview")
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := updated.(model)
	if got.editFocused != 1 {
		t.Fatal("Tab should focus the date checkbox")
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeySpace})
	got = updated.(model)
	if got.includeDates || strings.Contains(got.View(), "Published 2020/1/2") {
		t.Fatal("Space should disable date context and hide its preview")
	}
}

func TestNewlyProcessedArticleResetsDateContextToOn(t *testing.T) {
	m := initialModel()
	m.state = editScreen
	m.article = &readability.Article{Title: "Previous article"}
	m.includeDates = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(model)
	if m.state != inputScreen || m.includeDates {
		t.Fatal("returning to input should preserve the previous review choice until another article is processed")
	}

	updated, _ = m.Update(postProcessingCompleteMsg{
		article:  &readability.Article{Title: "New article"},
		filename: "New article.html",
	})
	got := updated.(model)
	if got.state != editScreen || !got.includeDates {
		t.Fatal("date context should default on for each newly processed article")
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

func TestEnterSubmitsURLFromOptionRows(t *testing.T) {
	for _, focus := range []int{0, 2, 3} {
		t.Run(fmt.Sprintf("focus_%d", focus), func(t *testing.T) {
			m := initialModel(WithURL("https://example.com/article"))
			m.checkboxFocused = focus

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			got := updated.(model)
			if got.state != retrievalScreen || cmd == nil {
				t.Fatalf("expected URL retrieval from focus %d", focus)
			}
		})
	}
}

func TestEscFromEditPreservesInputConfiguration(t *testing.T) {
	m := initialModel(
		WithURL("/tmp/quoted path/article.html"),
		WithDebugFlag(true),
	)
	m.state = editScreen
	m.checkboxFocused = 3
	m.excludeImages = true
	m.useChromedp = true
	m.inputSource = "Web"
	m.filename = "article.html"

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(model)
	if cmd != nil {
		t.Fatal("expected no command when returning to input")
	}
	if got.state != inputScreen {
		t.Fatalf("expected input screen, got %v", got.state)
	}
	if got.urlInput.Value() != "/tmp/quoted path/article.html" || !got.inputFromCLI || !got.debug {
		t.Fatal("CLI input state was not preserved")
	}
	if !got.excludeImages || !got.useChromedp || got.checkboxFocused != 3 {
		t.Fatal("input controls were not preserved")
	}
	if got.filename != "" || got.inputSource != "" {
		t.Fatal("processed document state was not cleared")
	}
}
