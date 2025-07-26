package retrieval

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yfzhou0904/go-to-kindle/util"
)

// VanillaMethod implements vanilla HTTP GET requests
type VanillaMethod struct{}

// NewVanillaMethod creates a new vanilla HTTP method
func NewVanillaMethod() *VanillaMethod {
	return &VanillaMethod{}
}

// Name returns the name of this retrieval method
func (v *VanillaMethod) Name() string {
	return "Vanilla HTTP"
}

// CanHandle checks if this method can handle the given URL
func (v *VanillaMethod) CanHandle(url *url.URL) bool {
	return strings.HasPrefix(url.String(), "http://") || strings.HasPrefix(url.String(), "https://")
}


// Retrieve fetches content using vanilla HTTP GET with timeout and blocking detection
func (v *VanillaMethod) Retrieve(url *url.URL) *Result {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return &Result{Error: NewFallbackError(v.Name(), "request creation failed", err)}
	}

	// Set the User-Agent header to mimic a normal browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	// Create HTTP client with 5-second timeout and proxy support
	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: util.CreateHTTPTransportWithProxy(),
	}

	// Send the request using the client
	resp, err := client.Do(req)
	if err != nil {
		// Check if it's a timeout error
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
			return &Result{Error: NewFallbackError(v.Name(), "request timeout (>5s)", err)}
		}
		return &Result{Error: NewFallbackError(v.Name(), "network error", err)}
	}

	// Check for HTTP error status codes that might indicate blocking
	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		resp.Body.Close()
		return &Result{Error: NewFallbackError(v.Name(), "blocked by server", nil)}
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return &Result{Error: NewFallbackError(v.Name(), "HTTP error", nil)}
	}

	// Read content to check for blocking patterns
	contentBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return &Result{Error: NewFallbackError(v.Name(), "failed to read response", err)}
	}

	content := string(contentBytes)
	if isContentBlocked(content) {
		return &Result{Error: NewFallbackError(v.Name(), "content indicates blocking (Cloudflare/JS required)", nil)}
	}

	// Return successful result with content as ReadCloser
	contentReader := io.NopCloser(bytes.NewReader(contentBytes))
	return &Result{
		Content: contentReader,
		URL:     resp.Request.URL,
		Error:   nil,
	}
}
