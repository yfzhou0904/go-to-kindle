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

	"github.com/BurntSushi/toml"
	"github.com/PuerkitoBio/goquery"
	"github.com/abadojack/whatlanggo"
	readability "github.com/go-shiori/go-readability"
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
	var err error

	if err = loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide a URL as a command line argument.")
	}

	link := os.Args[1]
	if !strings.HasPrefix(link, "http") { // when user omit http prefix, add it back
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

	fmt.Println("Filename:", filename)

	contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		panic(err)
	}
	contentDoc.Find("img,source,figure").Remove()
	contentDoc.Find("a").Each(func(i int, s *goquery.Selection) {
		var buf strings.Builder
		s.Contents().Each(func(j int, c *goquery.Selection) {
			buf.WriteString(c.Text())
		})
		s.ReplaceWithHtml(buf.String())
	})
	article.Content, err = contentDoc.Html()
	if err != nil {
		panic(err)
	}
	fmt.Println("Removed media.")

	// language detection for better word counting
	langInfo := whatlanggo.Detect(article.TextContent)
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

	err = mail.SendEmailWithAttachment(Conf.Email.SMTPServer, Conf.Email.From, Conf.Email.Password, Conf.Email.To, strings.TrimSuffix(filename, ".html"), filepath.Join(baseDir(), "archive", filename), Conf.Email.Port)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	fmt.Println("Email sent.")
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
	client := &http.Client{}

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
	title := article.Title
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

type Config struct {
	Email ConfigEmail
}
type ConfigEmail struct {
	SMTPServer string `toml:"smtp_server"`
	Port       int
	From       string
	Password   string
	To         string
}

func loadConfig() error {
	filepath := filepath.Join(baseDir(), "config.toml")

	// init example config file if does not exist
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		if err = initConfig(); err != nil {
			return err
		}
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, &Conf)
}

func initConfig() error {
	path := filepath.Join(baseDir(), "config.toml")
	fmt.Println("Initializing config file at", path)

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(&Conf); err != nil {
		return err
	}

	return nil
}
