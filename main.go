package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/yfzhou0904/go-to-kindle/mail"

	"github.com/PuerkitoBio/goquery"
	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"

	tea "github.com/charmbracelet/bubbletea"
)

var Conf Config = Config{
	Email: ConfigEmail{
		SMTPServer: "smtp.example.com",
		Port:       456,
		From:       "YOUR@EMAIL.com",
		Password:   "YOUR_EMAIL_PSWD",
		To:         "YOU@kindle.com",
	},
}

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}

// Extracted logic from original Send function
func fetchAndParse(input string) (*readability.Article, string, string, int, error) {
	link := input
	var resp *http.Response

	if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
		// web url
		validURL, err := url.Parse(link)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to parse URL: %v", err)
		}

		resp, err = getWebPage(validURL)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to get webpage: %v", err)
		}
		defer resp.Body.Close()
	} else {
		// local file
		absPath, err := filepath.Abs(link)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to resolve local file path: %v", err)
		}
		file, err := os.Open(absPath)
		if err != nil {
			return nil, "", "", 0, fmt.Errorf("failed to open local file: %v", err)
		}
		defer file.Close()
		resp = &http.Response{
			Body: file,
			Request: &http.Request{
				URL: &url.URL{
					Path: link,
				},
			},
		}
	}

	article, filename, err := parseWebPage(resp, resp.Request.URL)
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to parse webpage: %v", err)
	}

	// Check if article contains any blocked key elements indicating parsing failure
	for _, blockedElem := range blockedKeyElems {
		if strings.Contains(article.Content, blockedElem) {
			return nil, "", "", 0, fmt.Errorf("failed to parse webpage: we have probably been blocked, pattern: '%s'", blockedElem)
		}
	}

	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to parse content: %v", err)
	}
	contentDoc.Find("img,source,figure,svg").Remove()
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		var buf strings.Builder
		s.Contents().Each(func(j int, c *goquery.Selection) {
			buf.WriteString(c.Text())
		})
		s.ReplaceWithHtml(buf.String())
	})
	article.Content, err = contentDoc.Find("body").Html()
	if err != nil {
		return nil, "", "", 0, fmt.Errorf("failed to extract content: %v", err)
	}

	// language detection for better word counting
	lang := whatlanggo.DetectLangWithOptions(article.TextContent, whatlanggo.Options{
		Whitelist: map[whatlanggo.Lang]bool{
			whatlanggo.Cmn: true,
			whatlanggo.Eng: true,
		},
	})
	wordCount := 0
	if lang == whatlanggo.Cmn {
		wordCount = utf8.RuneCountInString(article.Content)
	} else {
		wordCount = len(strings.Fields(article.Content))
	}
	if wordCount < 100 {
		return nil, "", "", 0, fmt.Errorf("article is too short (%d words)", wordCount)
	}

	return article, filename, lang.String(), wordCount, nil
}

// Process and send article
func processAndSend(article *readability.Article, filename string) error {
	createFile(filepath.Join(baseDir(), "archive", filename))
	err := writeToFile(article, filepath.Join(baseDir(), "archive", filename))
	if err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	err = mail.SendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), filepath.Join(baseDir(), "archive", filename), Conf.Email.Port)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func getWebPage(url *url.URL) (*http.Response, error) {
	// Create a new request using http
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set the User-Agent header to mimic a normal browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")

	// Create a new http client
	client := http.Client{
		Transport: http.DefaultTransport.(*http.Transport).Clone(),
	}

	// Send the request using the client
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func parseWebPage(resp *http.Response, url *url.URL) (*readability.Article, string, error) {
	article, err := readability.FromReader(resp.Body, url)
	if err != nil {
		return nil, "", err
	}
	var title string
	if strings.HasPrefix(url.String(), "http") {
		title = article.Title
	} else {
		title = filepath.Base(url.Path)
		title = strings.TrimSuffix(title, filepath.Ext(title))
	}
	return &article, titleToFilename(title), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
	<title>{{.Title}}</title>
	<meta name="author" content="{{.Author}}">
</head>
<body>
	{{.Content}}
</body>
</html>
`

type HtmlData struct {
	Title   string
	Content string
	Author  string
}

func writeToFile(article *readability.Article, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.New("html").Parse(htmlTemplate))
	err = t.Execute(file, HtmlData{
		Title:   article.Title,
		Author:  article.Byline,
		Content: article.Content,
	})
	if err != nil {
		return err
	}

	return nil
}

// replace problematic characters in page title to give a generally valid filename
func titleToFilename(title string) string {
	filename := strings.ReplaceAll(title, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")
	return filename + ".html"
}

// user config and article data are stored in ~/.go-to-kindle
func baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(home, ".go-to-kindle")
}

func createFile(p string) (*os.File, error) {
	// Create directories if they do not exist
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}

	// Create the file
	return os.Create(p)
}

var (
	// seeing these elements means parsing failed
	blockedKeyElems = []string{
		`<div id="cf-error-details">`,
		`<title>Attention Required! | Cloudflare</title>`,
	}
)
