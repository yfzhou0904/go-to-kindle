package webarchive

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"
	"howett.net/plist"
)

type archive struct {
	MainResource        Resource   `plist:"MainResource"`
	Subresources        []Resource `plist:"Subresources"`
	WebMainResource     Resource   `plist:"WebMainResource"`
	WebSubresources     []Resource `plist:"WebSubresources"`
	WebSubframeArchives []archive  `plist:"WebSubframeArchives"`
}

type Resource struct {
	Data             []byte `plist:"WebResourceData"`
	URL              string `plist:"WebResourceURL"`
	MIMEType         string `plist:"WebResourceMIMEType"`
	TextEncodingName string `plist:"WebResourceTextEncodingName"`
}

// DecodeFile parses a .webarchive plist and returns the main HTML, base URL, and resources.
func DecodeFile(contents []byte) ([]byte, *url.URL, map[string]Resource, error) {
	var ar archive
	if _, err := plist.Unmarshal(contents, &ar); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse webarchive plist: %w", err)
	}

	main := ar.MainResource
	if ar.WebMainResource.URL != "" || len(ar.WebMainResource.Data) > 0 {
		main = ar.WebMainResource
	}

	baseURL, err := url.Parse(main.URL)
	if err != nil {
		baseURL = nil
	}

	resources := make(map[string]Resource)
	appendResources(resources, ar.Subresources)
	appendResources(resources, ar.WebSubresources)
	for _, sub := range ar.WebSubframeArchives {
		appendResources(resources, sub.Subresources)
		appendResources(resources, sub.WebSubresources)
	}

	data := main.Data
	if decoded, ok := decodeTextResource(data, main.TextEncodingName); ok {
		data = decoded
	}

	return data, baseURL, resources, nil
}

// InlineImages replaces image URLs in HTML with data URLs using webarchive resources.
func InlineImages(html []byte, baseURL *url.URL, resources map[string]Resource) ([]byte, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse webarchive HTML: %w", err)
	}

	inlineURL := func(raw string) (string, bool) {
		resolved, ok := resolveResource(raw, baseURL, resources)
		if !ok {
			return "", false
		}
		dataURL, ok := resourceToDataURL(resolved)
		return dataURL, ok
	}

	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		inlinedSrc := ""
		updatedSrcset := ""

		if dataAttrs, ok := s.Attr("data-attrs"); ok {
			if raw := extractURLFromDataAttrs(dataAttrs); raw != "" {
				if dataURL, ok := inlineURL(raw); ok {
					s.SetAttr("src", dataURL)
					inlinedSrc = dataURL
				}
				s.RemoveAttr("data-attrs")
			}
		}

		if src, ok := s.Attr("src"); ok {
			if dataURL, ok := inlineURL(src); ok {
				s.SetAttr("src", dataURL)
				inlinedSrc = dataURL
			} else if isExternalURL(src) {
				s.RemoveAttr("src")
			}
		}
		if src, ok := s.Attr("data-src"); ok {
			if dataURL, ok := inlineURL(src); ok {
				s.SetAttr("data-src", dataURL)
				if inlinedSrc == "" {
					s.SetAttr("src", dataURL)
					inlinedSrc = dataURL
				}
			} else if isExternalURL(src) {
				s.RemoveAttr("data-src")
			}
		}
		if srcset, ok := s.Attr("srcset"); ok {
			if updated, ok := inlineSrcset(srcset, baseURL, resources); ok {
				s.SetAttr("srcset", updated)
				updatedSrcset = updated
			} else {
				s.RemoveAttr("srcset")
			}
		}
		if srcset, ok := s.Attr("data-srcset"); ok {
			if updated, ok := inlineSrcset(srcset, baseURL, resources); ok {
				s.SetAttr("data-srcset", updated)
				if updatedSrcset == "" {
					updatedSrcset = updated
				}
			} else {
				s.RemoveAttr("data-srcset")
			}
		}

		if inlinedSrc == "" && updatedSrcset != "" {
			first := strings.TrimSpace(strings.Split(updatedSrcset, ",")[0])
			fields := strings.Fields(first)
			if len(fields) > 0 {
				s.SetAttr("src", fields[0])
			}
		}
	})

	doc.Find("source").Each(func(_ int, s *goquery.Selection) {
		if src, ok := s.Attr("src"); ok {
			if dataURL, ok := inlineURL(src); ok {
				s.SetAttr("src", dataURL)
			} else if isExternalURL(src) {
				s.RemoveAttr("src")
			}
		}
		if srcset, ok := s.Attr("srcset"); ok {
			if updated, ok := inlineSrcset(srcset, baseURL, resources); ok {
				s.SetAttr("srcset", updated)
			} else {
				s.RemoveAttr("srcset")
			}
		}
	})

	rendered, err := doc.Html()
	if err != nil {
		return nil, fmt.Errorf("failed to render webarchive HTML: %w", err)
	}
	return []byte(rendered), nil
}

