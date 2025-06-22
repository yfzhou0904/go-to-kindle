package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
	"github.com/yfzhou0904/go-to-kindle/retrieval"
	"github.com/yfzhou0904/go-to-kindle/util"
)

// unescapeFilePath removes terminal auto-escaping from file paths
func unescapeFilePath(path string) string {
	// Handle common auto-escaped characters in macOS terminal
	path = strings.ReplaceAll(path, "\\ ", " ")
	path = strings.ReplaceAll(path, "\\(", "(")
	path = strings.ReplaceAll(path, "\\)", ")")
	path = strings.ReplaceAll(path, "\\[", "[")
	path = strings.ReplaceAll(path, "\\]", "]")
	path = strings.ReplaceAll(path, "\\&", "&")
	path = strings.ReplaceAll(path, "\\;", ";")
	path = strings.ReplaceAll(path, "\\'", "'")
	return path
}

// retrieveContent handles both web URLs and local files, returning raw HTTP response
func retrieveContent(ctx context.Context, input string, forceScrapingBee bool) (*http.Response, error) {
	link := input
	var resp *http.Response

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url - use retrieval chain
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %v", err)
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
			return nil, fmt.Errorf("failed to retrieve webpage: %v", result.Error)
		}

		// Create http.Response-like structure for compatibility
		resp = &http.Response{
			Body: result.Content,
			Request: &http.Request{
				URL: result.URL,
			},
		}
	} else {
		// local file - handle both drag-drop and copy-paste (both may be escaped)
		var file *os.File
		var err error
		var absPath string
		
		// Clean input - remove any leading/trailing whitespace and handle quotes
		cleanLink := strings.TrimSpace(link)
		// Remove surrounding quotes if present (sometimes drag-drop adds them)
		if (strings.HasPrefix(cleanLink, `"`) && strings.HasSuffix(cleanLink, `"`)) ||
		   (strings.HasPrefix(cleanLink, `'`) && strings.HasSuffix(cleanLink, `'`)) {
			cleanLink = cleanLink[1 : len(cleanLink)-1]
		}
		
		// Try unescaping first (handles most terminal input cases)
		unescapedPath := unescapeFilePath(cleanLink)
		absPath, err = filepath.Abs(unescapedPath)
		if err == nil {
			file, err = os.Open(absPath)
		}
		
		// If that fails, try as-is (fallback case)
		if err != nil {
			absPath, err = filepath.Abs(cleanLink)
			if err == nil {
				file, err = os.Open(absPath)
			}
		}
		
		// If both methods fail, return error
		if err != nil {
			return nil, fmt.Errorf("failed to open local file: %v", err)
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

	// Save raw retrieved content for debug if needed
	if util.Debug(ctx) {
		// Read the response body to save it
		rawContent, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for debug: %v", err)
		}

		// Save raw retrieved content immediately
		timestamp := time.Now().Format("20060102150405")
		archiveDir := filepath.Join(util.BaseDir(), "archive")
		rawDebugPath := filepath.Join(archiveDir, fmt.Sprintf("%s_debug_retrieved.html", timestamp))
		err = os.WriteFile(rawDebugPath, rawContent, 0644)
		if err != nil {
			fmt.Printf("Warning: failed to save debug retrieved file: %v\n", err)
		}

		// Recreate the response body for further processing
		resp.Body = io.NopCloser(strings.NewReader(string(rawContent)))
	}

	return resp, nil
}

// processes the retrieved content into a final article
func postProcessContent(ctx context.Context, resp *http.Response, excludeImages bool) (*readability.Article, string, string, int, int, string, error) {
	defer resp.Body.Close()

	article, filename, imageCount, err := postprocessing.ProcessArticleWithContext(ctx, resp, excludeImages)
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

	archivePath := filepath.Join(util.BaseDir(), "archive", filename)
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
