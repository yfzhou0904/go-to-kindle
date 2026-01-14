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
	return ProcessArticleWithResolver(resp, excludeImages, NewNetworkImageResolver(nil))
}

// ProcessArticleWithResolver handles the complete post-processing pipeline for articles with a resolver.
func ProcessArticleWithResolver(resp *http.Response, excludeImages bool, resolver ImageResolver) (*readability.Article, string, int, error) {
	ctx := context.Background()
	return ProcessArticleWithContext(ctx, resp, excludeImages, resolver)
}

// handles the complete post-processing pipeline with context support
func ProcessArticleWithContext(ctx context.Context, resp *http.Response, excludeImages bool, resolver ImageResolver) (*readability.Article, string, int, error) {
	// Parse webpage using readability
	article, err := readability.FromReader(resp.Body, resp.Request.URL)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse webpage: %v", err)
	}

	// If the input came from a webarchive, re-inline images after readability.
	if waResolver, ok := resolver.(*WebarchiveImageResolver); ok {
		inlined, err := webarchive.InlineImages([]byte(article.Content), resp.Request.URL, waResolver.resources)
		if err != nil {
			return nil, "", 0, fmt.Errorf("failed to inline webarchive images: %v", err)
		}
		article.Content = string(inlined)
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
	processedArticle, imageCount, err := processContent(&article, resp.Request.URL, excludeImages, resolver)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to post-process article: %v", err)
	}

	return processedArticle, filename, imageCount, nil
}

// ImageResolver resolves an image source URL into a data URL.
type ImageResolver interface {
	ResolveImage(src string, baseURL *url.URL) (string, bool, error)
}

// NetworkImageResolver downloads and processes images over the network.
type NetworkImageResolver struct {
	client *http.Client
}

// NewNetworkImageResolver creates a network-backed resolver.
func NewNetworkImageResolver(client *http.Client) *NetworkImageResolver {
	return &NetworkImageResolver{client: client}
}

// ResolveImage downloads and encodes an image as a data URL.
func (r *NetworkImageResolver) ResolveImage(src string, baseURL *url.URL) (string, bool, error) {
	data, _, err := downloadImage(src, baseURL, r.client)
	if err != nil {
		return "", false, err
	}
	dataURL, err := processImageData(data)
	if err != nil {
		return "", false, err
	}
	return dataURL, true, nil
}

// WebarchiveImageResolver resolves images from embedded webarchive resources.
type WebarchiveImageResolver struct {
	resources map[string]webarchive.Resource
	fallback  ImageResolver
}

// NewWebarchiveImageResolver creates a resolver backed by webarchive resources.
func NewWebarchiveImageResolver(resources map[string]webarchive.Resource) *WebarchiveImageResolver {
	return &WebarchiveImageResolver{resources: resources}
}

// WithFallback enables network fetching when a webarchive lacks a resource.
func (r *WebarchiveImageResolver) WithFallback(fallback ImageResolver) *WebarchiveImageResolver {
	r.fallback = fallback
	return r
}

// ResolveImage resolves an image using embedded webarchive resources.
func (r *WebarchiveImageResolver) ResolveImage(src string, baseURL *url.URL) (string, bool, error) {
	dataURL, ok := webarchive.ResolveImageDataURL(src, baseURL, r.resources)
	if !ok {
		if r.fallback != nil {
			return r.fallback.ResolveImage(src, baseURL)
		}
		return "", false, nil
	}
	processed, err := processBase64ImageData(dataURL)
	if err != nil {
		return "", false, err
	}
	return processed, true, nil
}

// processContent cleans up the article content by processing images and removing unwanted elements
func processContent(article *readability.Article, baseURL *url.URL, excludeImages bool, resolver ImageResolver) (*readability.Article, int, error) {
	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse content: %v", err)
	}

	imageCount := 0

	// Process images based on excludeImages flag
	if !excludeImages {
		imageCount += processPictureElements(contentDoc, baseURL, resolver)
		imageCount += processImageElements(contentDoc, baseURL, resolver)
		imageCount += processLoneSourceElements(contentDoc, baseURL, resolver)

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
