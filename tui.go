package main

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/internal/repositories"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
	"github.com/yfzhou0904/go-to-kindle/util"
)

// Screen states for the TUI
type screenState int

const (
	inputScreen screenState = iota
	retrievalScreen
	postProcessingScreen
	editScreen
	sendingScreen
	completionScreen
)

// Main TUI model
type model struct {
	state           screenState
	urlInput        textinput.Model
	titleInput      textinput.Model
	spinner         spinner.Model
	article         *readability.Article
	filename        string
	archivePath     string
	language        string
	wordCount       int
	imageCount      int
	err             error
	excludeImages   bool
	useChromedp     bool
	debug           bool
	checkboxFocused int  // 0 = URL input, 1 = clipboard, 2 = images, 3 = browser
	inputFromCLI    bool // true when input was supplied as a CLI arg (already shell-unescaped)
	inputSource     string
	includeDates    bool
	editFocused     int // 0 = title, 1 = date context
	width           int
}

// Messages for async operations
type retrievalCompleteMsg struct {
	input *InputResult
	err   error
}

type postProcessingCompleteMsg struct {
	article     *readability.Article
	filename    string
	archivePath string
	language    string
	wordCount   int
	imageCount  int
	err         error
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

// ModelOption represents a configuration option for initialModel
type ModelOption func(*model)

// WithURL sets the initial URL value and marks it as coming from the CLI
// (already shell-unescaped by the invoking shell).
func WithURL(url string) ModelOption {
	return func(m *model) {
		m.urlInput.SetValue(url)
		m.inputFromCLI = true
	}
}

// WithDebugFlag enables debug mode
func WithDebugFlag(debug bool) ModelOption {
	return func(m *model) {
		m.debug = debug
	}
}

func initialModel(opts ...ModelOption) model {
	// Initialize URL input
	urlInput := textinput.New()
	urlInput.Placeholder = "Enter URL or local file path..."
	urlInput.Focus()
	urlInput.CharLimit = 500

	// Initialize title input
	titleInput := textinput.New()
	titleInput.CharLimit = 500

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := model{
		state:         inputScreen,
		urlInput:      urlInput,
		titleInput:    titleInput,
		spinner:       s,
		excludeImages: false,
		includeDates:  true,
	}

	// Apply options
	for _, opt := range opts {
		opt(&m)
	}

	return m
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
		case "tab", "down":
			if m.state == inputScreen {
				m.checkboxFocused = (m.checkboxFocused + 1) % 4
				if m.checkboxFocused == 0 {
					m.urlInput.Focus()
				} else {
					m.urlInput.Blur()
				}
			} else if m.state == editScreen {
				m.editFocused = (m.editFocused + 1) % 2
				m.updateEditFocus()
			}
		case "up":
			if m.state == inputScreen {
				m.checkboxFocused = (m.checkboxFocused + 3) % 4
				if m.checkboxFocused == 0 {
					m.urlInput.Focus()
				} else {
					m.urlInput.Blur()
				}
			} else if m.state == editScreen {
				m.editFocused = (m.editFocused + 1) % 2
				m.updateEditFocus()
			}
		case " ":
			if m.state == inputScreen {
				switch m.checkboxFocused {
				case 2:
					m.excludeImages = !m.excludeImages
				case 3:
					m.useChromedp = !m.useChromedp
				}
			} else if m.state == editScreen && m.editFocused == 1 {
				m.includeDates = !m.includeDates
			}
		case "esc":
			if m.state == completionScreen {
				return m, tea.Quit
			}
			if m.state == editScreen {
				m.state = inputScreen
				m.article = nil
				m.filename = ""
				m.archivePath = ""
				m.language = ""
				m.wordCount = 0
				m.imageCount = 0
				m.inputSource = ""
				m.err = nil
				m.titleInput.Blur()
				if m.checkboxFocused == 0 {
					m.urlInput.Focus()
				}
				return m, nil
			}
		case "enter":
			switch m.state {
			case inputScreen:
				if m.checkboxFocused == 1 {
					m.state = retrievalScreen
					return m, tea.Batch(m.spinner.Tick, retrieveClipboardContentCmd())
				}
				if m.urlInput.Value() != "" {
					m.state = retrievalScreen
					return m, tea.Batch(m.spinner.Tick, retrieveContentCmd(m.urlInput.Value(), m.useChromedp, m.debug, m.inputFromCLI))

				}
			case editScreen:
				// Update title if changed
				if m.titleInput.Value() != "" {
					m.article.Title = m.titleInput.Value()
					m.filename = postprocessing.TitleToFilename(m.titleInput.Value())
				}
				m.state = sendingScreen
				sentTime := time.Now()
				return m, tea.Batch(m.spinner.Tick, sendArticle(m.article, m.filename, m.archivePath, m.includeDates, sentTime))
			case completionScreen:
				// Reset to initial state and return to input screen
				initial := initialModel()
				return initial, initial.spinner.Tick
			}
		}

	case retrievalCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = completionScreen
		} else {
			m.inputSource = msg.input.Source
			m.state = postProcessingScreen
			return m, tea.Batch(m.spinner.Tick, processContentCmd(msg.input, m.excludeImages, m.debug))
		}
		return m, nil

	case postProcessingCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = completionScreen
		} else {
			m.article = msg.article
			m.filename = msg.filename
			m.archivePath = msg.archivePath
			m.language = msg.language
			m.wordCount = msg.wordCount
			m.imageCount = msg.imageCount
			m.titleInput.SetValue(msg.article.Title)
			m.editFocused = 0
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		inputWidth := max(m.contentWidth(), 20)
		m.urlInput.Width = inputWidth
		m.titleInput.Width = inputWidth
		return m, nil
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

func (m model) contentWidth() int {
	if m.width <= 0 {
		return 80
	}
	return max(m.width-2, 20)
}

func (m model) wrapText(s string) string {
	return ansi.Wrap(s, m.contentWidth(), " /:_")
}

func (m *model) updateEditFocus() {
	if m.editFocused == 0 {
		m.titleInput.Focus()
	} else {
		m.titleInput.Blur()
	}
}

func (m model) View() string {
	switch m.state {
	case inputScreen:
		excludeImagesCheckbox := "☐"
		if m.excludeImages {
			excludeImagesCheckbox = "☑"
		}

		browserCheckbox := "☐"
		if m.useChromedp {
			browserCheckbox = "☑"
		}

		clipboardStyle := subtleStyle
		excludeImagesStyle := subtleStyle
		browserStyle := subtleStyle
		switch m.checkboxFocused {
		case 1:
			clipboardStyle = headerStyle
		case 2:
			excludeImagesStyle = headerStyle
		case 3:
			browserStyle = headerStyle
		}

		// Get proxy info for display
		proxyInfo := util.GetProxyInfoForDisplay()
		proxyDisplay := ""
		if proxyInfo != nil {
			proxyDisplay = fmt.Sprintf("\n\n%s", subtleStyle.Render(fmt.Sprintf("🌐 Proxy detected: %s (from %s)", proxyInfo.URL, proxyInfo.Source)))
		}

		browserControl := fmt.Sprintf("\n%s %s", browserStyle.Render(browserCheckbox), browserStyle.Render("Use browser"))
		if m.checkboxFocused == 1 {
			browserControl = ""
		}

		return fmt.Sprintf(
			"%s\n\n%s\n\n%s\n\n%s %s%s%s\n\n%s\n",
			headerStyle.Render("📚 Go to Kindle"),
			m.urlInput.View(),
			clipboardStyle.Render("▸ Read Markdown from clipboard"),
			excludeImagesStyle.Render(excludeImagesCheckbox),
			excludeImagesStyle.Render("Exclude images"),
			browserControl,
			proxyDisplay,
			subtleStyle.Render("Enter to select • Tab/↑↓ to navigate • Space to toggle • Ctrl+C to quit"),
		)

	case retrievalScreen:
		message := "🔍 Retrieving content..."
		if m.checkboxFocused == 1 {
			message = "📋 Reading Markdown from clipboard..."
		}
		return fmt.Sprintf(
			"%s %s\n\n%s\n",
			m.spinner.View(),
			message,
			subtleStyle.Render("Ctrl+C to quit"),
		)

	case postProcessingScreen:
		return fmt.Sprintf(
			"%s %s\n\n%s\n",
			m.spinner.View(),
			"⚙️ Processing article...",
			subtleStyle.Render("Ctrl+C to quit"),
		)

	case sendingScreen:
		return fmt.Sprintf(
			"%s %s\n\n%s\n",
			m.spinner.View(),
			"📧 Sending to Kindle...",
			subtleStyle.Render("Ctrl+C to quit"),
		)

	case editScreen:
		// Make file path clickable using OSC 8 hyperlink escape sequence
		clickableFilePath := fmt.Sprintf("\033]8;;file://%s\033\\%s\033]8;;\033\\", m.archivePath, m.archivePath)

		source := m.inputSource
		if source == "" {
			source = "Web / file"
		}
		var metadata string
		if !m.excludeImages && m.imageCount > 0 {
			metadata = fmt.Sprintf("Source: %s • Language: %s • Words: %d • Images: %d • File: %s",
				source, m.language, m.wordCount, m.imageCount, clickableFilePath)
		} else {
			metadata = fmt.Sprintf("Source: %s • Language: %s • Words: %d • File: %s",
				source, m.language, m.wordCount, clickableFilePath)
		}
		dateCheckbox := "☐"
		if m.includeDates {
			dateCheckbox = "☑"
		}
		dateStyle := subtleStyle
		if m.editFocused == 1 {
			dateStyle = headerStyle
		}
		datePreview := ""
		if m.includeDates {
			datePreview = "\n" + subtleStyle.Render(m.wrapText(repositories.FormatDateContext(m.article, time.Now())))
		}
		return fmt.Sprintf(
			"%s\n\n%s\n%s\n\n%s\n\n%s %s%s\n\n%s\n\n%s\n",
			headerStyle.Render("✏️  Edit Article Title"),
			subtleStyle.Render(m.wrapText(fmt.Sprintf("Original: %s", m.article.Title))),
			subtleStyle.Render(m.wrapText(metadata)),
			m.titleInput.View(),
			dateStyle.Render(dateCheckbox),
			dateStyle.Render("Add date context to article"),
			datePreview,
			subtleStyle.Render("Press Enter to send to Kindle • Tab/↑↓ to navigate • Space to toggle"),
			subtleStyle.Render("Esc to go back • Ctrl+C to quit"),
		)

	case completionScreen:
		if m.err != nil {
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n\n%s\n",
				errorStyle.Render("❌ Error"),
				errorStyle.Render(m.wrapText(m.err.Error())),
				subtleStyle.Render(m.wrapText(fmt.Sprintf("Original URL: %s", m.urlInput.Value()))),
				subtleStyle.Render("Press Enter to send another • Esc/Ctrl+C to quit"),
			)
		} else {
			return fmt.Sprintf(
				"%s\n\n%s\n\n%s\n",
				successStyle.Render("✅ Success!"),
				"Going to your kindle!",
				subtleStyle.Render("Press Enter to send another • Esc/Ctrl+C to quit"),
			)
		}
	}

	return ""
}

