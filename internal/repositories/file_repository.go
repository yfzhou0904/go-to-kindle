package repositories

import (
	"os"
	"path/filepath"
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
		Content:     article.Content,
		DateContext: dateContext,
	}
	return t.Execute(file, data)
}

// FormatDateContext returns the date line shown in the review screen and final article.
func FormatDateContext(article *readability.Article, sentTime time.Time) string {
	const dateFormat = "2006/1/2"
	parts := make([]string, 0, 3)
	if article.PublishedTime != nil {
		parts = append(parts, "Published "+article.PublishedTime.Format(dateFormat))
	}
	if article.ModifiedTime != nil && (article.PublishedTime == nil || article.ModifiedTime.Format(dateFormat) != article.PublishedTime.Format(dateFormat)) {
		parts = append(parts, "Updated "+article.ModifiedTime.Format(dateFormat))
	}
	parts = append(parts, "Sent "+sentTime.Format(dateFormat))
	return strings.Join(parts, " · ")
}
