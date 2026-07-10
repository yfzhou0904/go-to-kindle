package repositories

import (
	"os"
	"path/filepath"
	"text/template"

	readability "github.com/go-shiori/go-readability"
)

type FileRepository interface {
	SaveArticle(article *readability.Article, path string) error
}

type localFileRepository struct{}

func NewLocalFileRepository() FileRepository {
	return &localFileRepository{}
}

type htmlData struct {
	Title   string
	Content string
	Author  string
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
       </style>
</head>
<body>
        {{.Content}}
</body>
</html>
`

func (r *localFileRepository) SaveArticle(article *readability.Article, path string) error {
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
		Title:   article.Title,
		Author:  article.Byline,
		Content: article.Content,
	}
	return t.Execute(file, data)
}
