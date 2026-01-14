package webarchive

import (
	"net/url"
	"strings"
	"testing"
)

func TestInlineImages(t *testing.T) {
	baseURL := mustParseURL(t, "https://example.com/article")
	resources := map[string]Resource{
		"https://example.com/img.png": {
			URL:      "https://example.com/img.png",
			MIMEType: "image/png",
			Data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
		},
	}
	html := []byte(`<html><body><img src="/img.png"></body></html>`)

	inlined, err := InlineImages(html, baseURL, resources)
	if err != nil {
		t.Fatalf("InlineImages error: %v", err)
	}

	if !strings.Contains(string(inlined), "data:image/png;base64,") {
		t.Fatalf("expected inlined data URL, got: %s", string(inlined))
	}
}

func TestInlineImagesRemovesExternalFallbacks(t *testing.T) {
	baseURL := mustParseURL(t, "https://example.com/article")
	resources := map[string]Resource{}
	html := []byte(`<html><body><img src="https://example.com/remote.png" srcset="https://example.com/a.png 1x"></body></html>`)

	inlined, err := InlineImages(html, baseURL, resources)
	if err != nil {
		t.Fatalf("InlineImages error: %v", err)
	}

	out := string(inlined)
	if strings.Contains(out, "remote.png") {
		t.Fatalf("expected external src to be removed, got: %s", out)
	}
	if strings.Contains(out, "a.png") {
		t.Fatalf("expected external srcset to be removed, got: %s", out)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return u
}
