package repositories

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	readability "github.com/go-shiori/go-readability"
)

type FileRepository interface {
	SaveArticle(article *readability.Article, path string) error
	SaveArticleWithDates(article *readability.Article, path string, sentTime time.Time) error
}

type localFileRepository struct{}

func NewLocalFileRepository() FileRepository {
	return &localFileRepository{}
}

type htmlData struct {
	Title       string
	Content     string
	Author      string
	DateContext string
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
        <meta charset="utf-8">
        <title>{{.Title}}</title>
        <meta name="author" content="{{.Author}}">
       <style>
               body { line-height: 1.5; }
               img { display: block; max-width: 100%; height: auto; margin-left: auto; margin-right: auto; }
               pre { white-space: pre-wrap; overflow-wrap: anywhere; padding: 0.75em; background: #f3f3f3; }
               code { font-family: monospace; }
               table { width: 100%; border-collapse: collapse; }
               th, td { border: 1px solid #999; padding: 0.35em; text-align: left; }
               blockquote { margin-left: 0; padding-left: 1em; border-left: 3px solid #999; }
               .article-dates { color: #666; font-size: 0.9em; margin: 0 0 0.75em; }
       </style>
</head>
<body>
        {{if .DateContext}}<p class="article-dates">{{.DateContext}}</p>{{end}}
        {{.Content}}
</body>
</html>
`

func (r *localFileRepository) SaveArticle(article *readability.Article, path string) error {
	return r.saveArticle(article, path, "")
}

func (r *localFileRepository) SaveArticleWithDates(article *readability.Article, path string, sentTime time.Time) error {
	return r.saveArticle(article, path, FormatDateContext(article, sentTime))
}

func (r *localFileRepository) saveArticle(article *readability.Article, path string, dateContext string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.New("html").Parse(htmlTemplate))
	data := htmlData{
		Title:       article.Title,
		Author:      article.Byline,
		Content:     defuseCodeTokens(article.Content),
		DateContext: dateContext,
	}
	return t.Execute(file, data)
}

// codeBlockRE matches the inner text of each <code> element (which is where all
// fenced, indented, and inline code lands after rendering).
var codeBlockRE = regexp.MustCompile(`(?s)(<code[^>]*>)(.*?)(</code>)`)

// wordOrEntityRE matches either an existing HTML entity (left untouched) or a
// word of source text (whose first letter we encode).
var wordOrEntityRE = regexp.MustCompile(`&#?[0-9A-Za-z]+;|[A-Za-z][A-Za-z0-9_]*`)

// defuseCodeTokens rewrites the first letter of every word inside <code> blocks
// as a numeric character reference (e.g. "from" -> "&#102;rom"). This renders
// byte-for-byte identically to the reader but ensures no source-language keyword
// (from/import/def/class/print/...) survives as literal ASCII in the file. That
// prevents content sniffers (libmagic, and Amazon's Send-to-Kindle converter)
// from misclassifying code-heavy documents as a script instead of HTML, which
// would otherwise be delivered as plain text and rendered with raw tags visible.
// The transform is content-agnostic: it defeats detection regardless of which
// keywords appear, without enumerating any language.
func defuseCodeTokens(content string) string {
	return codeBlockRE.ReplaceAllStringFunc(content, func(block string) string {
		m := codeBlockRE.FindStringSubmatch(block)
		open, inner, closeTag := m[1], m[2], m[3]
		encoded := wordOrEntityRE.ReplaceAllStringFunc(inner, func(tok string) string {
			if strings.HasPrefix(tok, "&") {
				return tok // existing entity such as &lt; or &#39; — leave intact
			}
			return "&#" + strconv.Itoa(int(tok[0])) + ";" + tok[1:]
		})
		return open + encoded + closeTag
	})
}

// FormatDateContext returns the date line shown in the review screen and final article.
func FormatDateContext(article *readability.Article, sentTime time.Time) string {
	const dateFormat = "2006/1/2"
	formatDate := func(date time.Time) string {
		return date.In(sentTime.Location()).Format(dateFormat)
	}
	parts := make([]string, 0, 3)
	if article.PublishedTime != nil {
		parts = append(parts, "Published "+formatDate(*article.PublishedTime))
	}
	if article.ModifiedTime != nil && (article.PublishedTime == nil || formatDate(*article.ModifiedTime) != formatDate(*article.PublishedTime)) {
		parts = append(parts, "Updated "+formatDate(*article.ModifiedTime))
	}
	parts = append(parts, "Sent "+formatDate(sentTime))
	return strings.Join(parts, " · ")
}
