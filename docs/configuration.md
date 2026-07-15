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

On startup the application also prunes the archive directory of files older
than `archive.retention_days` (see [Settings](#settings)). The sweep is
throttled to run at most once every 24 hours using a `~/.go-to-kindle/last_cleanup`
marker file, so frequent launches incur no repeated filesystem work. Cleanup is
best-effort: failures are logged but never block the application.

## Settings

- `email.smtp_server`, `port`, `from`, `password`, and `to` configure SMTP
  delivery.
- `browser.chrome_path` selects a Chrome or Chromium executable for headless
  retrieval.
- `archive.retention_days` sets how long processed articles are retained in the
  archive directory before automatic cleanup. It defaults to `365`; setting it
  to `0` keeps files permanently. The default is applied even when an existing
  config omits the `[archive]` section, so an explicit `0` is required to
  disable cleanup.
- The `--debug` CLI flag preserves intermediate retrieved or processed HTML in
  the archive directory.

Email credentials are local secrets and should never be committed. See
[retrieval.md](retrieval.md) for browser behavior and [delivery.md](delivery.md)
for SMTP usage.
