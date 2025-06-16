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
	_ "golang.org/x/image/webp" // Register WebP format
)

const (
	maxImageDimension = 300
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

	// Resize maintaining aspect ratio
	resizedImg := resize.Thumbnail(uint(maxDim), uint(maxDim), img, resize.Lanczos3)

	// Determine output format and encode
	var buf bytes.Buffer
	var outputFormat string

	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
		outputFormat = "jpeg"
	case "png":
		err = png.Encode(&buf, resizedImg)
		outputFormat = "png"
	case "gif":
		err = gif.Encode(&buf, resizedImg, nil)
		outputFormat = "gif"
	case "webp":
		// Convert WebP to JPEG since we can't encode WebP
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
		outputFormat = "jpeg"
	default:
		// Default to JPEG for unknown formats
		err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
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
func processURLImageData(src string, baseURL *url.URL) (string, error) {
	// Download the image
	imageData, _, err := downloadImage(src, baseURL)
	if err != nil {
		return "", err
	}

	// Process the image data
	return processImageData(imageData)
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

		var dataURL string
		var err error

		if strings.HasPrefix(src, "data:image/") {
			// Handle base64 images (works for both web and local)
			dataURL, err = processBase64ImageData(src)
		} else if baseURL != nil {
			// Handle URL images (only for web pages with valid baseURL)
			dataURL, err = processURLImageData(src, baseURL)
		} else {
			// For local files, remove non-base64 images
			s.Remove()
			return
		}

		if err == nil {
			// Update src attribute with processed image
			s.SetAttr("src", dataURL)
			// Remove size-related attributes
			removeImageAttributes(s)
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

		var dataURL string
		var err error

		if strings.HasPrefix(firstURL, "data:image/") {
			// Handle base64 images (works for both web and local)
			dataURL, err = processBase64ImageData(firstURL)
		} else if baseURL != nil {
			// Handle URL images (only for web pages with valid baseURL)
			dataURL, err = processURLImageData(firstURL, baseURL)
		} else {
			// For local files, remove non-base64 sources
			s.Remove()
			return
		}

		if err == nil {
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
