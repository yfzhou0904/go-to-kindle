package postprocessing

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
)

// ProcessArticle handles the complete post-processing pipeline for articles
func ProcessArticle(resp *http.Response, includeImages bool) (*readability.Article, string, int, error) {
	// Parse webpage using readability
	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse webpage: %v", err)
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
	processedArticle, imageCount, err := processContent(&article, resp.Request.URL, includeImages)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to post-process article: %v", err)
	}

	return processedArticle, filename, imageCount, nil
}

// processContent cleans up the article content by processing images and removing unwanted elements
func processContent(article *readability.Article, baseURL *url.URL, includeImages bool) (*readability.Article, int, error) {
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse content: %v", err)
	}

	imageCount := 0

	// Process images based on includeImages flag
	if includeImages {
		// Use unified functions that handle both web URLs and local files
		// For web URLs, baseURL will be valid; for local files, baseURL will be nil
		imageCount += processImageElements(contentDoc, baseURL)
		// Also process source elements in picture tags
		imageCount += processSourceElements(contentDoc, baseURL)
		// Remove figures and pictures that no longer contain images (empty after processing)
		contentDoc.Find("figure").Each(func(i int, s *goquery.Selection) {
			if s.Find("img").Length() == 0 {
				s.Remove()
			}
		})
		contentDoc.Find("picture").Each(func(i int, s *goquery.Selection) {
			if s.Find("img,source").Length() == 0 {
				s.Remove()
			}
		})
	} else {
		// Remove all images, figures, pictures, and sources
		contentDoc.Find("img,figure,picture,source").Remove()
	}

	// Remove other media and unwanted elements (but keep processed images)
	contentDoc.Find("svg").Remove()

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
