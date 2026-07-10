# Configuration

Runtime configuration supplies SMTP credentials and an optional Chrome binary
path. `config.go` owns loading and initialization; `util.BaseDir` owns the
application data location.

## Lifecycle

On startup, the application reads `~/.go-to-kindle/config.toml`. If the file is
missing, it creates the application directory, writes the default configuration,
and opens the file in the user's editor for setup.

Processed and debug artifacts are written under
`~/.go-to-kindle/archive/`. `example_config.toml` shows the supported TOML
shape without containing usable credentials.

## Settings

- `email.smtp_server`, `port`, `from`, `password`, and `to` configure SMTP
  delivery.
- `browser.chrome_path` selects a Chrome or Chromium executable for headless
  retrieval.
- The `--debug` CLI flag preserves intermediate retrieved or processed HTML in
  the archive directory.

Email credentials are local secrets and should never be committed. See
[retrieval.md](retrieval.md) for browser behavior and [delivery.md](delivery.md)
for SMTP usage.
