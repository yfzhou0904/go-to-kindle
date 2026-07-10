# Retrieval

The `retrieval` package fetches web URLs and returns a response body plus the
effective URL. Local files and clipboard input do not pass through this
package; they are normalized directly in `retrieval.go`.

## Methods

`VanillaMethod` uses Go's HTTP client and is the default. `ChromedpMethod`
starts a configured Chrome or Chromium executable, waits for the document to
be ready, and returns the rendered page HTML.

`retrieval.NewChain` selects one method from runtime options: direct HTTP by
default or Chromedp when browser mode is enabled. It reports retrieval errors
through `retrieval.Result`; it does not automatically retry with the other
method.

## Browser Option

Headless-browser retrieval is opt-in from the TUI. It is intended for pages
whose useful content requires JavaScript or cannot be retrieved correctly with
plain HTTP. The Chrome binary path comes from configuration.

## Network Behavior

HTTP retrieval uses the proxy-aware transport in `util/proxy.go`. The effective
response URL becomes the base URL for resolving relative links and images during
processing.

See [input.md](input.md) for source selection and [processing.md](processing.md)
for how the retrieved body is consumed.
