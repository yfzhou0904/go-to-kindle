# Development

Read the [architecture overview](architecture.md) before changing the project, then use the [documentation index](README.md) to find the relevant module contract.

## Project Layout

- `tui.go` coordinates the interactive workflow.
- `retrieval.go` normalizes inputs and connects retrieval to processing.
- `retrieval/` fetches web content directly or through Chrome.
- `postprocessing/` converts HTML or Markdown into Kindle-ready content.
- `internal/webarchive/` decodes Safari webarchives and their resources.
- `internal/repositories/` writes processed articles to the local archive.
- `mail/` sends archived HTML as an email attachment.
- `testdata/` contains stable fixtures shared by tests.

## Workflow

Build and test through the repository's Make targets:

```bash
make build
make test
```

During iteration, run the smallest relevant package or test first. Before submitting a Go change, run the full suite and format changed Go files with `gofmt`.

Keep tests beside the package they exercise and reusable fixtures in `testdata/`. Ordinary tests must not depend on the live network, clipboard, Chrome, SMTP, or the user's home directory; inject external operations and use `httptest` or temporary directories where practical. See [Testing](testing.md) for detailed boundaries.

## Documentation

Update documentation when a change alters a workflow, invariant, input type, configuration option, or module boundary. Keep the root README concise and move detailed user and technical guidance into `docs/`. Follow the [documentation guide](README.md), describe current behavior, and treat code as the source of truth.
