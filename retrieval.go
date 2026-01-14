package main

import (
	"bytes"
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
	"github.com/yfzhou0904/go-to-kindle/internal/repositories"
	"github.com/yfzhou0904/go-to-kindle/internal/webarchive"
	"github.com/yfzhou0904/go-to-kindle/postprocessing"
	"github.com/yfzhou0904/go-to-kindle/retrieval"
	"github.com/yfzhou0904/go-to-kindle/util"
)

// retrieveContent handles both web URLs and local files, returning a normalized input result.
func retrieveContent(ctx context.Context, input string, useChromedp bool) (*InputResult, error) {
	link := input
	var result InputResult

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url - use retrieval chain
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %v", err)
		}

		// Create retrieval chain
		retrievalConfig := retrieval.Config{
			ChromeExecPath: Conf.Browser.ChromePath,
			UseChromedp:    useChromedp,
		}
		chain := retrieval.NewChain(retrievalConfig)

		// Attempt retrieval
		retrievalResult := chain.Retrieve(validURL)
		if retrievalResult.Error != nil {
			return nil, fmt.Errorf("failed to retrieve webpage: %v", retrievalResult.Error)
		}

		result = InputResult{
			Body:     retrievalResult.Content,
			BaseURL:  retrievalResult.URL,
			Resolver: postprocessing.NewNetworkImageResolver(nil),
		}
	} else {
		absPath, err := filepath.Abs(normalizeLocalPath(link))
		if err != nil {
			return nil, fmt.Errorf("failed to resolve local file path: %v", err)
		}

		if strings.EqualFold(filepath.Ext(absPath), ".webarchive") {
			raw, err := os.ReadFile(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read webarchive: %v", err)
			}
			htmlContent, baseURL, resources, err := webarchive.DecodeFile(raw)
			if err != nil {
				return nil, err
			}
			if baseURL == nil {
				baseURL = &url.URL{Path: link}
			}
			resolver := postprocessing.NewWebarchiveImageResolver(resources).
				WithFallback(postprocessing.NewNetworkImageResolver(nil))
			result = InputResult{
				Body:     io.NopCloser(bytes.NewReader(htmlContent)),
				BaseURL:  baseURL,
				Resolver: resolver,
			}
		} else {
			file, err := os.Open(absPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open local file: %v", err)
			}
			result = InputResult{
				Body: file,
				BaseURL: &url.URL{
					Path: link,
				},
				Resolver: postprocessing.NewNetworkImageResolver(nil),
			}
		}
	}

	if util.Debug(ctx) {
		rawContent, err := io.ReadAll(result.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body for debug: %v", err)
		}
		err = os.WriteFile(
			filepath.Join(filepath.Join(util.BaseDir(), "archive"),
				fmt.Sprintf("%s_debug_retrieved.html", time.Now().Format("20060102150405"))),
			rawContent, 0644)
		if err != nil {
			fmt.Printf("Warning: failed to save debug retrieved file: %v\n", err)
		}

		// Recreate the response body for further processing
		result.Body = io.NopCloser(strings.NewReader(string(rawContent)))
	}

	return &result, nil
}

// processes the retrieved content into a final article
func postProcessContent(ctx context.Context, input *InputResult, excludeImages bool) (*readability.Article, string, string, int, int, string, error) {
	defer input.Body.Close()

	resp := &http.Response{
		Body: input.Body,
		Request: &http.Request{
			URL: input.BaseURL,
		},
	}

	article, filename, imageCount, err := postprocessing.ProcessArticleWithContext(ctx, resp, excludeImages, input.Resolver)
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
	repo := repositories.NewLocalFileRepository()
	if err := repo.SaveArticle(article, archivePath); err != nil {
		return nil, "", "", 0, 0, "", fmt.Errorf("failed to write to archive file: %v", err)
	}

	return article, filename, lang.String(), wordCount, imageCount, archivePath, nil
}

// takes a file path string supplied by user dragging a file into terminal, or via copy-paste
// returns a normalized absolute path that can be used to open the file
func normalizeLocalPath(path string) string {
	// Clean input - remove leading/trailing whitespace
	clean := strings.TrimSpace(path)

	// Remove surrounding quotes (drag-drop adds these)
	if (strings.HasPrefix(clean, `"`) && strings.HasSuffix(clean, `"`)) ||
		(strings.HasPrefix(clean, `'`) && strings.HasSuffix(clean, `'`)) {
		clean = clean[1 : len(clean)-1]
	}

	// Unescape common terminal-escaped characters
	replacements := map[string]string{
		"\\ ": " ",
		"\\(": "(",
		"\\)": ")",
		"\\[": "[",
		"\\]": "]",
		"\\&": "&",
		"\\;": ";",
		"\\'": "'",
		"\\?": "?",
		"\\|": "|",
	}

	for escaped, unescaped := range replacements {
		clean = strings.ReplaceAll(clean, escaped, unescaped)
	}

	return clean
}
