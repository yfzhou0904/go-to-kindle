# go-to-kindle

📚 An interactive CLI tool that fetches web articles, processes them for readability, and sends them to your Kindle device.

## Features

- **Interactive Terminal UI**: Modern Bubbletea-based interface with guided workflow
- **Multiple Sources**: Supports web URLs (http/https) and local HTML files
- **Smart Processing**: Extracts readable content, removes media, detects language
- **Editable Titles**: Review and customize article titles before sending
- **Kindle Optimized**: Generates clean HTML files perfect for Kindle reading
- **Email Delivery**: Automatic SMTP delivery to your Kindle email address

The tool provides an intuitive 4-step workflow:
1. **URL Input**: Enter any web article URL or local file path
2. **Processing**: Automatic content extraction with progress indicator
3. **Title Edit**: Review article metadata and customize the title
4. **Delivery**: Confirmation of successful email delivery to Kindle

## Requirements

- Go 1.21 or newer
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
[Email]
smtp_server = "smtp.gmail.com"  # Your SMTP server
Port = 587                      # SMTP port (587 for TLS)
From = "your-email@gmail.com"   # Your email address
Password = "your-app-password"  # Email password or app password
To = "your-kindle@kindle.com"   # Your Kindle email address
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

### Workflow
1. **Enter URL or File Path**: Paste any web article URL or path to local HTML file
2. **Wait for Processing**: The tool fetches and processes the content automatically
3. **Review & Edit**: Check article metadata (language, word count) and edit the title if needed
4. **Send to Kindle**: Press Enter to email the article to your Kindle device

### Supported Content
- News articles from most websites
- Blog posts and long-form content
- Local HTML files
- Content in English and Chinese (with proper word counting)

### Keyboard Shortcuts
- **Enter**: Proceed to next step
- **Ctrl+C**: Quit at any time
- **Tab/Shift+Tab**: Navigate between input fields (when available)

## Troubleshooting

**Config file issues**: Delete `~/.go-to-kindle/config.toml` to recreate it
**SMTP errors**: Verify email credentials and server settings
**Cloudflare blocks**: Tool detects and reports when websites block the request
**Short articles**: Articles under 100 words are rejected (likely parsing failures)

## File Storage

Processed articles are saved to `~/.go-to-kindle/archive/` as HTML files for your records.
