package postprocessing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/nfnt/resize"
	"github.com/yfzhou0904/go-to-kindle/util"
	_ "golang.org/x/image/webp" // Register WebP format
)

const (
	maxImageDimension = 300
)

// downloadImage downloads an image from a URL with timeout
func downloadImage(imageURL string, baseURL *url.URL, client *http.Client) ([]byte, string, error) {
	// Resolve relative URLs
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image URL: %v", err)
	}

	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Use provided client or create default one with timeout
	if client == nil {
		client = &http.Client{
			Transport: util.CreateHTTPTransportWithProxy(),
		}
	}

	// Create request with proper headers
	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
	}

	// Add user agent and accept headers that Next.js expects
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; go-to-kindle/1.0)")
	req.Header.Set("Accept", "image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("image download failed with status: %d", resp.StatusCode)
	}

	// Read image data
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %v", err)
	}

	// Get content type from header or URL
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback: guess from URL extension
		ext := strings.ToLower(filepath.Ext(parsedURL.Path))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		default:
			contentType = "image/jpeg" // default fallback
		}
	}

	return buf.Bytes(), contentType, nil
}

// resizeImageBytes resizes image data maintaining aspect ratio with max dimension
// Returns the resized image data and the format it was encoded as
func resizeImageBytes(data []byte, maxDim int) ([]byte, string, error) {
	// Try to detect format first
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image config: %v", err)
	}

	// Decode image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %v", err)
	}

	// Resize maintaining aspect ratio using Mitchell filter for better chart/text readability
	// Mitchell filter preserves sharp edges better than Lanczos3, which is crucial for charts
	resizedImg := resize.Thumbnail(uint(maxDim), uint(maxDim), img, resize.MitchellNetravali)

	// Determine output format and encode
	var buf bytes.Buffer
	var outputFormat string

	switch format {
	case "jpeg":
		// Use 95% quality for better compression while maintaining readability
		// 100% can actually introduce artifacts and larger file sizes
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 95})
		outputFormat = "jpeg"
	case "png":
		err = png.Encode(&buf, resizedImg)
		outputFormat = "png"
	case "gif":
		err = gif.Encode(&buf, resizedImg, nil)
		outputFormat = "gif"
	case "webp":
		// Convert WebP to JPEG since we can't encode WebP
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 95})
		outputFormat = "jpeg"
	default:
		// Default to JPEG for unknown formats
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 95})
		outputFormat = "jpeg"
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to encode resized image: %v", err)
	}

	return buf.Bytes(), outputFormat, nil
}

// imageToBase64DataURL converts image data to base64 data URL
func imageToBase64DataURL(data []byte, contentType string) string {
	base64Data := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
}

// parseBase64DataURL extracts the base64 data from a data URL
func parseBase64DataURL(dataURL string) ([]byte, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid base64 data URL format")
	}

	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 data: %v", err)
	}

	return imageData, nil
}

// processImageData resizes image data and converts it to a base64 data URL
func processImageData(imageData []byte) (string, error) {
	// Resize image
	resizedData, outputFormat, err := resizeImageBytes(imageData, maxImageDimension)
	if err != nil {
		return "", err
	}

	// Convert to base64 data URL using actual output format
	outputContentType := "image/" + outputFormat
	dataURL := imageToBase64DataURL(resizedData, outputContentType)

	return dataURL, nil
}

// processBase64ImageData processes a base64 data URL and returns the processed data URL
func processBase64ImageData(src string) (string, error) {
	// Parse base64 data from the data URL
	imageData, err := parseBase64DataURL(src)
	if err != nil {
		return "", err
	}

	// Process the image data
	return processImageData(imageData)
}

// processURLImageData downloads and processes a URL-based image and returns the processed data URL
// processURLImageData removed in favor of ImageResolver implementations.

// extractURLFromDataAttrs parses Substack-style data-attrs to find the original image URL
func extractURLFromDataAttrs(dataAttr string) string {
	dataAttr = html.UnescapeString(dataAttr)
	var attrs struct {
		Src            string `json:"src"`
		SrcNoWatermark string `json:"srcNoWatermark"`
	}
	if err := json.Unmarshal([]byte(dataAttr), &attrs); err != nil {
		return ""
	}
	if attrs.SrcNoWatermark != "" {
		return attrs.SrcNoWatermark
	}
	return attrs.Src
}

