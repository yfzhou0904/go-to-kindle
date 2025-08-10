package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yfzhou0904/go-to-kindle/internal/repositories"
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
	repo := repositories.NewLocalFileRepository()

	// Check if we need to update the file with a new title
	currentArchivePath := filepath.Join(util.BaseDir(), "archive", filename)
	if currentArchivePath != archivePath {
		// Title was changed, need to rewrite the file with new filename
		if err := repo.SaveArticle(article, currentArchivePath); err != nil {
			return fmt.Errorf("failed to write to new archive file: %v", err)
		}

		// Remove old file if different from new one
		if archivePath != currentArchivePath {
			os.Remove(archivePath)
		}

		archivePath = currentArchivePath
	} else {
		// Title unchanged, but we might need to update content if title was edited
		if err := repo.SaveArticle(article, archivePath); err != nil {
			return fmt.Errorf("failed to update archive file: %v", err)
		}
	}

	if err := mail.SendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), archivePath, Conf.Email.Port); err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}
