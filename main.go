package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yfzhou0904/go-to-kindle/mail"
	"github.com/yfzhou0904/go-to-kindle/util"

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
	ScrapingBee: ConfigScrapingBee{
		APIKey: "YOUR_SCRAPINGBEE_API_KEY",
	},
}

func main() {
	// Parse command line arguments
	debug := flag.Bool("debug", false, "Enable debug mode to save intermediate HTML files")

	// Set custom usage function
	flag.Usage = func() {
		fmt.Println(helpMessage)
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Extract URL argument from non-flag arguments
	var url string
	if flag.NArg() > 0 {
		url = flag.Arg(0)
	}

	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var modelOpts []ModelOption
	if url != "" {
		modelOpts = append(modelOpts, WithURL(url))
	}
	if *debug {
		modelOpts = append(modelOpts, WithDebugFlag(*debug))
	}

	p := tea.NewProgram(initialModel(modelOpts...), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

// Process and send article
func processAndSend(article *readability.Article, filename string, archivePath string) error {
	// Check if we need to update the file with a new title
	currentArchivePath := filepath.Join(util.BaseDir(), "archive", filename)
	if currentArchivePath != archivePath {
		// Title was changed, need to rewrite the file with new filename
		_, err := createFile(currentArchivePath)
		if err != nil {
			return fmt.Errorf("failed to create new archive file: %v", err)
		}

		err = writeToFile(article, currentArchivePath)
		if err != nil {
			return fmt.Errorf("failed to write to new archive file: %v", err)
		}

		// Remove old file if different from new one
		if archivePath != currentArchivePath {
			os.Remove(archivePath)
		}

		archivePath = currentArchivePath
	} else {
		// Title unchanged, but we might need to update content if title was edited
		err := writeToFile(article, archivePath)
		if err != nil {
			return fmt.Errorf("failed to update archive file: %v", err)
		}
	}

	err := mail.SendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), archivePath, Conf.Email.Port)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

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

func createFile(p string) (*os.File, error) {
	// Create directories if they do not exist
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Create the file
	return os.Create(p)
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
