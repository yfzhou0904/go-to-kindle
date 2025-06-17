# Go to Kindle

An interactive CLI tool that fetches web articles or local HTML files, processes them for readability, and emails them to your Kindle device.

## Architecture

### Core Components

- **Interactive TUI** (`tui.go`): Bubbletea-based terminal interface with 5 screen states
- **Web Retrieval** (`retrieval/`): Multi-tier fetching system with ScrapingBee fallback
- **Post-Processing** (`postprocessing/`): Content cleaning, image processing, readability extraction
- **Email Delivery** (`mail/mail.go`): SMTP email with HTML attachment to Kindle
- **Configuration** (`config.go`): TOML-based email and ScrapingBee settings management

### Workflow

1. **Input Screen**: User enters URL/file path, toggles image inclusion and ScrapingBee options
2. **Retrieval Screen**: Multi-tier content fetching (web URLs or local files)
3. **Post-Processing Screen**: Content extraction, readability processing, image processing
4. **Edit Screen**: Shows metadata (language, word count, image count, filename) with editable title
5. **Completion Screen**: Confirms successful email delivery or shows errors

### Key Features

- **Content Sources**: Web URLs (http/https) and local HTML files
- **Multi-tier Retrieval**: Direct HTTP with ScrapingBee fallback for blocked sites
- **Readability Processing**: Uses go-shiori/go-readability for clean text extraction
- **Image Processing**: Optional inclusion with automatic resizing and base64 embedding
- **Language Detection**: Supports Chinese and English with appropriate word counting
- **Content Cleaning**: Removes ads, navigation, and unwanted elements while preserving images
- **Email Formatting**: Generates optimized HTML files with embedded images for Kindle

### Technical Details

- **UI Framework**: Bubbletea (The Elm Architecture pattern)
- **UI Components**: Bubbles (textinput, spinner) + Lipgloss (styling)
- **Retrieval Chain**: Configurable multi-tier system (HTTP → ScrapingBee)
- **Image Processing**: Go image libraries with resize and base64 encoding
- **Async Operations**: Non-blocking commands for web requests and email sending
- **State Management**: Immutable updates through Update/View cycle
- **Error Handling**: User-friendly error messages with graceful fallbacks

### Configuration

Settings stored in `~/.go-to-kindle/config.toml`:
- **Email**: SMTP server details (server, port, credentials), From/To addresses
- **ScrapingBee**: API key for premium retrieval service
- Auto-generated on first run with example values

### File Storage

Articles archived in `~/.go-to-kindle/archive/` as HTML files with sanitized filenames.

### Dependencies

- **Core**: go-shiori/go-readability, PuerkitoBio/goquery, abadojack/whatlanggo
- **TUI**: charmbracelet/bubbletea, charmbracelet/bubbles, charmbracelet/lipgloss
- **Images**: nfnt/resize for image processing, built-in image/* for format support
- **HTTP**: Built-in net/http with custom ScrapingBee integration
- **Email**: Built-in net/smtp with TLS support
- **Config**: BurntSushi/toml

### Build
`go build -o bin/go-to-kindle` or simply run `make` (same thing)

### Configuration Example
```toml
[Email]
SMTPServer = "smtp.gmail.com"
Port = 587
From = "your-email@gmail.com"
Password = "your-app-password"
To = "your-kindle@kindle.com"

[ScrapingBee]
APIKey = "your-scrapingbee-api-key"  # Optional, for premium retrieval
```

The tool launches directly into the interactive interface - no command-line arguments needed.

### Testing

To test the workflow:
1. Launch the program
2. Enter a news article URL or local HTML file path
3. Toggle "Include Images" to test image processing
4. Toggle "Force ScrapingBee" to test premium retrieval (requires API key)
5. Review processed article metadata (language, word count, image count)
6. Edit the title if desired
7. Confirm to send to Kindle (requires email configuration)

### Retrieval Architecture

```
Input URL/File
      |
      v
┌─────────────┐
│ Retrieval   │  ┌──> Direct HTTP ───┐
│ Screen      │──┤                   ├──> HTTP Response
│ retrieval/  │  └──> ScrapingBee ───┘      |
└─────────────┘         (fallback)          v
                                    ┌───────────────┐
                                    │ Post-Process  │
                                    │ Screen        │
                                    │ postprocess/  │──> Clean Content
                                    │ • Readability │
                                    │ • Images      │──> Resized Images
                                    │ • Cleanup     │
                                    └───────────────┘
```

Use Ctrl+C to quit at any time.

# scrapingbee-integration
ScrapingBee is used as a fallback retrieval method when direct HTTP requests fail or are blocked by anti-bot measures. The integration:
- Requires API key configuration in config.toml
- Can be forced via UI toggle for testing or difficult sites
- Handles JavaScript-rendered content and bypasses Cloudflare
- Is automatically attempted when direct requests return blocked responses

# image-processing-pipeline
The unified image processing system handles both web URLs and local files:
- Base64 images: Processed directly from data URLs (works for both web and local)
- URL images: Downloaded, resized, and converted to base64 (web only, requires baseURL)
- Local files: Only base64 images are preserved, URL references removed
- All images resized to 300px max dimension maintaining aspect ratio
- Embedded as data URLs for Kindle compatibility
