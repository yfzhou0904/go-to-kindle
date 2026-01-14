package postprocessing

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/yfzhou0904/go-to-kindle/internal/webarchive"
	"github.com/yfzhou0904/go-to-kindle/util"
)

// ProcessArticle handles the complete post-processing pipeline for articles
func ProcessArticle(resp *http.Response, excludeImages bool) (*readability.Article, string, int, error) {
	return ProcessArticleWithClient(resp, excludeImages, nil)
}

// ProcessArticleWithClient handles the complete post-processing pipeline for articles with custom HTTP client
func ProcessArticleWithClient(resp *http.Response, excludeImages bool, client *http.Client) (*readability.Article, string, int, error) {
	ctx := context.Background()
	return ProcessArticleWithContext(ctx, resp, excludeImages, client)
}

// handles the complete post-processing pipeline with context support
func ProcessArticleWithContext(ctx context.Context, resp *http.Response, excludeImages bool, client *http.Client) (*readability.Article, string, int, error) {
	// Parse webpage using readability
	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse webpage: %v", err)
	}

	// If the input came from a webarchive, re-inline images after readability.
	if baseURL, resources, ok := archiveFromContext(ctx, resp); ok {
		inlined, err := webarchive.InlineImagesInHTML(article.Content, baseURL, resources)
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to inline webarchive images: %v", err)
		}
		article.Content = inlined
	}

	// Save readability-parsed content for debug if needed
	if util.Debug(ctx) {
		timestamp := time.Now().Format("20060102150405")
		archiveDir := filepath.Join(util.BaseDir(), "archive")
		parsedDebugPath := filepath.Join(archiveDir, fmt.Sprintf("%s_debug_parsed.html", timestamp))
		err = os.WriteFile(parsedDebugPath, []byte(article.Content), 0644)
		if err != nil {
			fmt.Printf("Warning: failed to save debug parsed file: %v\n", err)
		}
	}

	// Generate filename from title or path
	var filename string
	if strings.HasPrefix(resp.Request.URL.String(), "http") {
		filename = TitleToFilename(article.Title)
	} else {
		// For local files, extract filename from path
		title := filepath.Base(resp.Request.URL.Path)
		title = strings.TrimSuffix(title, filepath.Ext(title))
		filename = TitleToFilename(title)
	}

	// Post-process the article content
	processedArticle, imageCount, err := processContent(&article, resp.Request.URL, excludeImages, client)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to post-process article: %v", err)
	}

	return processedArticle, filename, imageCount, nil
}

func archiveFromContext(ctx context.Context, resp *http.Response) (*url.URL, map[string]webarchive.Resource, bool) {
	if baseURL, resources, ok := webarchive.GetArchive(ctx); ok {
		return baseURL, resources, true
	}
	if resp != nil && resp.Request != nil {
		if baseURL, resources, ok := webarchive.GetArchive(resp.Request.Context()); ok {
			return baseURL, resources, true
		}
	}
	return nil, nil, false
}

// processContent cleans up the article content by processing images and removing unwanted elements
func processContent(article *readability.Article, baseURL *url.URL, excludeImages bool, client *http.Client) (*readability.Article, int, error) {
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse content: %v", err)
	}

	imageCount := 0

	// Process images based on excludeImages flag
	if !excludeImages {
		imageCount += processPictureElements(contentDoc, baseURL, client)
		imageCount += processImageElements(contentDoc, baseURL, client)
		imageCount += processLoneSourceElements(contentDoc, baseURL, client)

		contentDoc.Find("source").Remove()
		contentDoc.Find("picture").Remove()
		contentDoc.Find("figure").Each(func(i int, s *goquery.Selection) {
			if s.Find("img").Length() == 0 {
				s.Remove()
			}
		})
	} else {
		contentDoc.Find("img,figure,picture,source").Remove()
	}

	// Remove other media and unwanted elements (but keep processed images)
	contentDoc.Find("svg").Remove()

	// Remove <a> tags but keep their contents (text, images, etc.)
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		html, _ := s.Html()
		s.ReplaceWithHtml(html)
	})

	article.Content, err = contentDoc.Find("body").Html()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract content: %v", err)
	}

	return article, imageCount, nil
}

// TitleToFilename replaces problematic characters in page title to give a generally valid filename
func TitleToFilename(title string) string {
	filename := strings.ReplaceAll(title, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	return filename + ".html"
}