// Command to retrieve content
func retrieveContentCmd(input string, useChromedp bool, debug bool, inputFromCLI bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if debug {
			ctx = util.WithDebug(ctx, debug)
		}
		result, err := retrieveContent(ctx, input, useChromedp, inputFromCLI)
		return retrievalCompleteMsg{input: result, err: err}
	}
}

func retrieveClipboardContentCmd() tea.Cmd {
	return func() tea.Msg {
		result, err := retrieveClipboardContent()
		return retrievalCompleteMsg{input: result, err: err}
	}
}

// Command to process content
func processContentCmd(input *InputResult, excludeImages bool, debug bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if debug {
			ctx = util.WithDebug(ctx, debug)
		}
		article, filename, language, wordCount, imageCount, archivePath, err := postProcessContent(ctx, input, excludeImages)
		return postProcessingCompleteMsg{article: article, filename: filename, archivePath: archivePath, language: language, wordCount: wordCount, imageCount: imageCount, err: err}
	}
}

// Command to send article
func sendArticle(article *readability.Article, filename string, archivePath string, includeDateContext bool, sentTime time.Time) tea.Cmd {
	return func() tea.Msg {
		err := processAndSend(article, filename, archivePath, includeDateContext, sentTime)
		return sendCompleteMsg{err: err}
	}
}
