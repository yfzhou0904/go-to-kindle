# Architecture

go-to-kindle is an interactive Go CLI that turns a web page or local source
into a self-contained HTML article, archives it locally, and emails it to a
Kindle address.

## Data Flow

```text
URL, file, webarchive, or clipboard
              |
              v
       Normalize input
              |
              v
       Retrieve content
              |
              v
   Parse, clean, and embed images
              |
              v
       Review article title
              |
              v
        Archive and email
```

The Bubble Tea model in `tui.go` owns this lifecycle. Slow operations run as
commands and return messages that advance the model to the next screen. The
non-UI orchestration in `retrieval.go` converts every source into an
`InputResult`, dispatches it to the appropriate processor, and saves the first
archive copy.

## Modules

- **Input and orchestration** (`input.go`, `retrieval.go`) normalize source
  types and connect retrieval, processing, and archive storage.
- **Web retrieval** (`retrieval/`) selects direct HTTP or Chromedp according to
  user options.
- **Webarchive decoding** (`internal/webarchive/`) extracts HTML and resolves
  bundled Safari resources.
- **Processing** (`postprocessing/`) extracts readable HTML, renders Markdown,
  embeds images, and creates safe filenames.
- **TUI** (`tui.go`) owns navigation, progress, metadata review, and delivery
  commands.
- **Storage** (`internal/repositories/`) renders the final HTML wrapper and
  writes archive files.
- **Delivery** (`main.go`, `mail/`) reconciles title changes and sends the HTML
  attachment over SMTP.
- **Configuration and utilities** (`config.go`, `util/`) own runtime settings,
  application paths, proxy detection, and debug context.

## Boundaries

`InputResult` is the contract between input retrieval and processing. Image
sources are abstracted behind `postprocessing.ImageResolver`, allowing normal
web content and webarchive resources to share the same processing pipeline.
The final artifact is a `readability.Article`; storage and delivery do not need
to know which input produced it.

See the [documentation index](README.md) for each module's detailed behavior.