// processPictureElements collapses each <picture> into a single <img>
// selecting the best candidate from srcset or src attributes.
func processPictureElements(doc *goquery.Document, baseURL *url.URL, resolver ImageResolver) int {
	processed := 0

	// helper to choose largest width from a srcset
	pickBestFromSrcset := func(srcset string) string {
		best := ""
		bestW := -1
		for _, part := range strings.Split(srcset, ",") {
			part = strings.TrimSpace(part)
			fields := strings.Fields(part)
			if len(fields) == 0 {
				continue
			}
			u := fields[0]
			w := 0
			if len(fields) > 1 && strings.HasSuffix(fields[1], "w") {
				fmt.Sscanf(fields[1], "%dw", &w)
			}
			if w > bestW {
				bestW = w
				best = u
			}
			if best == "" {
				best = u
			}
		}
		return best
	}

	tryResolve := func(raw string) (string, bool) {
		if raw == "" {
			return "", false
		}
		if strings.HasPrefix(raw, "data:image/") {
			dataURL, err := processBase64ImageData(raw)
			return dataURL, err == nil
		}
		if resolver == nil {
			return "", false
		}
		dataURL, ok, err := resolver.ResolveImage(raw, baseURL)
		if err != nil || !ok {
			return "", false
		}
		return dataURL, true
	}

	doc.Find("picture").Each(func(i int, p *goquery.Selection) {
		img := p.Find("img").First()
		var alt string
		candidates := make([]string, 0, 6)

		if img.Length() > 0 {
			if v, ok := img.Attr("alt"); ok {
				alt = v
			}
			if da, ok := img.Attr("data-attrs"); ok {
				if u := extractURLFromDataAttrs(da); u != "" {
					candidates = append(candidates, u)
				}
			}
			if ss, ok := img.Attr("srcset"); ok && strings.TrimSpace(ss) != "" {
				if u := pickBestFromSrcset(ss); u != "" {
					candidates = append(candidates, u)
				}
			}
			if s, ok := img.Attr("src"); ok && s != "" {
				candidates = append(candidates, s)
			}
		}

		p.Find("source").Each(func(_ int, s *goquery.Selection) {
			if ss, ok := s.Attr("srcset"); ok && ss != "" {
				if u := pickBestFromSrcset(ss); u != "" {
					candidates = append(candidates, u)
				}
			}
			if src, ok := s.Attr("src"); ok && src != "" {
				candidates = append(candidates, src)
			}
		})

		if len(candidates) == 0 {
			p.Remove()
			return
		}

		var dataURL string
		resolved := false
		for _, cand := range candidates {
			if out, ok := tryResolve(cand); ok {
				dataURL = out
				resolved = true
				break
			}
		}
		if !resolved {
			p.Remove()
			return
		}

		repl := `<img src="` + dataURL + `"`
		if alt != "" {
			repl += ` alt="` + html.EscapeString(alt) + `"`
		}
		repl += ` data-processed="1" />`
		p.ReplaceWithHtml(repl)
		processed++
	})

	return processed
}

// processImageElements processes all img elements in the document (unified for web and local)
func processImageElements(doc *goquery.Document, baseURL *url.URL, resolver ImageResolver) int {
	processedCount := 0
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if v, ok := s.Attr("data-processed"); ok && v == "1" {
			removeImageAttributes(s)
			s.RemoveAttr("data-processed")
			return
		}

		src, exists := s.Attr("src")
		candidates := make([]string, 0, 2)
		if da, ok := s.Attr("data-attrs"); ok {
			if u := extractURLFromDataAttrs(da); u != "" {
				candidates = append(candidates, u)
			}
		}
		if exists && src != "" {
			candidates = append(candidates, src)
		}
		if len(candidates) == 0 {
			s.Remove()
			return
		}

		var dataURL string
		resolved := false
		for _, cand := range candidates {
			if strings.HasPrefix(cand, "data:image/") {
				out, err := processBase64ImageData(cand)
				if err == nil {
					dataURL = out
					resolved = true
					break
				}
				continue
			}
			if resolver == nil {
				continue
			}
			out, ok, err := resolver.ResolveImage(cand, baseURL)
			if err == nil && ok {
				dataURL = out
				resolved = true
				break
			}
		}
		if !resolved {
			s.Remove()
			return
		}

		s.SetAttr("src", dataURL)
		removeImageAttributes(s)
		processedCount++
	})

	return processedCount
}

// processLoneSourceElements processes <source> tags that are not inside <picture>.
func processLoneSourceElements(doc *goquery.Document, baseURL *url.URL, resolver ImageResolver) int {
	processed := 0
	doc.Find("source").Each(func(i int, s *goquery.Selection) {
		if s.ParentsFiltered("picture").Length() > 0 {
			return
		}

		srcset, exists := s.Attr("srcset")
		if !exists || strings.TrimSpace(srcset) == "" {
			if src, ok := s.Attr("src"); ok && strings.TrimSpace(src) != "" {
				srcset = src
			} else {
				s.Remove()
				return
			}
		}

		urls := strings.Split(srcset, ",")
		if len(urls) == 0 {
			s.Remove()
			return
		}

		firstURL := strings.TrimSpace(strings.Split(urls[0], " ")[0])

		var dataURL string
		var err error
		if strings.HasPrefix(firstURL, "data:image/") {
			dataURL, err = processBase64ImageData(firstURL)
		} else if resolver != nil {
			dataURL, _, err = resolver.ResolveImage(firstURL, baseURL)
		} else {
			s.Remove()
			return
		}

		if err == nil {
			s.ReplaceWithHtml(fmt.Sprintf(`<img src="%s" />`, dataURL))
			processed++
		} else {
			s.Remove()
		}
	})

	return processed
}

// removeImageAttributes removes size and WordPress-specific attributes from images
func removeImageAttributes(s *goquery.Selection) {
	s.RemoveAttr("srcset")
	s.RemoveAttr("sizes")
	s.RemoveAttr("loading")
	s.RemoveAttr("width")
	s.RemoveAttr("height")
	s.RemoveAttr("style")
	s.RemoveAttr("class")
	s.RemoveAttr("data-attrs")

	// Remove WordPress-specific data attributes
	s.RemoveAttr("data-orig-size")
	s.RemoveAttr("data-medium-file")
	s.RemoveAttr("data-large-file")
}