// ResolveImageDataURL resolves a source URL to a data URL from resources.
func ResolveImageDataURL(raw string, baseURL *url.URL, resources map[string]Resource) (string, bool) {
	resolved, ok := resolveResource(raw, baseURL, resources)
	if !ok {
		return "", false
	}
	dataURL, ok := resourceToDataURL(resolved)
	return dataURL, ok
}

func inlineSrcset(srcset string, baseURL *url.URL, resources map[string]Resource) (string, bool) {
	parts := strings.Split(srcset, ",")
	kept := make([]string, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		rawURL := fields[0]
		if res, ok := resolveResource(rawURL, baseURL, resources); ok {
			if dataURL, ok := resourceToDataURL(res); ok {
				fields[0] = dataURL
				kept = append(kept, strings.Join(fields, " "))
				continue
			}
		}
		_ = i
	}
	if len(kept) == 0 {
		return "", false
	}
	return strings.Join(kept, ", "), true
}

func resolveResource(raw string, baseURL *url.URL, resources map[string]Resource) (Resource, bool) {
	parsed, err := url.Parse(raw)
	if err == nil {
		if !parsed.IsAbs() && baseURL != nil {
			parsed = baseURL.ResolveReference(parsed)
		}
		if res, ok := resources[parsed.String()]; ok {
			return res, true
		}
		if parsed.Path != "" {
			if stripped := stripSizeSuffix(parsed.Path); stripped != parsed.Path {
				alt := *parsed
				alt.Path = stripped
				if res, ok := resources[alt.String()]; ok {
					return res, true
				}
			}
		}
	}

	if res, ok := resources[raw]; ok {
		return res, true
	}

	return Resource{}, false
}

func resourceToDataURL(res Resource) (string, bool) {
	mimeType := res.MIMEType
	if mimeType == "" {
		mimeType = http.DetectContentType(res.Data)
	}
	if !strings.HasPrefix(mimeType, "image/") {
		return "", false
	}
	encoded := base64.StdEncoding.EncodeToString(res.Data)
	return "data:" + mimeType + ";base64," + encoded, true
}

func extractURLFromDataAttrs(dataAttr string) string {
	var attrs struct {
		Src            string `json:"src"`
		SrcNoWatermark string `json:"srcNoWatermark"`
	}
	if err := json.Unmarshal([]byte(dataAttr), &attrs); err != nil {
		return ""
	}
	if attrs.SrcNoWatermark != "" {
		return attrs.SrcNoWatermark
	}
	return attrs.Src
}

func appendResources(dest map[string]Resource, items []Resource) {
	for _, res := range items {
		if res.URL == "" {
			continue
		}
		if _, exists := dest[res.URL]; !exists {
			dest[res.URL] = res
		}
	}
}

func decodeTextResource(data []byte, encodingName string) ([]byte, bool) {
	if encodingName == "" {
		return nil, false
	}
	reader, err := charset.NewReaderLabel(encodingName, bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return nil, false
	}
	if len(decoded) == 0 || !utf8.Valid(decoded) {
		return nil, false
	}
	return decoded, true
}

func isExternalURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func stripSizeSuffix(path string) string {
	dot := strings.LastIndex(path, ".")
	if dot == -1 {
		return path
	}
	base := path[:dot]
	ext := path[dot:]
	dash := strings.LastIndex(base, "-")
	if dash == -1 {
		return path
	}
	size := base[dash+1:]
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return path
	}
	if !isDigits(parts[0]) || !isDigits(parts[1]) {
		return path
	}
	return base[:dash] + ext
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
