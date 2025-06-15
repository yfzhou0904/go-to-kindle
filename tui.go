package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	readability "github.com/go-shiori/go-readability"
)

// Screen states for the TUI
type screenState int

const (
	inputScreen screenState = iota
	processingScreen
	editScreen
	completionScreen
)

// Main TUI model
type model struct {
	state      screenState
	urlInput   textinput.Model
	titleInput textinput.Model
	spinner    spinner.Model
	article    *readability.Article
	filename   string
	language   string
	wordCount  int
	err        error
}

// Messages for async operations
type fetchCompleteMsg struct {
	article   *readability.Article
	filename  string
	language  string
	wordCount int
	err       error
}

type sendCompleteMsg struct {
	err error
}

// Styles
var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#50FA7B"))
	subtleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4"))
)

func initialModel() model {
	// Initialize URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "Enter URL or local file path..."
	urlInput.Focus()
	urlInput.CharLimit = 500
	urlInput.Width = 50

	// Initialize title input
	titleInput := textinput.New()
	titleInput.CharLimit = 200
	titleInput.Width = 50

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		state:      inputScreen,
		urlInput:   urlInput,
		titleInput: titleInput,
		spinner:    s,
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			switch m.state {
			case inputScreen:
				if m.urlInput.Value() != "" {
					m.state = processingScreen
					return m, tea.Batch(m.spinner.Tick, fetchArticle(m.urlInput.Value()))
				}
			case editScreen:
				// Update title if changed
				if m.titleInput.Value() != "" {
					m.article.Title = m.titleInput.Value()
					m.filename = titleToFilename(m.titleInput.Value())
				}
				m.state = processingScreen
				return m, tea.Batch(m.spinner.Tick, sendArticle(m.article, m.filename))
			case completionScreen:
				return m, tea.Quit
			}
		}

	case fetchCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = completionScreen
		} else {
			m.article = msg.article
			m.filename = msg.filename
			m.language = msg.language
			m.wordCount = msg.wordCount
			m.titleInput.SetValue(msg.article.Title)
			m.titleInput.Focus()
			m.state = editScreen
		}
		return m, nil

	case sendCompleteMsg:
		m.err = msg.err
		m.state = completionScreen
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update inputs based on current screen
	switch m.state {
	case inputScreen:
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	case editScreen:
		var cmd tea.Cmd
		m.titleInput, cmd = m.titleInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) View() string {
	switch m.state {
	case inputScreen:
		return fmt.Sprintf(
			"%s\n\n%s\n\n%s\n",
			headerStyle.Render("📚 Go to Kindle"),
			m.urlInput.View(),
			subtleStyle.Render("Press Enter to fetch • Ctrl+C to quit"),
		)

	case processingScreen:
		return fmt.Sprintf(
			"%s %s\n\n%s\n",
			m.spinner.View(),
			"Processing...",
			subtleStyle.Render("Ctrl+C to quit"),
		)

	case editScreen:
		metadata := fmt.Sprintf("Language: %s • Words: %d • File: %s", 
			m.language, m.wordCount, m.filename)
		return fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n\n%s\n\n%s\n",
			headerStyle.Render("✏️  Edit Article Title"),
			subtleStyle.Render(fmt.Sprintf("Original: %s", m.article.Title)),
			subtleStyle.Render(metadata),
			m.titleInput.View(),
			subtleStyle.Render("Press Enter to send to Kindle • Edit title or keep as-is"),
			subtleStyle.Render("Ctrl+C or q to quit"),
		)

	case completionScreen:
		if m.err != nil {
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n",
				errorStyle.Render("❌ Error"),
				m.err.Error(),
				subtleStyle.Render("Press Enter or Ctrl+C to quit"),
			)
		} else {
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n",
				successStyle.Render("✅ Success!"),
				"Article sent to your Kindle successfully.",
				subtleStyle.Render("Press Enter or Ctrl+C to quit"),
			)
		}
	}

	return ""
}

// Async command to fetch and parse article
func fetchArticle(input string) tea.Cmd {
	return func() tea.Msg {
		article, filename, language, wordCount, err := fetchAndParse(input)
		return fetchCompleteMsg{article: article, filename: filename, language: language, wordCount: wordCount, err: err}
	}
}

// Async command to send article
func sendArticle(article *readability.Article, filename string) tea.Cmd {
	return func() tea.Msg {
		err := processAndSend(article, filename)
		return sendCompleteMsg{err: err}
	}
}