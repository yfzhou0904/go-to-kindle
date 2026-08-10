package postprocessing

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"
)

// buildPage mimics the WeChat layout: a long article body plus a short block of
// surrounding chrome. bodyStyle is applied to the article container.
func buildPage(bodyStyle string) []byte {
	var paragraphs strings.Builder
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&paragraphs,
			"<p>This is a sufficiently long sentence of real article prose, number %d, "+
				"written so that readability scores this container as the main content.</p>", i)
	}
	return []byte(`<html><head><title>Hidden Article</title></head><body>
<div id="js_content" class="rich_media_content" style="` + bodyStyle + `">` + paragraphs.String() + `</div>
<div class="related"><p>Some unrelated recommendation text that is much shorter than the body.</p></div>
</body></html>`)
}

func TestParseArticleRecoversVisibilityHiddenBody(t *testing.T) {
	pageURL, _ := url.Parse("https://mp.weixin.qq.com/s/example")

	hidden, err := parseArticle(buildPage("visibility: hidden; opacity: 0; "), pageURL)
	if err != nil {
		t.Fatalf("parseArticle error: %v", err)
	}
	visible, err := parseArticle(buildPage(""), pageURL)
	if err != nil {
		t.Fatalf("parseArticle error: %v", err)
	}

	hiddenLen := utf8.RuneCountInString(hidden.TextContent)
	visibleLen := utf8.RuneCountInString(visible.TextContent)
	if hiddenLen < visibleLen {
		t.Fatalf("hidden page yielded %d runes, want at least the %d runes of the visible page",
			hiddenLen, visibleLen)
	}
	if !strings.Contains(hidden.TextContent, "number 19") {
		t.Fatalf("article body was not recovered: %q", hidden.TextContent)
	}
}

func TestStripVisibilityHidden(t *testing.T) {
	tests := []struct {
		name    string
		style   string
		want    string
		changed bool
	}{
		{"only declaration", "visibility: hidden;", "", true},
		{"leading declaration", "visibility: hidden; opacity: 0;", " opacity: 0;", true},
		{"trailing declaration", "color: red;visibility:hidden", "color: red;", true},
		{"important", "color: red; visibility: hidden !important", "color: red;", true},
		{"untouched", "visibility: visible; display: none", "visibility: visible; display: none", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := []byte(`<html><body><div id="x" style="` + tt.style + `">hi</div></body></html>`)
			out, changed, err := stripVisibilityHidden(body)
			if err != nil {
				t.Fatalf("stripVisibilityHidden error: %v", err)
			}
			if changed != tt.changed {
				t.Fatalf("changed = %v, want %v", changed, tt.changed)
			}
			if tt.want == "" && changed {
				if strings.Contains(string(out), "style=") {
					t.Fatalf("expected style attribute to be dropped, got %s", out)
				}
				return
			}
			if !strings.Contains(string(out), `style="`+tt.want+`"`) {
				t.Fatalf("got %s, want style %q", out, tt.want)
			}
		})
	}
}

func TestParseArticleKeepsNormalParseWhenNoGain(t *testing.T) {
	pageURL, _ := url.Parse("https://example.com/post")
	page := []byte(`<html><head><title>Normal</title></head><body>
<article><p>` + strings.Repeat("Visible article prose that readability will happily extract. ", 30) + `</p></article>
<div style="visibility: hidden"><p>tiny hidden tooltip</p></div>
</body></html>`)

	article, err := parseArticle(page, pageURL)
	if err != nil {
		t.Fatalf("parseArticle error: %v", err)
	}
	if strings.Contains(article.TextContent, "tiny hidden tooltip") {
		t.Fatalf("hidden tooltip should not be pulled in: %q", article.TextContent)
	}
}
