package readability

import (
	_ "embed"
	"fmt"
	"net/url"

	"github.com/grafana/sobek"
)

//go:embed readability.js
var readabilityJS string

//go:embed linkedom.js
var linkedomJS string

type Article struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	TextContent string `json:"textContent"`
	Length      int    `json:"length"`
	Excerpt     string `json:"excerpt"`
	Byline      string `json:"byline"`
	Dir         string `json:"dir"`
	SiteName    string `json:"siteName"`
	Lang        string `json:"lang"`
}

type Parser struct {
	runtime *sobek.Runtime
}

func NewParser() (*Parser, error) {
	runtime := sobek.New()
	
	// Load LinkedOM first to provide DOM implementation
	_, err := runtime.RunString(linkedomJS)
	if err != nil {
		return nil, fmt.Errorf("failed to load linkedom.js: %w", err)
	}
	
	// Load readability.js
	_, err = runtime.RunString(readabilityJS)
	if err != nil {
		return nil, fmt.Errorf("failed to load readability.js: %w", err)
	}
	
	return &Parser{runtime: runtime}, nil
}

func (p *Parser) Parse(htmlContent string, baseURL *url.URL) (*Article, error) {
	// Use LinkedOM to create a proper DOM document, following Mozilla's recommended approach
	script := fmt.Sprintf(`
		// Create DOM using LinkedOM (similar to JSDOM approach)
		const { DOMParser } = linkedom;
		const parser = new DOMParser();
		const document = parser.parseFromString(%q, 'text/html');
		
		// Set the document URI for relative URL resolution
		document.documentURI = %q;
		document.URL = %q;
		
		// Create Readability instance and parse
		const readability = new Readability(document, {});
		const article = readability.parse();
		
		JSON.stringify(article);
	`, htmlContent, baseURL.String(), baseURL.String())
	
	result, err := p.runtime.RunString(script)
	if err != nil {
		return nil, fmt.Errorf("failed to parse article: %w", err)
	}
	
	if result.String() == "null" {
		return nil, fmt.Errorf("readability failed to parse article")
	}
	
	var article Article
	exported := result.ToObject(p.runtime).Export()
	if exported == nil {
		return nil, fmt.Errorf("failed to export article")
	}
	
	// Convert the exported value to our Article struct
	if articleMap, ok := exported.(map[string]interface{}); ok {
		if title, ok := articleMap["title"].(string); ok {
			article.Title = title
		}
		if content, ok := articleMap["content"].(string); ok {
			article.Content = content
		}
		if textContent, ok := articleMap["textContent"].(string); ok {
			article.TextContent = textContent
		}
		if length, ok := articleMap["length"].(int64); ok {
			article.Length = int(length)
		}
		if excerpt, ok := articleMap["excerpt"].(string); ok {
			article.Excerpt = excerpt
		}
		if byline, ok := articleMap["byline"].(string); ok {
			article.Byline = byline
		}
		if dir, ok := articleMap["dir"].(string); ok {
			article.Dir = dir
		}
		if siteName, ok := articleMap["siteName"].(string); ok {
			article.SiteName = siteName
		}
		if lang, ok := articleMap["lang"].(string); ok {
			article.Lang = lang
		}
	} else {
		return nil, fmt.Errorf("failed to convert result to article map")
	}
	
	return &article, nil
}