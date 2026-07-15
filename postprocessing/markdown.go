package postprocessing

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ProcessMarkdownWithContext renders Markdown without applying webpage readability extraction.
func ProcessMarkdownWithContext(_ context.Context, resp *http.Response, excludeImages bool, resolver ImageResolver) (*readability.Article, string, int, error) {
	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to read Markdown: %w", err)
	}

	md := newMarkdownParser()
	pc := parser.NewContext()
	doc := md.Parser().Parse(text.NewReader(source), parser.WithContext(pc))
	title := markdownTitle(doc, source, meta.Get(pc))

	var rendered bytes.Buffer
	if err := md.Renderer().Render(&rendered, source, doc); err != nil {
		return nil, "", 0, fmt.Errorf("failed to render Markdown: %w", err)
	}

	parsed, err := goquery.NewDocumentFromReader(bytes.NewReader(rendered.Bytes()))
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to parse rendered Markdown: %w", err)
	}
	text := strings.TrimSpace(parsed.Text())
	article := &readability.Article{Title: title, Content: rendered.String(), TextContent: text}
	article, imageCount, err := processContent(article, resp.Request.URL, excludeImages, resolver, true)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to post-process Markdown: %w", err)
	}
	return article, TitleToFilename(title), imageCount, nil
}

func newMarkdownParser() goldmark.Markdown {
	// meta.Meta parses a leading YAML frontmatter block (delimited by ---) and
	// removes it from the rendered output instead of letting goldmark render it
	// as a stray <hr> plus setext heading.
	return goldmark.New(goldmark.WithExtensions(extension.GFM, meta.Meta))
}

// markdownTitle picks a document title, preferring a frontmatter title/name key
// (from parsed YAML metadata) before falling back to the first heading or
// paragraph in the body.
func markdownTitle(doc ast.Node, source []byte, metadata map[string]interface{}) string {
	if title := frontmatterTitle(metadata); title != "" {
		return title
	}

	var firstHeading string
	var firstText string
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if heading, ok := node.(*ast.Heading); ok && heading.Level == 1 && firstHeading == "" {
			firstHeading = strings.TrimSpace(string(heading.Text(source)))
		}
		if paragraph, ok := node.(*ast.Paragraph); ok && firstText == "" {
			firstText = strings.TrimSpace(string(paragraph.Text(source)))
		}
		return ast.WalkContinue, nil
	})
	if firstHeading != "" {
		return firstHeading
	}
	if firstText != "" {
		const maxTitleRunes = 100
		runes := []rune(firstText)
		if len(runes) > maxTitleRunes {
			return strings.TrimSpace(string(runes[:maxTitleRunes])) + "..."
		}
		return firstText
	}
	return "Markdown from Clipboard"
}

// frontmatterTitle returns the first non-empty string value among common
// title-bearing frontmatter keys, or "" if none is present.
func frontmatterTitle(metadata map[string]interface{}) string {
	for _, key := range []string{"title", "name"} {
		if value, ok := metadata[key].(string); ok {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}
