# Processing

The `postprocessing` package converts normalized input into a Kindle-ready
`readability.Article`. HTML and Markdown have different parsers but share
content cleanup, image handling, filename generation, and output storage.

## HTML

`ProcessArticleWithContext` uses go-readability to extract the article title,
text, main HTML, and declared publication and modification times. Ordinary web
links are removed from the final content.
Articles with fewer than 100 detected words are rejected by orchestration as a
likely extraction failure.

## Markdown

`ProcessMarkdownWithContext` uses Goldmark with GitHub Flavored Markdown and
the `goldmark-meta` extension. A leading YAML frontmatter block (delimited by
`---`) is parsed and stripped from the output instead of rendering as a stray
horizontal rule and heading. A frontmatter `title` or `name` becomes the title
when present; otherwise the first level-one heading is used, falling back to the
first paragraph or a default title. Markdown links are preserved, and short
Markdown is allowed because it is explicit user-provided content.

## Images

Image processing in `postprocessing/images.go` handles `img`, `picture`, source
sets, lazy-loading attributes, and base64 data URLs. Resolvers obtain bytes
from either the network or bundled webarchive resources. Supported images are
resized to a maximum dimension and embedded as data URLs unless the user
excludes images.

## Output Metadata

Orchestration detects English or Chinese for display and word counting. The
processor counts embedded images and derives a filesystem-safe `.html`
filename from the title. Filename truncation respects UTF-8 boundaries.

The HTML wrapper and Kindle-oriented CSS live in
`internal/repositories/file_repository.go`. See [delivery.md](delivery.md) for
how the generated artifact is archived and sent.
