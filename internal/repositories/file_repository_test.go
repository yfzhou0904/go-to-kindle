package repositories

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	readability "github.com/go-shiori/go-readability"
)

func TestSaveArticleWithDates(t *testing.T) {
	published := time.Date(2020, time.January, 2, 12, 0, 0, 0, time.UTC)
	modified := time.Date(2021, time.November, 9, 12, 0, 0, 0, time.UTC)
	article := &readability.Article{
		Title:         "Dated article",
		Content:       "<p>Article body</p>",
		PublishedTime: &published,
		ModifiedTime:  &modified,
	}
	path := filepath.Join(t.TempDir(), "article.html")

	err := NewLocalFileRepository().SaveArticleWithDates(article, path, time.Date(2026, time.July, 14, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("SaveArticleWithDates error: %v", err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read article: %v", err)
	}
	html := string(contents)
	want := "Published 2020/1/2 · Updated 2021/11/9 · Sent 2026/7/14"
	if !strings.Contains(html, `<p class="article-dates">`+want+`</p>`) {
		t.Fatalf("expected date context %q in HTML:\n%s", want, html)
	}
	if strings.Index(html, want) > strings.Index(html, "Article body") {
		t.Fatal("date context should precede the article content")
	}
}

func TestFormatDateContextOmitsSameDayUpdate(t *testing.T) {
	published := time.Date(2026, time.July, 14, 1, 0, 0, 0, time.UTC)
	modified := time.Date(2026, time.July, 14, 22, 0, 0, 0, time.UTC)
	article := &readability.Article{PublishedTime: &published, ModifiedTime: &modified}

	got := FormatDateContext(article, time.Date(2026, time.July, 15, 0, 0, 0, 0, time.UTC))
	if got != "Published 2026/7/14 · Sent 2026/7/15" {
		t.Fatalf("unexpected date context: %q", got)
	}
}

func TestFormatDateContextNormalizesSourceDatesToSentTimezone(t *testing.T) {
	singapore := time.FixedZone("Singapore", 8*60*60)
	published := time.Date(2026, time.July, 14, 2, 47, 6, 0, singapore)
	modified := time.Date(2026, time.July, 13, 18, 52, 33, 0, time.UTC)
	article := &readability.Article{PublishedTime: &published, ModifiedTime: &modified}
	sent := time.Date(2026, time.July, 14, 18, 0, 0, 0, singapore)

	got := FormatDateContext(article, sent)
	if got != "Published 2026/7/14 · Sent 2026/7/14" {
		t.Fatalf("unexpected mixed-timezone date context: %q", got)
	}
}

func TestSaveArticleWithoutDates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "article.html")
	article := &readability.Article{Title: "Plain article", Content: "<p>Body</p>"}
	if err := NewLocalFileRepository().SaveArticle(article, path); err != nil {
		t.Fatalf("SaveArticle error: %v", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read article: %v", err)
	}
	if strings.Contains(string(contents), "article-dates") && strings.Contains(string(contents), "Sent ") {
		t.Fatal("plain article should not contain a date line")
	}
}
