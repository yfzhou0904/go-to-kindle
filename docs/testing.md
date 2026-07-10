# Testing

Tests should be deterministic, focused on observable behavior, and placed near
the package they exercise. The current suite covers input normalization, TUI
state transitions, Markdown and HTML processing, filenames, images, and Safari
webarchives.

## Commands

Run all tests:

```bash
make test
# equivalent to: go test ./...
```

During iteration, run one package or test:

```bash
go test ./postprocessing
go test ./... -run TestEnterSubmitsURLFromOptionRows
```

Build the application after changes that affect package integration:

```bash
make build
```

## Placement

- Keep package tests in `*_test.go` beside the implementation.
- Use root-package tests for orchestration and TUI behavior.
- Store reusable content and binary fixtures under `testdata/`.
- Use `t.TempDir` and `t.Setenv` for filesystem and environment isolation.

## External Boundaries

Ordinary tests must not require a live website, SMTP server, clipboard, Chrome
installation, or the user's actual home directory. Inject operations such as
clipboard access, use mock HTTP transports or `httptest`, and keep webarchive
coverage fixture-based. A future test that truly requires an external service
should be explicitly separated from the default suite.

When behavior changes, test the public outcome and the invariant that prevents
regression rather than duplicating internal implementation steps.

