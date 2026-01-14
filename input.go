package main

import (
	"io"
	"net/url"

	"github.com/yfzhou0904/go-to-kindle/postprocessing"
)

// InputResult represents normalized input ready for processing.
type InputResult struct {
	Body     io.ReadCloser
	BaseURL  *url.URL
	Resolver postprocessing.ImageResolver
}
