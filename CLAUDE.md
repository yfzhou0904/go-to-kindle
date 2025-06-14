# Go to Kindle

An interactive CLI tool that fetches web articles or local HTML files, processes them for readability, and emails them to your Kindle device.

## Architecture

### Core Components

- **Interactive TUI** (`tui.go`): Bubbletea-based terminal interface with 4 screen states
- **Article Processing** (`main.go`): Web fetching, readability extraction, content cleaning  
- **Email Delivery** (`mail/mail.go`): SMTP email with HTML attachment to Kindle
- **Configuration** (`config.go`): TOML-based email settings management

### Workflow

1. **Input Screen**: User enters URL or local file path
2. **Processing Screen**: Fetches content, extracts readable text, removes media
3. **Edit Screen**: Shows article metadata (language, word count, filename) with editable title
4. **Completion Screen**: Confirms successful email delivery or shows errors

### Key Features

- **Content Sources**: Web URLs (http/https) and local HTML files
- **Readability Processing**: Uses go-shiori/go-readability for clean text extraction
- **Language Detection**: Supports Chinese and English with appropriate word counting
- **Content Cleaning**: Removes images, media, and links for Kindle compatibility
- **Cloudflare Detection**: Identifies and handles blocked requests
- **Email Formatting**: Generates proper HTML files with metadata for Kindle

### Technical Details

- **Framework**: Bubbletea (The Elm Architecture pattern)
- **UI Components**: Bubbles (textinput, spinner) + Lipgloss (styling)
- **Async Operations**: Non-blocking commands for web requests and email sending
- **State Management**: Immutable updates through Update/View cycle
- **Error Handling**: User-friendly error messages with graceful fallbacks

### Configuration

Email settings stored in `~/.go-to-kindle/config.toml`:
- SMTP server details (server, port, credentials)
- From/To email addresses
- Auto-generated on first run with example values

### File Storage

Articles archived in `~/.go-to-kindle/archive/` as HTML files with sanitized filenames.

### Dependencies

- **Core**: go-shiori/go-readability, PuerkitoBio/goquery, abadojack/whatlanggo
- **TUI**: charmbracelet/bubbletea, charmbracelet/bubbles, charmbracelet/lipgloss
- **Email**: Built-in net/smtp with TLS support
- **Config**: BurntSushi/toml

### Build & Run

```bash
go build -o go-to-kindle
./go-to-kindle
```

The tool launches directly into the interactive interface - no command-line arguments needed.

### Testing

To test the workflow:
1. Launch the program
2. Enter a news article URL or local HTML file path
3. Review the processed article metadata
4. Edit the title if desired
5. Confirm to send to Kindle (requires email configuration)

Use Ctrl+C to quit at any time.