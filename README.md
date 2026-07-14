# go-to-kindle

📚 An interactive CLI tool that fetches web articles, processes them for readability, and sends them to your Kindle device.

## Features

- **Interactive Terminal UI**: Modern Bubbletea-based interface with guided workflow
- **Multiple Sources**: Supports web URLs, local HTML/Markdown files, Safari .webarchive files, and Markdown from the clipboard
- **Smart Processing**: Extracts readable content, processes images, detects language
- **Robust Retrieval**: Optional headless browser mode for JS-heavy or blocked sites
- **Image Support**: Optional image inclusion with automatic resizing (300px max)
- **Editable Titles**: Review and customize article titles before sending
- **Date Context**: Optionally prepend published, updated, and sent dates to the Kindle article
- **Kindle Optimized**: Generates clean HTML files perfect for Kindle reading
- **Email Delivery**: Automatic SMTP delivery to your Kindle email address

## Workflow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   URL Input     │    │   🔍 Retrieval   │    │ ⚙️ Processing   │
│                 │    │                  │    │                 │
│ • Web URL       │───▶│ 1. Direct HTTP   │───▶│ • Readability   │
│ • Local file    │    │ 2. Headless      │    │ • Image resize  │
│ • Options       │    │    Browser       │    │ • Content clean │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
┌─────────────────┐    ┌──────────────────┐              │
│ Email Delivery  │◀───│ Title Editing    │◀─────────────┘
│                 │    │                  │
│ • SMTP send     │    │ • Review meta    │
│ • Archive copy  │    │ • Edit title     │
│ • Confirmation  │    │ • Word/img count │
└─────────────────┘    └──────────────────┘
```

The tool provides an intuitive workflow:
1. **Input & Options**: Enter a URL/file path or read Markdown from the clipboard
2. **Content Retrieval**: Fetch content using direct HTTP or the headless browser with progress indicator
3. **Content Processing**: Readability extraction, image processing, and content cleaning
4. **Review & Edit**: Check metadata, customize the title, and choose whether to add date context
5. **Delivery**: Email to Kindle with local archive copy

## Requirements

- Go 1.25 or newer
- Email account with SMTP access (Gmail, Outlook, etc.)
- Kindle device with email delivery enabled

## Installation

### Option 1: Install from source
```bash
git clone https://github.com/yfzhou0904/go-to-kindle.git
cd go-to-kindle
go build -o go-to-kindle
```

### Option 2: Go Install
```bash
go install github.com/yfzhou0904/go-to-kindle@latest
```

### Option 3: Using Make
```bash
make
```

## Configuration

On first run, the tool will create a configuration file at `~/.go-to-kindle/config.toml` and open it in your default editor. Configure your email settings:

```toml
[email]
smtp_server = "smtp.gmail.com"   # Your SMTP server
port = 465                       # Implicit TLS port
from = "your-email@gmail.com"    # Your email address
password = "your-app-password"   # Email password or app password
to = "your-kindle@kindle.com"    # Your Kindle email address
```

Optionally configure a custom Chrome/Chromium binary:

```toml
[browser]
chrome_path = "/usr/bin/chromium"
```

### Gmail Setup
1. Enable 2FA on your Google account
2. Generate an App Password for go-to-kindle
3. Use the App Password in the config file

### Kindle Setup
1. Go to Amazon's "Manage Your Content and Devices"
2. Find your Kindle's email address (e.g., `username_123@kindle.com`)
3. Add your sender email to the approved list

## Usage

Simply run the program to start the interactive interface:

```bash
./go-to-kindle
```

Or if installed globally:
```bash
go-to-kindle
```

### Input Options
- **URL or File Path**: Web articles, local HTML/Markdown files, or Safari `.webarchive` files
- **Clipboard Markdown**: Copy Markdown, focus **Read Markdown from clipboard**, and press Enter
- **Include Images**: Toggle to include resized images (300px max, base64 embedded)
- **Use Headless Browser**: Handle JS-heavy or protected sites (slower but more reliable)

### Processing Features
- **Retrieval Modes**: Direct HTTP or headless browser for JS-heavy or blocked pages
- **Content Extraction**: Uses go-readability for clean article text
- **Image Processing**: Downloads, resizes, and embeds images as base64 data URLs
- **Language Detection**: Supports English and Chinese with appropriate word counting
- **Content Cleaning**: Removes ads, navigation, and irrelevant elements
- **Date Context**: Adds declared publication/update dates and the send date in `YYYY/M/D` format; enabled by default on the review screen

### Supported Content
- News articles from most websites
- Blog posts and long-form content
- Local HTML files
- Local Markdown files and Markdown copied from LLM interfaces
- Safari .webarchive files
- Content in English and Chinese (with proper word counting)

### Keyboard Shortcuts
- **Enter**: Proceed to next step
- **Ctrl+C**: Quit at any time
- **Tab/↑↓**: Navigate between input fields and options
- **Space**: Toggle the focused checkbox, including date context on the review screen

## Development

Contributor setup and conventions are documented in
[CONTRIBUTING.md](CONTRIBUTING.md). For the internal design, start with the
[architecture overview](docs/architecture.md) and the
[documentation index](docs/README.md).

## Troubleshooting

**Config file issues**: Delete `~/.go-to-kindle/config.toml` to recreate it
**SMTP errors**: Verify email credentials and server settings
**Blocked websites**: Try enabling "Use Headless Browser" option for difficult sites
**Image issues**: Some email providers may reject large embedded images
**Short articles**: Articles under 100 words are rejected (likely parsing failures)

## File Storage

Processed articles are saved to `~/.go-to-kindle/archive/` as HTML files for your records. After a successful send, the archive matches the delivered article, including its optional date context line.
