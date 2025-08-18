# Go to Kindle

Interactive CLI tool that fetches web articles or local HTML files, processes them for readability, and emails them to your Kindle device.

## Architecture

### Core Components
- **TUI** (`tui.go`): Bubbletea-based interface with 5 screen states
- **Retrieval** (`retrieval.go` + `retrieval/`): Direct HTTP or headless browser fetching
- **Post-Processing** (`postprocessing/`): Content extraction and image processing
- **Email** (`mail/mail.go`): SMTP delivery with HTML attachments
- **Config** (`config.go`): TOML-based settings management
- **Utilities** (`util/`): Shared utilities including debug context and base directory helpers

### Data Flow
```
Input (URL/file) → Retrieval → Post-Processing → Edit Title → Email to Kindle
```

### TUI State Machine
```go
// tui.go:16-25
type screenState int
const (
    inputScreen screenState = iota      // URL input + options
    retrievalScreen                     // Fetching content  
    postProcessingScreen               // Extract + process content
    editScreen                         // Review metadata + edit title
    sendingScreen                      // Email delivery
    completionScreen                   // Success/error result
)
```

### Key Data Structures
```go
// tui.go:28-43 - Main TUI model
type model struct {
    state            screenState
    article          *readability.Article
    filename         string
    excludeImages   bool
    useChromedp    bool
    // ... UI components (inputs, spinner, etc.)
}

// retrieval/interface.go:12-16 - Retrieval result
type Result struct {
    Content io.ReadCloser
    URL     *url.URL  
    Error   error
}
```

### Retrieval Modes
```go
// retrieval/interface.go:44-62
// Selects either direct HTTP or Chromedp based on config
func NewChain(config Config) *Chain
```

The retrieval system picks a single method (HTTP or headless browser) and runs it.

### Retrieval Architecture

```
Input URL/File
      |
      v
┌─────────────┐
│ Retrieval   │  ┌──> Direct HTTP ───┐
│ Screen      │──┤                   ├──> HTTP Response
│ retrieval/  │  └──> Chromedp ─────┘      |
└─────────────┘                           v
                                    ┌───────────────┐
                                    │ Post-Process  │
                                    │ Screen        │
                                    │ postprocess/  │──> Clean Content
                                    │ • Readability │
                                    │ • Images      │──> Resized Images
                                    │ • Cleanup     │
                                    └───────────────┘
```

## Build & Run
```bash
go build -o bin/go-to-kindle  # or just `make`
./bin/go-to-kindle            # launches TUI
```

Configuration auto-generated at `~/.go-to-kindle/config.toml` on first run.
Articles archived in `~/.go-to-kindle/archive/`.
