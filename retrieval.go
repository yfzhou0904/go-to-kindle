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

// fetchAndParse handles both web URLs and local files, returning processed article data
func fetchAndParse(input string, includeImages bool, forceScrapingBee bool) (*readability.Article, string, string, int, int, string, error) {
	link := input
	var resp *http.Response
	var err error

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url - use retrieval chain
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, "", "", 0, 0, "", fmt.Errorf("failed to parse URL: %v", err)
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
			return nil, "", "", 0, 0, "", fmt.Errorf("failed to retrieve webpage: %v", result.Error)
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
			return nil, "", "", 0, 0, "", fmt.Errorf("failed to resolve local file path: %v", err)
		}
		file, err := os.Open(absPath)
		if err != nil {
			return nil, "", "", 0, 0, "", fmt.Errorf("failed to open local file: %v", err)
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
