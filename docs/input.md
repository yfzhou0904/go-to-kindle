# Input

Input handling converts URLs, local files, Safari webarchives, and clipboard
Markdown into one normalized value for the processing pipeline. The contract is
`InputResult` in `input.go`; orchestration lives in `retrieval.go`.

## Sources

- `http://` and `https://` inputs are fetched through the retrieval chain.
- `.md` and `.markdown` files are treated as Markdown.
- `.webarchive` files are decoded with their bundled resources.
- Other local files are treated as HTML.
- The clipboard action reads its contents as Markdown and rejects empty text.

Each result records its source label, input kind, readable body, base URL, and
image resolver. Processing uses the input kind to choose HTML readability or
Markdown rendering without re-detecting the source.

## Local Paths

Interactive paths are parsed as one POSIX shell token so quoted paths and
backslash-escaped spaces work. A path supplied as a CLI argument has already
been unescaped by the shell and is not parsed a second time. Relative paths are
resolved before opening the file.

## Invariants

- Callers own the returned body and processing closes it.
- Markdown inputs bypass the minimum article-length check.
- Webarchive inputs prefer bundled image resources and may fall back to the
  network.
- Debug mode may copy retrieved bytes to the archive directory, then restores
  the body for normal processing.

Related behavior is documented in [retrieval.md](retrieval.md) and
[processing.md](processing.md).

