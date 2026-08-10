package postprocessing

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	readability "github.com/go-shiori/go-readability"
)

// Some sites (WeChat's mp.weixin.qq.com being the common case) ship the article
// body with an inline `visibility: hidden` style and reveal it from JavaScript
// once the page has loaded. Readability drops such nodes as invisible, so plain
// HTTP retrieval extracts only the surrounding chrome while the same page saved
// as a webarchive (after scripts ran) parses fine.
var rxVisibilityHiddenDecl = regexp.MustCompile(`(?i)(^|;)\s*visibility\s*:\s*hidden\s*(?:!\s*important\s*)?(;|$)`)

// unhideGainFactor is how much more text the unhidden parse must yield before we
// prefer it over the normal parse. Requiring a clear win keeps pages that
// legitimately hide content from regressing.
const unhideGainFactor = 2

// parseArticle runs readability over body. If the normal parse looks like it lost
// the article to JS-revealed content, it retries with `visibility: hidden` inline
// styles stripped and keeps that result when it is substantially richer.
func parseArticle(body []byte, pageURL *url.URL) (readability.Article, error) {
	article, err := readability.FromReader(bytes.NewReader(body), pageURL)
	if err != nil {
		return article, err
	}

	unhidden, changed, err := stripVisibilityHidden(body)
	if err != nil || !changed {
		return article, nil
	}

	retry, err := readability.FromReader(bytes.NewReader(unhidden), pageURL)
	if err != nil {
		return article, nil
	}
	if utf8.RuneCountInString(retry.TextContent) > unhideGainFactor*utf8.RuneCountInString(article.TextContent) {
		return retry, nil
	}
	return article, nil
}

// stripVisibilityHidden removes `visibility: hidden` declarations from inline
// styles, reporting whether anything changed.
func stripVisibilityHidden(body []byte) ([]byte, bool, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}

	changed := false
	doc.Find("[style]").Each(func(_ int, s *goquery.Selection) {
		style, ok := s.Attr("style")
		if !ok || !rxVisibilityHiddenDecl.MatchString(style) {
			return
		}
		cleaned := rxVisibilityHiddenDecl.ReplaceAllString(style, "$1")
		if strings.TrimSpace(strings.Trim(cleaned, ";")) == "" {
			s.RemoveAttr("style")
		} else {
			s.SetAttr("style", cleaned)
		}
		changed = true
	})
	if !changed {
		return body, false, nil
	}

	out, err := doc.Html()
	if err != nil {
		return nil, false, fmt.Errorf("failed to re-render unhidden document: %v", err)
	}
	return []byte(out), true, nil
}
