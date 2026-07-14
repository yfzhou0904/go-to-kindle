package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/internal/repositories"
	"github.com/yfzhou0904/go-to-kindle/util"
)

func TestProcessAndSendPersistsDateContextOnSuccess(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	article := &readability.Article{Title: "Article", Content: "<p>Body</p>"}
	archivePath := filepath.Join(util.BaseDir(), "archive", "Article.html")
	if err := repositories.NewLocalFileRepository().SaveArticle(article, archivePath); err != nil {
		t.Fatalf("create initial archive: %v", err)
	}

	originalSend := sendEmailWithAttachment
	t.Cleanup(func() { sendEmailWithAttachment = originalSend })
	sendEmailWithAttachment = func(_, _, _, _, _, path string, _ int) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(contents), "Sent 2026/7/14") {
			return errors.New("attachment lacks date context")
		}
		return nil
	}

	if err := processAndSend(article, "Article.html", archivePath, true, time.Date(2026, time.July, 14, 8, 0, 0, 0, time.Local)); err != nil {
		t.Fatalf("processAndSend error: %v", err)
	}
	contents, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read final archive: %v", err)
	}
	if !strings.Contains(string(contents), "Sent 2026/7/14") {
		t.Fatal("successful archive should retain date context")
	}
}

func TestProcessAndSendRemovesSentDateOnFailure(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	article := &readability.Article{Title: "Article", Content: "<p>Body</p>"}
	archivePath := filepath.Join(util.BaseDir(), "archive", "Article.html")
	if err := repositories.NewLocalFileRepository().SaveArticle(article, archivePath); err != nil {
		t.Fatalf("create initial archive: %v", err)
	}

	originalSend := sendEmailWithAttachment
	t.Cleanup(func() { sendEmailWithAttachment = originalSend })
	sendEmailWithAttachment = func(_, _, _, _, _, _ string, _ int) error { return errors.New("smtp failed") }

	err := processAndSend(article, "Article.html", archivePath, true, time.Date(2026, time.July, 14, 8, 0, 0, 0, time.Local))
	if err == nil || !strings.Contains(err.Error(), "smtp failed") {
		t.Fatalf("expected SMTP failure, got %v", err)
	}
	contents, readErr := os.ReadFile(archivePath)
	if readErr != nil {
		t.Fatalf("read restored archive: %v", readErr)
	}
	if strings.Contains(string(contents), "Sent 2026/7/14") {
		t.Fatal("failed send archive should not claim it was sent")
	}
}
