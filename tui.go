package main

import (
	"fmt"
	"net/http"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
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
	state            screenState
	urlInput         textinput.Model
	titleInput       textinput.Model
	spinner          spinner.Model
	article          *readability.Article
	filename         string
	archivePath      string
	language         string
	wordCount        int
	imageCount       int
	err              error
	excludeImages    bool
	forceScrapingBee bool
	checkboxFocused  int // 0 = url input, 1 = include images, 2 = force scrapingbee
}

// Messages for async operations
type retrievalCompleteMsg struct {
	resp *http.Response
	err  error
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

func initialModel() model {
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

	return model{
		state:         inputScreen,
		urlInput:      urlInput,
		titleInput:    titleInput,
		spinner:       s,
		excludeImages: false,
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
		case "tab", "down":
			if m.state == inputScreen {
				m.checkboxFocused = (m.checkboxFocused + 1) % 3
				if m.checkboxFocused == 0 {
					m.urlInput.Focus()
				} else {
					m.urlInput.Blur()
				}
			}
		case "up":
			if m.state == inputScreen {
				m.checkboxFocused = (m.checkboxFocused + 2) % 3
				if m.checkboxFocused == 0 {
					m.urlInput.Focus()
				} else {
					m.urlInput.Blur()
				}
			}
		case " ":
			if m.state == inputScreen {
				if m.checkboxFocused == 1 {
					m.excludeImages = !m.excludeImages
				} else if m.checkboxFocused == 2 {
					m.forceScrapingBee = !m.forceScrapingBee
				}
			}
		case "enter":
			switch m.state {
			case inputScreen:
				if m.urlInput.Value() != "" {
					m.state = retrievalScreen
					return m, tea.Batch(m.spinner.Tick, retrieveContentCmd(m.urlInput.Value(), m.forceScrapingBee))
				}
			case editScreen:
				// Update title if changed
				if m.titleInput.Value() != "" {
					m.article.Title = m.titleInput.Value()
					m.filename = postprocessing.TitleToFilename(m.titleInput.Value())
				}
				m.state = sendingScreen
				return m, tea.Batch(m.spinner.Tick, sendArticle(m.article, m.filename, m.archivePath))
			case completionScreen:
				return m, tea.Quit
			}
		}

	case retrievalCompleteMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = completionScreen
		} else {
			m.state = postProcessingScreen
			return m, tea.Batch(m.spinner.Tick, processContentCmd(msg.resp, m.excludeImages))
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
		excludeImagesCheckbox := "☐"
		if m.excludeImages {
			excludeImagesCheckbox = "☑"
		}

		scrapingBeeCheckbox := "☐"
		if m.forceScrapingBee {
			scrapingBeeCheckbox = "☑"
		}

		excludeImagesStyle := subtleStyle
		scrapingBeeStyle := subtleStyle
		if m.checkboxFocused == 1 {
			excludeImagesStyle = headerStyle
		} else if m.checkboxFocused == 2 {
			scrapingBeeStyle = headerStyle
		}

		return fmt.Sprintf(
			"%s\n\n%s\n\n%s %s\n%s %s\n\n%s\n",
			headerStyle.Render("📚 Go to Kindle"),
			m.urlInput.View(),
			excludeImagesStyle.Render(excludeImagesCheckbox),
			excludeImagesStyle.Render("Exclude Images (resized to 300px)"),
			scrapingBeeStyle.Render(scrapingBeeCheckbox),
			scrapingBeeStyle.Render("Force ScrapingBee (slower but more reliable)"),
			subtleStyle.Render("Press Enter to fetch • Tab/↑↓ to navigate • Space to toggle • Ctrl+C to quit"),
		)

	case retrievalScreen:
		return fmt.Sprintf(
			"%s %s\n\n%s\n",
			m.spinner.View(),
			"🔍 Retrieving content...",
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

		var metadata string
		if !m.excludeImages && m.imageCount > 0 {
			metadata = fmt.Sprintf("Language: %s • Words: %d • Images: %d • File: %s",
				m.language, m.wordCount, m.imageCount, clickableFilePath)
		} else {
			metadata = fmt.Sprintf("Language: %s • Words: %d • File: %s",
				m.language, m.wordCount, clickableFilePath)
		}
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

// Command to retrieve content
func retrieveContentCmd(input string, forceScrapingBee bool) tea.Cmd {
	return func() tea.Msg {
		resp, err := retrieveContent(input, forceScrapingBee)
		return retrievalCompleteMsg{resp: resp, err: err}
	}
}

// Command to process content
func processContentCmd(resp *http.Response, excludeImages bool) tea.Cmd {
	return func() tea.Msg {
		article, filename, language, wordCount, imageCount, archivePath, err := postProcessContent(resp, excludeImages)
		return postProcessingCompleteMsg{article: article, filename: filename, archivePath: archivePath, language: language, wordCount: wordCount, imageCount: imageCount, err: err}
	}
}

// Command to send article
func sendArticle(article *readability.Article, filename string, archivePath string) tea.Cmd {
	return func() tea.Msg {
		err := processAndSend(article, filename, archivePath)
		return sendCompleteMsg{err: err}
	}
}
