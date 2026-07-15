package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Browser: ConfigBrowser{
		ChromePath: "",
	},
	Archive: ConfigArchive{
		RetentionDays: 365,
	},
}

var sendEmailWithAttachment = mail.SendEmailWithAttachment

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

	maybeCleanupArchive(Conf.Archive.RetentionDays)

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
func processAndSend(article *readability.Article, filename string, archivePath string, includeDateContext bool, sentTime time.Time) error {
	repo := repositories.NewLocalFileRepository()

	// Check if we need to update the file with a new title
	currentArchivePath := filepath.Join(util.BaseDir(), "archive", filename)
	if currentArchivePath != archivePath {
		// Title was changed, need to rewrite the file with new filename
		if err := saveFinalArticle(repo, article, currentArchivePath, includeDateContext, sentTime); err != nil {
			return fmt.Errorf("failed to write to new archive file: %v", err)
		}

		// Remove old file if different from new one
		if archivePath != currentArchivePath {
			os.Remove(archivePath)
		}

		archivePath = currentArchivePath
	} else {
		// Title unchanged, but we might need to update content if title was edited
		if err := saveFinalArticle(repo, article, archivePath, includeDateContext, sentTime); err != nil {
			return fmt.Errorf("failed to update archive file: %v", err)
		}
	}

	if err := sendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), archivePath, Conf.Email.Port); err != nil {
		if includeDateContext {
			if restoreErr := repo.SaveArticle(article, archivePath); restoreErr != nil {
				return fmt.Errorf("failed to send email: %v; failed to remove unsent date context from archive: %v", err, restoreErr)
			}
		}
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func saveFinalArticle(repo repositories.FileRepository, article *readability.Article, path string, includeDateContext bool, sentTime time.Time) error {
	if includeDateContext {
		return repo.SaveArticleWithDates(article, path, sentTime)
	}
	return repo.SaveArticle(article, path)
}
