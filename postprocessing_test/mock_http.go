package postprocessing_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// MockRoundTripper implements http.RoundTripper to serve local images
type MockRoundTripper struct {
	testDataDir string
}

// NewMockRoundTripper creates a new mock HTTP transport for serving local test images
func NewMockRoundTripper(testDataDir string) *MockRoundTripper {
	return &MockRoundTripper{
		testDataDir: testDataDir,
	}
}

// RoundTrip implements http.RoundTripper interface
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Parse the requested URL
	requestPath := req.URL.Path

	// Handle query parameters - extract just the path
	if strings.Contains(requestPath, "?") {
		requestPath = strings.Split(requestPath, "?")[0]
	}

	// Map common test image paths to local files
	var localPath string
	switch {
	case strings.Contains(requestPath, "sample@2x.png"):
		localPath = filepath.Join(m.testDataDir, "test_images", "sample@2x.png")
	case strings.Contains(requestPath, "sample.png"):
		localPath = filepath.Join(m.testDataDir, "test_images", "sample.png")
	case strings.Contains(requestPath, "sample.jpg"):
		localPath = filepath.Join(m.testDataDir, "test_images", "sample.jpg")
	case strings.Contains(requestPath, "sample.gif"):
		localPath = filepath.Join(m.testDataDir, "test_images", "sample.gif")
	case strings.Contains(requestPath, "sample.webp"):
		localPath = filepath.Join(m.testDataDir, "test_images", "sample.webp")
	default:
		// Return 404 for unknown images
		return &http.Response{
			StatusCode: 404,
			Status:     "404 Not Found",
			Body:       io.NopCloser(strings.NewReader("Image not found")),
			Request:    req,
		}, nil
	}

	// Check if file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// If file doesn't exist, create a minimal valid image
		return m.createFallbackImage(req, requestPath)
	}

	// Read the local file
	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return &http.Response{
			StatusCode: 500,
			Status:     "500 Internal Server Error",
			Body:       io.NopCloser(strings.NewReader("Failed to read image")),
			Request:    req,
		}, nil
	}

	// Determine content type from file extension
	contentType := "image/jpeg" // default
	ext := strings.ToLower(filepath.Ext(localPath))
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	}

	// Create successful response
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(fileData)),
		Header:     make(http.Header),
		Request:    req,
	}
	resp.Header.Set("Content-Type", contentType)
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(fileData)))

	return resp, nil
}

// createFallbackImage creates a minimal valid PNG when test images don't exist
func (m *MockRoundTripper) createFallbackImage(req *http.Request, path string) (*http.Response, error) {
	// Minimal 1x1 PNG (same as used in the base64 test case)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR chunk length
		0x49, 0x48, 0x44, 0x52, // IHDR
		0x00, 0x00, 0x00, 0x01, // width: 1
		0x00, 0x00, 0x00, 0x01, // height: 1
		0x08, 0x02, 0x00, 0x00, 0x00, // bit depth, color type, compression, filter, interlace
		0x90, 0x77, 0x53, 0xDE, // CRC
		0x00, 0x00, 0x00, 0x0C, // IDAT chunk length
		0x49, 0x44, 0x41, 0x54, // IDAT
		0x08, 0x99, 0x01, 0x01, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x02, 0x00, 0x01, // image data
		0xE2, 0x21, 0xBC, 0x33, // CRC
		0x00, 0x00, 0x00, 0x00, // IEND chunk length
		0x49, 0x45, 0x4E, 0x44, // IEND
		0xAE, 0x42, 0x60, 0x82, // CRC
	}

	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(pngData)),
		Header:     make(http.Header),
		Request:    req,
	}
	resp.Header.Set("Content-Type", "image/png")
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(pngData)))

	return resp, nil
}

// CreateMockHTTPClient creates an HTTP client that uses the mock transport
func CreateMockHTTPClient(testDataDir string) *http.Client {
	return &http.Client{
		Transport: NewMockRoundTripper(testDataDir),
	}
}
