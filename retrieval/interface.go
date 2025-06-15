package retrieval

import (
	"errors"
	"fmt"
	"io"
	"net/url"
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
	ScrapingBeeAPIKey string
	ForceScrapingBee  bool
}

// Chain represents a chain of retrieval methods with fallback support
type Chain struct {
	methods []Method
	config  Config
}

// NewChain creates a new retrieval chain
func NewChain(config Config) *Chain {
	chain := &Chain{config: config}

	// Add methods in order of preference
	if !config.ForceScrapingBee {
		chain.methods = append(chain.methods, NewVanillaMethod())
	}

	if config.ScrapingBeeAPIKey != "" {
		chain.methods = append(chain.methods, NewScrapingBeeMethod(config.ScrapingBeeAPIKey))
	}

	// Add vanilla as fallback if we started with ScrapingBee
	if config.ForceScrapingBee {
		chain.methods = append(chain.methods, NewVanillaMethod())
	}

	return chain
}

// Retrieve attempts to fetch content using the chain of methods
func (c *Chain) Retrieve(url *url.URL) *Result {
	return c.RetrieveWithCallback(url, nil)
}

// RetrieveWithCallback attempts to fetch content using the chain of methods, calling onMethodChange when switching methods
func (c *Chain) RetrieveWithCallback(url *url.URL, onMethodChange func(string)) *Result {
	var lastError error
	var fallbackErrors []error

	for _, method := range c.methods {
		if !method.CanHandle(url) {
			continue
		}

		// Notify about method change
		if onMethodChange != nil {
			onMethodChange(method.Name())
		}

		result := method.Retrieve(url)
		if result.Error != nil {
			var fallbackErr *FallbackError
			if errors.As(result.Error, &fallbackErr) {
				// This is a fallback error, try next method
				fallbackErrors = append(fallbackErrors, result.Error)
				continue
			}
			// Non-fallback error, stop trying
			lastError = result.Error
			break
		}

		// Success
		return result
	}

	// All methods failed
	if len(fallbackErrors) > 0 {
		return &Result{
			Error: fmt.Errorf("all retrieval methods failed for URL %s with fallback errors: %v", url.String(), fallbackErrors),
		}
	}

	if lastError != nil {
		return &Result{
			Error: fmt.Errorf("retrieval failed for URL %s: %v", url.String(), lastError),
		}
	}

	return &Result{
		Error: fmt.Errorf("no suitable retrieval method found for URL: %s", url.String()),
	}
}
