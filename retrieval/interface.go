package retrieval

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Result represents the result of a content retrieval operation
type Result struct {
	Content io.ReadCloser
	URL     *url.URL
	Error   error
}

// Method represents a content retrieval method
type Method interface {
	// Name returns the name of the retrieval method
	Name() string

	// Retrieve fetches content from the given URL
	// Returns FallbackError if this method failed and fallback should be attempted
	Retrieve(url *url.URL) *Result

	// CanHandle checks if this method can handle the given URL
	CanHandle(url *url.URL) bool
}

// Config holds configuration for retrieval methods
type Config struct {
	ChromeExecPath string
	UseChromedp    bool
}

// Chain represents a chain of retrieval methods with fallback support
type Chain struct {
	methods []Method
	config  Config
}

// NewChain creates a new retrieval chain
func NewChain(config Config) *Chain {
	chain := &Chain{config: config}

	if config.UseChromedp {
		chain.methods = append(chain.methods, NewChromedpMethod(config.ChromeExecPath))
	} else {
		chain.methods = append(chain.methods, NewVanillaMethod())
	}

	return chain
}

// isContentBlocked checks if the content indicates blocking (shared utility function)
func isContentBlocked(content string) bool {
	blockedKeyElems := []string{
		`<div id="cf-error-details">`,
		`<title>Attention Required! | Cloudflare</title>`,
		`<title>Just a moment...</title>`,
		`<div class="cf-browser-verification">`,
		`<title>Access denied</title>`,
		`<title>Forbidden</title>`,
	}

	for _, blockedElem := range blockedKeyElems {
		if strings.Contains(content, blockedElem) {
			return true
		}
	}
	return false
}

// Retrieve attempts to fetch content using the chain of methods
func (c *Chain) Retrieve(url *url.URL) *Result {
	return c.RetrieveWithCallback(url, nil)
}

// RetrieveWithCallback attempts to fetch content using the chain of methods, calling onMethodChange for the selected method
func (c *Chain) RetrieveWithCallback(url *url.URL, onMethodChange func(string)) *Result {
	for _, method := range c.methods {
		if !method.CanHandle(url) {
			continue
		}

		if onMethodChange != nil {
			onMethodChange(method.Name())
		}

		return method.Retrieve(url)
	}

	return &Result{
		Error: fmt.Errorf("no suitable retrieval method found for URL: %s", url.String()),
	}
}
