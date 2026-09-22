package postprocessing

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
		// Some hand-written pages carry no <title>, no metadata and no single
		// <h1>, which leaves readability with nothing to work from. Fall back to
		// the URL, the way a browser labels such a tab.
		if strings.TrimSpace(article.Title) == "" {
			article.Title = TitleFromURL(resp.Request.URL)
		}
		filename = TitleToFilename(article.Title)
	} else {
		// For local files, extract filename from path
		title := filepath.Base(resp.Request.URL.Path)
		title = strings.TrimSuffix(title, filepath.Ext(title))
		filename = TitleToFilename(title)
	}

	// Post-process the article content
	processedArticle, imageCount, err := processContent(&article, resp.Request.URL, excludeImages, resolver, false)
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
func processContent(article *readability.Article, baseURL *url.URL, excludeImages bool, resolver ImageResolver, preserveLinks bool) (*readability.Article, int, error) {
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
		cleanInlineSVGs(contentDoc)
	} else {
		contentDoc.Find("img,figure,picture,source,svg").Remove()
	}

	// Remove other media and unwanted elements (but keep processed images)
	replaceEmbeddedMedia(contentDoc)

	// Remove <a> tags but keep their contents (text, images, etc.)
	if !preserveLinks {
		contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
			html, _ := s.Html()
			s.ReplaceWithHtml(html)
		})
	}

	article.Content, err = contentDoc.Find("body").Html()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to extract content: %v", err)
	}

	return article, imageCount, nil
}

// iconSVGMaxSize is the largest declared width and height, in pixels, at which
// an inline SVG is treated as an icon rather than article content.
const iconSVGMaxSize = 64

// cleanInlineSVGs keeps content SVGs such as diagrams, which Kindle renders in
// reflowable layout, and removes decorative icons. Kept SVGs are reduced to
// static drawing.
func cleanInlineSVGs(doc *goquery.Document) {
	doc.Find("svg").Each(func(i int, s *goquery.Selection) {
		if isDecorativeSVG(s) {
			s.Remove()
			return
		}
		s.Find("script,foreignObject,animate,animateMotion,animateTransform,set").Remove()
		s.Find("*").AddSelection(s).Each(func(i int, el *goquery.Selection) {
			for _, attr := range el.Nodes[0].Attr {
				if strings.HasPrefix(strings.ToLower(attr.Key), "on") {
					el.RemoveAttr(attr.Key)
				}
			}
		})
	})
}

func isDecorativeSVG(s *goquery.Selection) bool {
	if s.AttrOr("aria-hidden", "") == "true" || s.AttrOr("role", "") == "presentation" {
		return true
	}
	if s.ParentsFiltered("a,button").Length() > 0 {
		return true
	}
	width, height, ok := svgSize(s)
	return ok && width <= iconSVGMaxSize && height <= iconSVGMaxSize
}

// svgSize returns the declared width and height, falling back to the viewBox.
func svgSize(s *goquery.Selection) (float64, float64, bool) {
	width, wok := parseSVGLength(s.AttrOr("width", ""))
	height, hok := parseSVGLength(s.AttrOr("height", ""))
	if wok && hok {
		return width, height, true
	}
	fields := strings.FieldsFunc(s.AttrOr("viewBox", s.AttrOr("viewbox", "")), func(r rune) bool {
		return r == ' ' || r == ','
	})
	if len(fields) != 4 {
		return 0, 0, false
	}
	width, werr := strconv.ParseFloat(fields[2], 64)
	height, herr := strconv.ParseFloat(fields[3], 64)
	return width, height, werr == nil && herr == nil
}

// parseSVGLength parses unitless or pixel lengths; percentages and other units
// are not treated as sizes.
func parseSVGLength(v string) (float64, bool) {
	n, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(v), "px"), 64)
	return n, err == nil
}

// replaceEmbeddedMedia swaps video, audio, and embedded frames for a short text
// placeholder. Kindle cannot render them, and a <video> element makes Kindle's
// converter fall back to a fixed, non-reflowable layout (error E016).
func replaceEmbeddedMedia(doc *goquery.Document) {
	labels := map[string]string{"video": "Video", "audio": "Audio"}
	doc.Find("video,audio,iframe,embed,object").Each(func(i int, s *goquery.Selection) {
		label, ok := labels[goquery.NodeName(s)]
		if !ok {
			label = "Embedded content"
		}
		if title := strings.TrimSpace(s.AttrOr("title", "")); title != "" {
			label += ": " + title
		}
		s.ReplaceWithHtml("<p><em>[" + html.EscapeString(label) + "]</em></p>")
	})
}

const maxFilenameBaseBytes = 200

// TitleFromURL builds a human-readable stand-in title from a URL, mirroring how
// browsers label a tab for a page that declares no title: host plus path, with
// the scheme and any trailing slash dropped.
func TitleFromURL(u *url.URL) string {
	if u == nil {
		return ""
	}

	title := u.Host + u.Path
	title = strings.TrimSuffix(title, "/")
	if title == "" {
		title = u.String()
	}
	return title
}

// TitleToFilename replaces problematic characters in page title to give a generally valid filename.
func TitleToFilename(title string) string {
	if strings.TrimSpace(title) == "" {
		title = "untitled"
	}

	filename := strings.ReplaceAll(title, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	filename = truncateStringBytes(filename, maxFilenameBaseBytes)
	// URL-derived titles often already end in .html; don't double the suffix.
	if strings.HasSuffix(strings.ToLower(filename), ".html") {
		return filename
	}
	return filename + ".html"
}

func truncateStringBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	var b strings.Builder
	for _, r := range s {
		runeString := string(r)
		if b.Len()+len(runeString) > maxBytes {
			break
		}
		b.WriteString(runeString)
	}
	return b.String()
}
