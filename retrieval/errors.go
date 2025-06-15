package retrieval

import "fmt"

// FallbackError indicates that this retrieval method failed and fallback should be attempted
type FallbackError struct {
	Method string // "Vanilla HTTP" or "ScrapingBee"
	Reason string // Human-readable reason
	Err    error  // Wrapped underlying error
}

// Error implements the error interface
func (e *FallbackError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s failed (%s): %v", e.Method, e.Reason, e.Err)
	}
	return fmt.Sprintf("%s failed: %s", e.Method, e.Reason)
}

// Unwrap returns the wrapped error for error unwrapping
func (e *FallbackError) Unwrap() error {
	return e.Err
}

// NewFallbackError creates a new FallbackError
func NewFallbackError(method, reason string, err error) *FallbackError {
	return &FallbackError{
		Method: method,
		Reason: reason,
		Err:    err,
	}
}
