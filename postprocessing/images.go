package postprocessing

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/nfnt/resize"
)

// downloadImage downloads an image from a URL with timeout
func downloadImage(imageURL string, baseURL *url.URL) ([]byte, string, error) {
	// Resolve relative URLs
	parsedURL, err := url.Parse(imageURL)
	if err != nil {
		return nil, "", fmt.Errorf("invalid image URL: %v", err)
	}

	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(parsedURL.String())
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
		default:
			contentType = "image/jpeg" // default fallback
		}
	}

	return buf.Bytes(), contentType, nil
}

// resizeImageBytes resizes image data maintaining aspect ratio with max dimension
func resizeImageBytes(data []byte, maxDim int) ([]byte, error) {
	// Decode image
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// Resize maintaining aspect ratio
	resizedImg := resize.Thumbnail(uint(maxDim), uint(maxDim), img, resize.Lanczos3)

	// Encode back to bytes
	var buf bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
	case "png":
		err = png.Encode(&buf, resizedImg)
	case "gif":
		err = gif.Encode(&buf, resizedImg, nil)
	default:
		// Default to JPEG for unknown formats
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode resized image: %v", err)
	}

	return buf.Bytes(), nil
}

// imageToBase64DataURL converts image data to base64 data URL
func imageToBase64DataURL(data []byte, contentType string) string {
	base64Data := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", contentType, base64Data)
}

// processImageElements processes all img elements in the document (unified for web and local)
func processImageElements(doc *goquery.Document, baseURL *url.URL) int {
	processedCount := 0
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			// Remove img tags without src
			s.Remove()
			return
		}

		var processed bool
		if strings.HasPrefix(src, "data:image/") {
			// Handle base64 images (works for both web and local)
			processed = processBase64Image(s, src)
		} else if baseURL != nil {
			// Handle URL images (only for web pages with valid baseURL)
			processed = processURLImage(s, src, baseURL)
		} else {
			// For local files, remove non-base64 images
			processed = false
		}

		if processed {
			processedCount++
		} else {
			// Remove images that couldn't be processed
			s.Remove()
		}
	})

	return processedCount
}

// processSourceElements processes source elements in picture tags (unified for web and local)
func processSourceElements(doc *goquery.Document, baseURL *url.URL) int {
	processedCount := 0
	doc.Find("source").Each(func(i int, s *goquery.Selection) {
		srcset, exists := s.Attr("srcset")
		if !exists || srcset == "" {
			// Try src attribute as fallback
			src, srcExists := s.Attr("src")
			if !srcExists || src == "" {
				s.Remove()
				return
			}
			srcset = src
		}

		// Process the first URL from srcset (usually the main one)
		urls := strings.Split(srcset, ",")
		if len(urls) == 0 {
			s.Remove()
			return
		}

		// Extract the URL part (before any size descriptor)
		firstURL := strings.TrimSpace(strings.Split(urls[0], " ")[0])

		var processed bool
		var dataURL string

		if strings.HasPrefix(firstURL, "data:image/") {
			// Handle base64 images (works for both web and local)
			if processBase64Source(firstURL, &dataURL) {
				processed = true
			}
		} else if baseURL != nil {
			// Handle URL images (only for web pages with valid baseURL)
			if processURLSource(firstURL, baseURL, &dataURL) {
				processed = true
			}
		} else {
			// For local files, remove non-base64 sources
			processed = false
		}

		if processed {
			// Convert source to img element with processed image
			s.ReplaceWithHtml(fmt.Sprintf(`<img src="%s" />`, dataURL))
			processedCount++
		} else {
			// Remove sources that couldn't be processed
			s.Remove()
		}
	})

	return processedCount
}

// processBase64Source processes a base64 source element
func processBase64Source(src string, dataURL *string) bool {
	// Extract base64 data and content type
	parts := strings.SplitN(src, ",", 2)
	if len(parts) != 2 {
		return false
	}

	// Get content type from data URL
	header := parts[0]
	var contentType string
	if strings.Contains(header, "image/jpeg") {
		contentType = "image/jpeg"
	} else if strings.Contains(header, "image/png") {
		contentType = "image/png"
	} else if strings.Contains(header, "image/gif") {
		contentType = "image/gif"
	} else {
		contentType = "image/jpeg" // default
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	// Resize image
	resizedData, err := resizeImageBytes(imageData, 300)
	if err != nil {
		return false
	}

	// Convert back to base64 data URL
	*dataURL = imageToBase64DataURL(resizedData, contentType)
	return true
}

// processURLSource processes a URL-based source element
func processURLSource(src string, baseURL *url.URL, dataURL *string) bool {
	// Download and process image
	imageData, contentType, err := downloadImage(src, baseURL)
	if err != nil {
		return false
	}

	// Resize image
	resizedData, err := resizeImageBytes(imageData, 300)
	if err != nil {
		return false
	}

	// Convert to base64 data URL
	*dataURL = imageToBase64DataURL(resizedData, contentType)
	return true
}

// processBase64Image processes a base64 data URL image
func processBase64Image(s *goquery.Selection, src string) bool {
	// Extract base64 data and content type
	parts := strings.SplitN(src, ",", 2)
	if len(parts) != 2 {
		return false
	}

	// Get content type from data URL
	header := parts[0]
	var contentType string
	if strings.Contains(header, "image/jpeg") {
		contentType = "image/jpeg"
	} else if strings.Contains(header, "image/png") {
		contentType = "image/png"
	} else if strings.Contains(header, "image/gif") {
		contentType = "image/gif"
	} else {
		contentType = "image/jpeg" // default
	}

	// Decode base64 data
	imageData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	// Resize image
	resizedData, err := resizeImageBytes(imageData, 300)
	if err != nil {
		return false
	}

	// Convert back to base64 data URL
	dataURL := imageToBase64DataURL(resizedData, contentType)

	// Update src attribute
	s.SetAttr("src", dataURL)

	// Remove size-related attributes
	removeImageAttributes(s)

	return true
}

// processURLImage processes a URL-based image
func processURLImage(s *goquery.Selection, src string, baseURL *url.URL) bool {
	// Download and process image
	imageData, contentType, err := downloadImage(src, baseURL)
	if err != nil {
		return false
	}

	// Resize image
	resizedData, err := resizeImageBytes(imageData, 300)
	if err != nil {
		return false
	}

	// Convert to base64 data URL
	dataURL := imageToBase64DataURL(resizedData, contentType)

	// Update src attribute
	s.SetAttr("src", dataURL)

	// Remove size-related attributes
	removeImageAttributes(s)

	return true
}

// removeImageAttributes removes size and WordPress-specific attributes from images
func removeImageAttributes(s *goquery.Selection) {
	s.RemoveAttr("srcset")
	s.RemoveAttr("sizes")
	s.RemoveAttr("loading")
	s.RemoveAttr("width")
	s.RemoveAttr("height")

	// Remove WordPress-specific data attributes
	s.RemoveAttr("data-orig-size")
	s.RemoveAttr("data-medium-file")
	s.RemoveAttr("data-large-file")
}
