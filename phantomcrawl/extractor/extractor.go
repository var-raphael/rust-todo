package extractor

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/var-raphael/phantomcrawl/output"
)

func Extract(rawURL string, html string, layer string) *output.ScrapedData {
	data := &output.ScrapedData{
		URL:       rawURL,
		LayerUsed: layer,
		Metadata:  map[string]string{},
		CrawledAt: time.Now(),
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		// If HTML can't be parsed it might be raw JSON from layer 2
		data.Content = html
		return data
	}

	base, err := url.Parse(rawURL)
	if err != nil {
		base = nil
	}

	// Title
	data.Title = strings.TrimSpace(doc.Find("title").First().Text())

	// Main text content
	data.Content = extractText(doc)

	// Links
	data.Links = extractLinks(doc, base)

	// Images
	data.Images = extractImages(doc, base)

	// Videos
	data.Videos = extractVideos(doc, base)

	// Documents
	data.Documents = extractDocuments(doc, base)

	// Emails
	data.Emails = extractEmails(doc)

	// Phone numbers
	data.Phones = extractPhones(doc)

	// Metadata
	data.Metadata = extractMetadata(doc)

	return data
}

func extractText(doc *goquery.Document) string {
	// Remove script and style tags
	doc.Find("script, style, noscript").Remove()

	var parts []string
	doc.Find("p, h1, h2, h3, h4, h5, h6, li, td, th, blockquote, pre").Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if len(text) > 10 {
			parts = append(parts, text)
		}
	})

	return strings.Join(parts, "\n")
}

func resolveURL(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
		return ""
	}
	if base == nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(ref).String()
}

func extractLinks(doc *goquery.Document, base *url.URL) []string {
	var links []string
	seen := map[string]bool{}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		resolved := resolveURL(base, href)
		if resolved == "" {
			return
		}
		// Skip mailto and tel — handled separately
		if strings.HasPrefix(resolved, "mailto:") || strings.HasPrefix(resolved, "tel:") {
			return
		}
		if !seen[resolved] {
			seen[resolved] = true
			links = append(links, resolved)
		}
	})

	return links
}

func extractImages(doc *goquery.Document, base *url.URL) []string {
	var images []string
	seen := map[string]bool{}

	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		if strings.HasPrefix(strings.TrimSpace(src), "data:") {
			return
		}
		resolved := resolveURL(base, src)
		if resolved == "" {
			return
		}
		if !seen[resolved] {
			seen[resolved] = true
			images = append(images, resolved)
		}
	})

	return images
}

func extractVideos(doc *goquery.Document, base *url.URL) []string {
	var videos []string
	seen := map[string]bool{}

	doc.Find("video source, iframe[src]").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("src")
		resolved := resolveURL(base, src)
		if resolved == "" {
			return
		}
		if !seen[resolved] {
			seen[resolved] = true
			videos = append(videos, resolved)
		}
	})

	return videos
}

func extractDocuments(doc *goquery.Document, base *url.URL) []string {
	var documents []string
	seen := map[string]bool{}

	extensions := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".csv"}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		resolved := resolveURL(base, href)
		if resolved == "" {
			return
		}
		for _, ext := range extensions {
			if strings.HasSuffix(strings.ToLower(resolved), ext) {
				if !seen[resolved] {
					seen[resolved] = true
					documents = append(documents, resolved)
				}
			}
		}
	})

	return documents
}

func extractEmails(doc *goquery.Document) []string {
	var emails []string
	seen := map[string]bool{}

	doc.Find("a[href^='mailto:']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		email := strings.TrimPrefix(href, "mailto:")
		email = strings.TrimSpace(email)
		if email != "" && !seen[email] {
			seen[email] = true
			emails = append(emails, email)
		}
	})

	return emails
}

func extractPhones(doc *goquery.Document) []string {
	var phones []string
	seen := map[string]bool{}

	doc.Find("a[href^='tel:']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		phone := strings.TrimPrefix(href, "tel:")
		phone = strings.TrimSpace(phone)
		if phone != "" && !seen[phone] {
			seen[phone] = true
			phones = append(phones, phone)
		}
	})

	return phones
}

func extractMetadata(doc *goquery.Document) map[string]string {
	meta := map[string]string{}

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		property, _ := s.Attr("property")
		content, _ := s.Attr("content")

		if name != "" && content != "" {
			meta[name] = content
		}
		if property != "" && content != "" {
			meta[property] = content
		}
	})

	return meta
}
