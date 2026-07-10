# Contributing

This project keeps user documentation in [README.md](README.md) and technical
documentation under [docs/](docs/README.md). Read
[docs/architecture.md](docs/architecture.md) before changing behavior, then
read the document for the module you are touching.

## Project Structure

- `tui.go` coordinates the interactive workflow.
- `retrieval.go` normalizes inputs and connects retrieval to processing.
- `retrieval/` fetches web content directly or through Chrome.
- `postprocessing/` converts HTML or Markdown into Kindle-ready content.
- `internal/webarchive/` decodes Safari webarchives and their resources.
- `internal/repositories/` writes processed articles to the local archive.
- `mail/` sends archived HTML as an email attachment.
- `testdata/` contains stable fixtures used by tests.

## Development

Build and test with the repository's Make targets:

```bash
make build
make test
```

During iteration, run the smallest relevant package or test first. Before
submitting a Go change, run the full test suite and format changed Go files with
`gofmt`.

## Tests

- Keep unit tests beside the package they exercise.
- Put reusable HTML, image, Markdown, or webarchive fixtures in `testdata/`.
- Do not make ordinary tests depend on the live network, clipboard, Chrome,
  SMTP, or the user's real home directory.
- Inject external operations where practical and use `httptest` or temporary
  directories for deterministic tests.

See [docs/testing.md](docs/testing.md) for the test boundaries and commands.

## Documentation

Update documentation when a change alters a documented workflow, invariant,
input type, configuration option, or module boundary. Follow
[docs/README.md](docs/README.md): keep documents concise, describe current
behavior, and treat code as the source of truth for implementation details.

