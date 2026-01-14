package postprocessing

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yfzhou0904/go-to-kindle/postprocessing_test"
)

// TestConfig represents the configuration for a single test case
type TestConfig struct {
	TestName           string   `json:"testName"`
	InputFile          string   `json:"inputFile"`
	BaseURL            string   `json:"baseURL"`
	ExcludeImages      bool     `json:"excludeImages"`
	ExpectedImageCount int      `json:"expectedImageCount"`
	MustNotContain     []string `json:"mustNotContain"`
	MustContain        []string `json:"mustContain"`
}

// loadTestConfig loads test configuration from JSON file
func loadTestConfig(configPath string) (*TestConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config TestConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// createMockResponse creates a mock HTTP response from HTML file
func createMockResponse(htmlPath string, baseURL string) (*http.Response, error) {
	file, err := os.Open(htmlPath)
	if err != nil {
		return nil, err
	}

	// Use provided baseURL or default
	if baseURL == "" {
		baseURL = "http://example.com/test-article"
	}
	mockURL, _ := url.Parse(baseURL)

	resp := &http.Response{
		StatusCode: 200,
		Body:       file,
		Request: &http.Request{
			URL: mockURL,
		},
	}

	return resp, nil
}

// TestProcessArticle_FromConfig runs tests based on configuration files
func TestProcessArticle_FromConfig(t *testing.T) {
	testDataDir := "../testdata/postprocessing"

	// Find all expected output files
	expectedOutputsDir := filepath.Join(testDataDir, "expected_outputs")
	entries, err := os.ReadDir(expectedOutputsDir)
	if err != nil {
		t.Fatalf("Failed to read expected outputs directory: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), "_expected.json") {
			continue
		}

		configPath := filepath.Join(expectedOutputsDir, entry.Name())
		t.Run(strings.TrimSuffix(entry.Name(), "_expected.json"), func(t *testing.T) {
			runSingleTest(t, configPath, testDataDir)
		})
	}
}

// runSingleTest executes a single test case based on configuration
func runSingleTest(t *testing.T, configPath, testDataDir string) {
	// Load test configuration
	config, err := loadTestConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}

	// Load HTML file
	htmlPath := filepath.Join(testDataDir, config.InputFile)
	resp, err := createMockResponse(htmlPath, config.BaseURL)
	if err != nil {
		t.Fatalf("Failed to create mock response: %v", err)
	}
	defer resp.Body.Close()

	// Duplicate the response body for reuse
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	resp.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Create mock HTTP client for image downloads
	mockClient := postprocessing_test.CreateMockHTTPClient(testDataDir)

	// Process the article with mock resolver
	resolver := NewNetworkImageResolver(mockClient)
	article, filename, imageCount, err := ProcessArticleWithResolver(resp, config.ExcludeImages, resolver)
	if err != nil {
		t.Fatalf("ProcessArticle failed: %v", err)
	}

	// Test filename generation
	if filename == "" {
		t.Error("Expected non-empty filename")
	}

	// Test image count
	if imageCount != config.ExpectedImageCount {
		t.Errorf("Expected %d images, got %d", config.ExpectedImageCount, imageCount)
	}

	// Test mustContain items are present
	for _, item := range config.MustContain {
		if !strings.Contains(article.Content, item) {
			t.Errorf("Required content not found: %q", item)
		}
	}

	// Test mustNotContain items are absent
	for _, item := range config.MustNotContain {
		if strings.Contains(article.Content, item) {
			t.Errorf("Forbidden content found: %q", item)
		}
	}

	// Additional validations
	if article.Title == "" {
		t.Error("Expected non-empty article title")
	}

	if article.Content == "" {
		t.Error("Expected non-empty article content")
	}

	// Log some debug info
	t.Logf("Test: %s", config.TestName)
	t.Logf("Title: %s", article.Title)
	t.Logf("Filename: %s", filename)
	t.Logf("Image Count: %d", imageCount)
	t.Logf("Content length: %d characters", len(article.Content))
}

// TestProcessArticle_Basic provides a simple test without external config
func TestProcessArticle_Basic(t *testing.T) {
	// Create a simple HTML response
	htmlContent := `
<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
	<h1>Test Title</h1>
	<p>This is a test paragraph.</p>
	<img src="data:image/jpeg;base64,/9j/4AAQSkZJRgABAQEAYABgAAD" alt="test" />
</body>
</html>`

	mockURL, _ := url.Parse("http://example.com/test")
	resp := &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(htmlContent)),
		Request:    &http.Request{URL: mockURL},
	}

	article, filename, imageCount, err := ProcessArticle(resp, true)
	if err != nil {
		t.Fatalf("ProcessArticle failed: %v", err)
	}

	if filename == "" {
		t.Error("Expected non-empty filename")
	}

	if !strings.Contains(article.Content, "Test Title") {
		t.Error("Expected title in content")
	}

	if !strings.Contains(article.Content, "test paragraph") {
		t.Error("Expected paragraph text in content")
	}

	t.Logf("Basic test passed - Title: %s, Images: %d", article.Title, imageCount)
}
