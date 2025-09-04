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
               img { display: block; margin-left: auto; margin-right: auto; }
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
