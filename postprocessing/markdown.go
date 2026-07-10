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
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// ProcessMarkdownWithContext renders Markdown without applying webpage readability extraction.
func ProcessMarkdownWithContext(_ context.Context, resp *http.Response, excludeImages bool, resolver ImageResolver) (*readability.Article, string, int, error) {
	source, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to read Markdown: %w", err)
	}

	md := newMarkdownParser()
	doc := md.Parser().Parse(text.NewReader(source))
	title := markdownTitle(doc, source)

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
	return goldmark.New(goldmark.WithExtensions(extension.GFM))
}

func markdownTitle(doc ast.Node, source []byte) string {
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
