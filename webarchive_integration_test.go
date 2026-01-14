package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/internal/webarchive"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
)

func TestWebarchiveIntegration(t *testing.T) {
	resp, err := retrieveContent(context.Background(), "testdata/The Stick in the Stream – Rands in Repose.webarchive", false)
	if err != nil {
		t.Fatalf("retrieveContent error: %v", err)
	}
	defer resp.Body.Close()
	if resp.Request == nil {
		t.Fatalf("expected request on response")
	}
	if _, _, ok := webarchive.GetArchive(resp.Request.Context()); !ok {
		t.Fatalf("expected webarchive context on response request")
	}

	blockedClient := &http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network disabled by test")
		}),
	}
	article, _, _, err := postprocessing.ProcessArticleWithContext(context.Background(), resp, false, blockedClient)
	if err != nil {
		t.Fatalf("ProcessArticleWithContext error: %v", err)
	}

	if strings.TrimSpace(article.Title) == "" {
		t.Fatalf("expected non-empty title")
	}

	lang := whatlanggo.DetectLangWithOptions(article.TextContent, whatlanggo.Options{
		Whitelist: map[whatlanggo.Lang]bool{
			whatlanggo.Cmn: true,
			whatlanggo.Eng: true,
		},
	})
	wordCount := 0
	if lang == whatlanggo.Cmn {
		wordCount = utf8.RuneCountInString(article.Content)
	} else {
		wordCount = len(strings.Fields(article.Content))
	}
	if wordCount <= 0 {
		t.Fatalf("expected non-zero word count, got %d", wordCount)
	}
	if !strings.Contains(article.Content, "data:image/") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
		if err != nil {
			t.Fatalf("parse article content: %v", err)
		}
		imgs := make([]string, 0, 5)
		doc.Find("img").EachWithBreak(func(_ int, s *goquery.Selection) bool {
			if src, ok := s.Attr("src"); ok {
				imgs = append(imgs, src)
			}
			return len(imgs) < 5
		})
		t.Fatalf("expected inlined image data URL in article content; sample imgs: %v", imgs)
	}
}

func TestWebarchiveInlineThenReadability(t *testing.T) {
	raw, err := os.ReadFile("testdata/The Stick in the Stream – Rands in Repose.webarchive")
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
	if !strings.Contains(string(inlined), "data:image/") {
		t.Fatalf("expected inlined image data URL in HTML after inlining")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(inlined)))
	if err != nil {
		t.Fatalf("parse inlined HTML: %v", err)
	}
	if doc.Find("img").Length() == 0 {
		t.Fatalf("expected at least one img in inlined HTML")
	}
}

func TestWebarchiveReadabilityKeepsImages(t *testing.T) {
	raw, err := os.ReadFile("testdata/The Stick in the Stream – Rands in Repose.webarchive")
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
	if !strings.Contains(string(inlined), "data:image/") {
		t.Fatalf("expected inlined image data URL in HTML after inlining")
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

	if strings.Contains(article.Content, "data:image/") {
		t.Fatalf("expected readability to drop data URLs before reinlining")
	}

	reinlined, err := webarchive.InlineImagesInHTML(article.Content, baseURL, resources)
	if err != nil {
		t.Fatalf("InlineImagesInHTML error: %v", err)
	}
	if !strings.Contains(reinlined, "data:image/") {
		t.Fatalf("expected data URLs after reinlining readability output")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
