package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yfzhou0904/go-to-kindle/mail"

	readability "github.com/go-shiori/go-readability"

	tea "github.com/charmbracelet/bubbletea"
)

var Conf Config = Config{
	Email: ConfigEmail{
		SMTPServer: "smtp.example.com",
		Port:       456,
		From:       "YOUR@EMAIL.com",
		Password:   "YOUR_EMAIL_PSWD",
		To:         "YOU@kindle.com",
	},
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

// Process and send article
func processAndSend(article *readability.Article, filename string) error {
	createFile(filepath.Join(baseDir(), "archive", filename))
	err := writeToFile(article, filepath.Join(baseDir(), "archive", filename))
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	err = mail.SendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), filepath.Join(baseDir(), "archive", filename), Conf.Email.Port)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
	<title>{{.Title}}</title>
	<meta name="author" content="{{.Author}}">
</head>
<body>
	{{.Content}}
</body>
</html>
`

type HtmlData struct {
	Title   string
	Content string
	Author  string
}

func writeToFile(article *readability.Article, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.New("html").Parse(htmlTemplate))
	err = t.Execute(file, HtmlData{
		Title:   article.Title,
		Author:  article.Byline,
		Content: article.Content,
	})
	if err != nil {
		return err
	}

	return nil
}

// replace problematic characters in page title to give a generally valid filename
func titleToFilename(title string) string {
	filename := strings.ReplaceAll(title, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	return filename + ".html"
}

// user config and article data are stored in ~/.go-to-kindle
func baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(home, ".go-to-kindle")
}

func createFile(p string) (*os.File, error) {
	// Create directories if they do not exist
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Create the file
	return os.Create(p)
}

