package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
)

var (
	// seeing these elements means parsing failed
	blockedKeyElems = []string{
		`<div id="cf-error-details">`,
		`<title>Attention Required! | Cloudflare</title>`,
	}
)

// fetchAndParse handles both web URLs and local files, returning processed article data
func fetchAndParse(input string) (*readability.Article, string, string, int, error) {
	link := input
	var resp *http.Response

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to parse URL: %v", err)
		}

		resp, err = getWebPage(validURL)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to get webpage: %v", err)
		}
		defer resp.Body.Close()
	} else {
		// local file
		absPath, err := filepath.Abs(link)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to resolve local file path: %v", err)
		}
		file, err := os.Open(absPath)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to open local file: %v", err)
		}
		defer file.Close()
		resp = &http.Response{
			Body: file,
			Request: &http.Request{
				URL: &url.URL{
					Path: link,
				},
			},
		}
	}

	article, filename, err := parseWebPage(resp, resp.Request.URL)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to parse webpage: %v", err)
	}

	// Check if article contains any blocked key elements indicating parsing failure
	for _, blockedElem := range blockedKeyElems {
		if strings.Contains(article.Content, blockedElem) {
			return nil, "", "", 0, fmt.Errorf("failed to parse webpage: we have probably been blocked, pattern: '%s'", blockedElem)
		}
	}

	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to parse content: %v", err)
	}
	contentDoc.Find("img,source,figure,svg").Remove()
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		var buf strings.Builder
		s.Contents().Each(func(j int, c *goquery.Selection) {
			buf.WriteString(c.Text())
		})
		s.ReplaceWithHtml(buf.String())
	})
	article.Content, err = contentDoc.Find("body").Html()
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to extract content: %v", err)
	}

	// language detection for better word counting
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
	if wordCount < 100 {
		return nil, "", "", 0, fmt.Errorf("article is too short (%d words)", wordCount)
	}

	return article, filename, lang.String(), wordCount, nil
}

// getWebPage fetches a webpage with proper headers to mimic a browser
func getWebPage(url *url.URL) (*http.Response, error) {
	// Create a new request using http
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set the User-Agent header to mimic a normal browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	// Create a new http client
	client := http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	// Send the request using the client
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// parseWebPage uses readability to extract clean content from a webpage
func parseWebPage(resp *http.Response, url *url.URL) (*readability.Article, string, error) {
	article, err := readability.FromReader(resp.Body, url)
	if err != nil {
		return nil, "", err
	}
	var title string
	if strings.HasPrefix(url.String(), "http") {
		title = article.Title
	} else {
		title = filepath.Base(url.Path)
		title = strings.TrimSuffix(title, filepath.Ext(title))
	}
	return &article, titleToFilename(title), nil
}