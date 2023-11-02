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

	"github.com/BurntSushi/toml"
	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
)

func main() {
	conf, err := loadConfig(filepath.Join(baseDir(), "config.toml"))
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide a URL as a command line argument.")
	}

	link := os.Args[1]
	if !strings.HasPrefix(link, "http") {
		link = "https://" + link
	}
	validURL, err := url.Parse(link)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}

	resp, err := getWebPage(validURL)
	if err != nil {
		log.Fatalf("Failed to get webpage: %v", err)
	}
	defer resp.Body.Close()
	fmt.Println("Retrieved.")

	article, filename, err := parseWebPage(resp, validURL)
	if err != nil {
		log.Fatalf("Failed to parse webpage: %v", err)
	}

	langInfo := whatlanggo.Detect(article.Content)
	fmt.Printf("Detected language: %s.\n", langInfo.Lang.String())
	wordCount := 0
	if langInfo.IsReliable() && langInfo.Lang == whatlanggo.Cmn {
		wordCount = utf8.RuneCountInString(article.Content)
		fmt.Printf("Parsed, length = %d.\n", wordCount)
	} else {
		wordCount = len(strings.Fields(article.Content))
		fmt.Printf("Parsed, length = %d.\n", wordCount)
	}
	if wordCount < 100 {
		fmt.Println()
		fmt.Println(article.Content)
		fmt.Println()
		log.Fatalln("Article is too short, exiting.")
	}

	createFile(filepath.Join(baseDir(), "archive", filename))
	err = writeToFile(article, filepath.Join(baseDir(), "archive", filename))
	if err != nil {
		log.Fatalf("Failed to write to file: %v", err)
	}
	fmt.Println("Written.")

	err = sendEmailWithAttachment(conf.Email.SMTPServer, conf.Email.From, conf.Email.Password, conf.Email.To, strings.TrimSuffix(filename, ".html"), filepath.Join(baseDir(), "archive", filename), conf.Email.Port)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	fmt.Println("Email sent.")
}

func getWebPage(url *url.URL) (*http.Response, error) {
	resp, err := http.Get(url.String())
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
	title := article.Title
	return &article, titleToFilename(title), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
	<title>{{.Title}}</title>
</head>
<body>
	{{.Content}}
</body>
</html>
`

type HtmlData struct {
	Title   string
	Content string
}

func writeToFile(article *readability.Article, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.New("html").Parse(htmlTemplate))
	err = t.Execute(file, HtmlData{Title: article.Title, Content: article.Content})
	if err != nil {
		return err
	}

	return nil
}

func titleToFilename(title string) string {
	filename := strings.ReplaceAll(title, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ":", "-")
	filename = strings.ReplaceAll(filename, "*", "-")
	filename = strings.ReplaceAll(filename, "?", "-")
	filename = strings.ReplaceAll(filename, "\"", "-")
	filename = strings.ReplaceAll(filename, "<", "-")
	filename = strings.ReplaceAll(filename, ">", "-")
	filename = strings.ReplaceAll(filename, "|", "-")
	return filename + ".html"
}

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

type Config struct {
	Email struct {
		SMTPServer string `toml:"smtp_server"`
		Port       int
		From       string
		Password   string
		To         string
	}
}

func loadConfig(filename string) (Config, error) {
	var config Config
	data, err := os.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = toml.Unmarshal(data, &config)
	return config, err
}
