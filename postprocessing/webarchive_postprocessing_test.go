package postprocessing

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/internal/webarchive"
)

func TestProcessContentWithWebarchiveImages(t *testing.T) {
	raw, err := os.ReadFile("../testdata/Test-Driven Development with an LLM for Fun and Profit - blog.yfzhou.webarchive")
	if err != nil {
		t.Fatalf("read webarchive: %v", err)
	}

	htmlContent, baseURL, resources, err := webarchive.DecodeFile(raw)
	if err != nil {
		t.Fatalf("DecodeFile error: %v", err)
	}
	inlined, err := webarchive.InlineImages(htmlContent, baseURL, resources)
	if err != nil {
		t.Fatalf("InlineImages error: %v", err)
	}

	resp := &http.Response{
		Body: io.NopCloser(strings.NewReader(string(inlined))),
		Request: &http.Request{
			URL: baseURL,
		},
	}
	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		t.Fatalf("readability error: %v", err)
	}

	inlined, err = webarchive.InlineImages([]byte(article.Content), baseURL, resources)
	if err != nil {
		t.Fatalf("InlineImages error: %v", err)
	}
	article.Content = string(inlined)
	if !strings.Contains(article.Content, "data:image/") {
		t.Fatalf("expected data URLs before processContent")
	}
	firstDataURL := extractFirstDataImage(article.Content)
	if firstDataURL == "" {
		t.Fatalf("expected at least one data:image URL before processContent")
	}
	if !strings.Contains(firstDataURL, ",") {
		snippet := dataURLSnippet(article.Content)
		t.Fatalf("data URL missing comma: %s; raw snippet: %s", firstDataURL[:min(len(firstDataURL), 120)], snippet)
	}
	if _, err := processBase64ImageData(firstDataURL); err != nil {
		t.Fatalf("processBase64ImageData error: %v", err)
	}

	resolver := NewWebarchiveImageResolver(resources)
	processed, imageCount, err := processContent(&article, baseURL, false, resolver)
	if err != nil {
		t.Fatalf("processContent error: %v", err)
	}
	if imageCount == 0 {
		t.Fatalf("expected images to survive processContent")
	}
	if !strings.Contains(processed.Content, "data:image/") {
		t.Fatalf("expected data URLs after processContent")
	}
}

func extractFirstDataImage(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	var out string
	doc.Find("img").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if src, ok := s.Attr("src"); ok && strings.HasPrefix(src, "data:image/") {
			out = src
			return false
		}
		return true
	})
	return out
}

func dataURLSnippet(html string) string {
	idx := strings.Index(html, "data:image/")
	if idx == -1 {
		return ""
	}
	start := idx - 120
	if start < 0 {
		start = 0
	}
	end := idx + 200
	if end > len(html) {
		end = len(html)
	}
	return html[start:end]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
