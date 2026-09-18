# Usage

go-to-kindle needs Go 1.26 or newer, an email account with SMTP access, and a Kindle address enabled for email delivery.

## Installation

Install the latest version with Go:

```bash
go install github.com/yfzhou0904/go-to-kindle@latest
```

To build from source instead:

```bash
git clone https://github.com/yfzhou0904/go-to-kindle.git
cd go-to-kindle
make build
./bin/go-to-kindle
```

## Setup

On first run, the app creates `~/.go-to-kindle/config.toml` and opens it in the default editor. Configure the SMTP account that will send documents to the Kindle:

```toml
[email]
smtp_server = "smtp.gmail.com"
port = 465
from = "your-email@gmail.com"
password = "your-app-password"
to = "your-kindle@kindle.com"
```

Gmail accounts need two-factor authentication and an app password. In Amazon's **Manage Your Content and Devices**, find the Kindle email address and add the sending address to the approved list.

Headless retrieval can use a custom Chrome or Chromium executable:

```toml
[browser]
chrome_path = "/usr/bin/chromium"
```

See [Configuration](configuration.md) for the complete runtime behavior.

## Running

Run `go-to-kindle`, then provide a web URL or local HTML, Markdown, or Safari `.webarchive` path. Alternatively, copy Markdown, focus **Read Markdown from clipboard**, and press Enter.

The input screen can include resized, embedded images and can opt into the slower headless-browser mode for JavaScript-heavy or protected sites. The review screen shows the detected language, word and image counts, editable title, and default-on date context before sending.

### Controls

- **Enter** proceeds or activates the focused action.
- **Tab** and **↑/↓** move between fields and options.
- **Space** toggles the focused checkbox.
- **Escape** returns from review to the existing input.
- **Ctrl+C** exits.

## Output

Processed articles are stored under `~/.go-to-kindle/archive/`. After a successful send, the archived HTML matches the delivered attachment, including optional publication, update, and sent dates.

## Troubleshooting

- **Config file issues**: Delete `~/.go-to-kindle/config.toml` to recreate it.
- **SMTP errors**: Verify the credentials, server settings, Kindle address, and approved sender list.
- **Blocked websites**: Enable **Use Headless Browser**.
- **Image issues**: Disable images if the email provider rejects a large attachment.
- **Short articles**: HTML extractions under 100 words are rejected as likely parsing failures.

For implementation details, see [Input](input.md), [Retrieval](retrieval.md), [TUI](tui.md), and [Delivery](delivery.md).
