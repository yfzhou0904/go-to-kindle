package main

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yfzhou0904/go-to-kindle/postprocessing"
)

func TestRetrieveMarkdownFileSetsInputKind(t *testing.T) {
	path := t.TempDir() + "/notes.md"
	if err := os.WriteFile(path, []byte("# Notes"), 0600); err != nil {
		t.Fatal(err)
	}
	input, err := retrieveContent(context.Background(), path, false, true)
	if err != nil {
		t.Fatalf("retrieveContent: %v", err)
	}
	defer input.Body.Close()
	if input.Kind != InputMarkdown {
		t.Fatalf("expected Markdown input, got %v", input.Kind)
	}
}

func TestRetrieveClipboardContent(t *testing.T) {
	original := readClipboard
	t.Cleanup(func() { readClipboard = original })
	readClipboard = func() (string, error) { return "# Clipboard title\n\nShort note.", nil }

	input, err := retrieveClipboardContent()
	if err != nil {
		t.Fatalf("retrieveClipboardContent: %v", err)
	}
	defer input.Body.Close()
	if input.Kind != InputMarkdown {
		t.Fatalf("expected Markdown input, got %v", input.Kind)
	}
	contents, _ := io.ReadAll(input.Body)
	if string(contents) != "# Clipboard title\n\nShort note." {
		t.Fatalf("unexpected clipboard contents: %q", contents)
	}
}

func TestRetrieveClipboardContentRejectsEmptyClipboard(t *testing.T) {
	original := readClipboard
	t.Cleanup(func() { readClipboard = original })
	readClipboard = func() (string, error) { return " \n", nil }

	if _, err := retrieveClipboardContent(); err == nil {
		t.Fatal("expected empty clipboard error")
	}
}

func TestPostProcessMarkdownAllowsShortContent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	input := &InputResult{
		Kind:     InputMarkdown,
		Body:     io.NopCloser(strings.NewReader("# Useful answer\n\nUse `go test ./...`.")),
		Resolver: postprocessing.NewNetworkImageResolver(nil),
	}

	article, filename, _, wordCount, imageCount, _, err := postProcessContent(context.Background(), input, false)
	if err != nil {
		t.Fatalf("postProcessContent: %v", err)
	}
	if article.Title != "Useful answer" {
		t.Fatalf("unexpected title: %q", article.Title)
	}
	if filename != "Useful answer.html" {
		t.Fatalf("unexpected filename: %q", filename)
	}
	if wordCount == 0 || imageCount != 0 {
		t.Fatalf("unexpected counts: words=%d images=%d", wordCount, imageCount)
	}
	if !strings.Contains(article.Content, "<code>go test ./...</code>") {
		t.Fatalf("expected rendered code, got %q", article.Content)
	}
}
