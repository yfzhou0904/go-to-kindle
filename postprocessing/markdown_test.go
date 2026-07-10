package postprocessing

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/yuin/goldmark/text"
)

func TestProcessMarkdownRendersGFMAndPreservesLinks(t *testing.T) {
	source := "# Reading Notes\n\n[Reference](https://example.com)\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n```go\nfmt.Println(\"hi\")\n```"
	resp := &http.Response{
		Body:    io.NopCloser(strings.NewReader(source)),
		Request: &http.Request{URL: &url.URL{}},
	}

	article, filename, images, err := ProcessMarkdownWithContext(context.Background(), resp, false, NewNetworkImageResolver(nil))
	if err != nil {
		t.Fatalf("ProcessMarkdownWithContext: %v", err)
	}
	if article.Title != "Reading Notes" || filename != "Reading Notes.html" {
		t.Fatalf("unexpected title/filename: %q %q", article.Title, filename)
	}
	for _, expected := range []string{"<table>", "<pre><code class=\"language-go\">", "href=\"https://example.com\""} {
		if !strings.Contains(article.Content, expected) {
			t.Errorf("expected %q in rendered content", expected)
		}
	}
	if images != 0 {
		t.Fatalf("expected no images, got %d", images)
	}
}

func TestMarkdownTitleFallsBackToFirstText(t *testing.T) {
	source := []byte("A concise explanation without a heading.\n\nMore detail.")
	md := newMarkdownParser()
	doc := md.Parser().Parse(text.NewReader(source))
	if got := markdownTitle(doc, source); got != "A concise explanation without a heading." {
		t.Fatalf("unexpected title: %q", got)
	}
}
