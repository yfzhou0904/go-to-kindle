package retrieval

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ScrapingBeeMethod implements ScrapingBee API requests
type ScrapingBeeMethod struct {
	apiKey string
}

// NewScrapingBeeMethod creates a new ScrapingBee method
func NewScrapingBeeMethod(apiKey string) *ScrapingBeeMethod {
	return &ScrapingBeeMethod{apiKey: apiKey}
}

// Name returns the name of this retrieval method
func (s *ScrapingBeeMethod) Name() string {
	return "ScrapingBee"
}

// CanHandle checks if this method can handle the given URL
func (s *ScrapingBeeMethod) CanHandle(url *url.URL) bool {
	return strings.HasPrefix(url.String(), "http://") || strings.HasPrefix(url.String(), "https://")
}


// Retrieve fetches content using ScrapingBee API with timeout and error detection
func (s *ScrapingBeeMethod) Retrieve(targetURL *url.URL) *Result {
	if s.apiKey == "" {
		return &Result{Error: NewFallbackError(s.Name(), "API key not configured", nil)}
	}

	// Build ScrapingBee API URL
	scrapingBeeURL := fmt.Sprintf("https://app.scrapingbee.com/api/v1/?api_key=%s&url=%s&render_js=true",
		s.apiKey, url.QueryEscape(targetURL.String()))

	// Create request
	req, err := http.NewRequest("GET", scrapingBeeURL, nil)
	if err != nil {
		return &Result{Error: NewFallbackError(s.Name(), "request creation failed", err)}
	}

	// Create HTTP client with longer timeout for ScrapingBee (JS rendering takes time)
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return &Result{Error: NewFallbackError(s.Name(), "API request timeout (>30s)", err)}
		}
		return &Result{Error: NewFallbackError(s.Name(), "API request failed", err)}
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode == 429 {
		return &Result{Error: NewFallbackError(s.Name(), "API rate limit exceeded", nil)}
	}

	if resp.StatusCode != http.StatusOK {
		return &Result{Error: NewFallbackError(s.Name(), fmt.Sprintf("API returned status %d", resp.StatusCode), nil)}
	}

	// Read response content to check for API errors
	contentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &Result{Error: NewFallbackError(s.Name(), "failed to read API response", err)}
	}

	content := string(contentBytes)
	if isContentBlocked(content) {
		return &Result{Error: NewFallbackError(s.Name(), "content indicates blocking (Cloudflare/JS required)", nil)}
	}

	// Return successful result
	contentReader := io.NopCloser(strings.NewReader(content))
	return &Result{
		Content: contentReader,
		URL:     targetURL, // Return original URL, not ScrapingBee URL
		Error:   nil,
	}
}
