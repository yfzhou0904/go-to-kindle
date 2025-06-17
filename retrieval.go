package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
	"github.com/yfzhou0904/go-to-kindle/retrieval"
)

// retrieveContent handles both web URLs and local files, returning raw HTTP response
func retrieveContent(input string, forceScrapingBee bool) (*http.Response, bool, error) {
	link := input
	var resp *http.Response

	isLocalFile := false

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url - use retrieval chain
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, false, fmt.Errorf("failed to parse URL: %v", err)
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
			return nil, false, fmt.Errorf("failed to retrieve webpage: %v", result.Error)
		}

		// Create http.Response-like structure for compatibility
		resp = &http.Response{
			Body: result.Content,
			Request: &http.Request{
				URL: result.URL,
			},
		}
	} else {
		// local file - handle escaped whitespace
		isLocalFile = true
		unescapedPath := strings.ReplaceAll(link, "\\ ", " ")
		absPath, err := filepath.Abs(unescapedPath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to resolve local file path: %v", err)
		}
		file, err := os.Open(absPath)
		if err != nil {
			return nil, false, fmt.Errorf("failed to open local file: %v", err)
		}
		resp = &http.Response{
			Body: file,
			Request: &http.Request{
				URL: &url.URL{
					Path: link,
				},
			},
		}
	}

	return resp, isLocalFile, nil
}

// postProcessContent processes the retrieved content into a final article
func postProcessContent(resp *http.Response, isLocalFile bool, includeImages bool) (*readability.Article, string, string, int, int, string, error) {

	// Close the response body after processing
	defer resp.Body.Close()

	// Process the article using the new postprocessing package
	article, filename, imageCount, err := postprocessing.ProcessArticle(resp, includeImages)
	if err != nil {
		return nil, "", "", 0, 0, "", fmt.Errorf("failed to process article: %v", err)
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
		return nil, "", "", 0, 0, "", fmt.Errorf("article is too short (%d words)", wordCount)
	}

	// Write HTML file to archive immediately after processing
	archivePath := filepath.Join(baseDir(), "archive", filename)
	_, err = createFile(archivePath)
	if err != nil {
		return nil, "", "", 0, 0, "", fmt.Errorf("failed to create archive file: %v", err)
	}

	err = writeToFile(article, archivePath)
	if err != nil {
		return nil, "", "", 0, 0, "", fmt.Errorf("failed to write to archive file: %v", err)
	}

	return article, filename, lang.String(), wordCount, imageCount, archivePath, nil
}
