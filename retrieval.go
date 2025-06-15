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
	"github.com/yfzhou0904/go-to-kindle/retrieval"
)

// fetchAndParse handles both web URLs and local files, returning processed article data
func fetchAndParse(input string, forceScrapingBee bool) (*readability.Article, string, string, int, error) {
	link := input
	var resp *http.Response
	var err error

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url - use retrieval chain
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to parse URL: %v", err)
		}

		// Create retrieval chain
		retrievalConfig := retrieval.Config{
			ScrapingBeeAPIKey: Conf.ScrapingBee.APIKey,
			ForceScrapingBee:  forceScrapingBee,
		}
		chain := retrieval.NewChain(retrievalConfig)

		// Attempt retrieval
		result := chain.Retrieve(validURL)
		if result.Error != nil {
			return nil, "", "", 0, fmt.Errorf("failed to retrieve webpage: %v", result.Error)
		}
		defer result.Content.Close()

		// Create http.Response-like structure for compatibility
		resp = &http.Response{
			Body: result.Content,
			Request: &http.Request{
				URL: result.URL,
			},
		}
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

	// Post-process the article content
	article, err = postProcessArticle(article)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to post-process article: %v", err)
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

// postProcessArticle cleans up the article content by removing images and links
func postProcessArticle(article *readability.Article) (*readability.Article, error) {
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse content: %v", err)
	}

	// Remove images, media, and other unwanted elements
	contentDoc.Find("img,source,figure,svg").Remove()

	// Replace links with their text content
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		var buf strings.Builder
		s.Contents().Each(func(j int, c *goquery.Selection) {
			buf.WriteString(c.Text())
		})
		s.ReplaceWithHtml(buf.String())
	})

	article.Content, err = contentDoc.Find("body").Html()
	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %v", err)
	}

	return article, nil
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
