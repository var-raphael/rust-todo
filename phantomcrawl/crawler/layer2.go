package crawler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/var-raphael/phantomcrawl/antibot"
)

var apiPatterns = []*regexp.Regexp{
	regexp.MustCompile(`fetch\(['"]([^'"]+)['"]\)`),
	regexp.MustCompile(`axios\.(?:get|post|put)\(['"]([^'"]+)['"]\)`),
	regexp.MustCompile(`\.open\(['"](?:GET|POST)['"],\s*['"]([^'"]+)['"]\)`),
	regexp.MustCompile(`['"]( /api/[^'"]+)['"']`),
	regexp.MustCompile(`['"](/v1/[^'"]+)['"']`),
	regexp.MustCompile(`['"](/v2/[^'"]+)['"']`),
	regexp.MustCompile(`['"](/graphql[^'"]*?)['"']`),
	regexp.MustCompile(`['"]([^'"]+\.json)['"']`),
}

var dataPatterns = []*regexp.Regexp{
	regexp.MustCompile(`window\.__INITIAL_STATE__\s*=\s*({.+?});`),
	regexp.MustCompile(`window\.__NEXT_DATA__\s*=\s*({.+?});`),
	regexp.MustCompile(`window\.__NUXT__\s*=\s*({.+?});`),
	regexp.MustCompile(`window\.DATA\s*=\s*({.+?});`),
}

func Layer2Fetch(url string, html string, rotate bool) FetchResult {
	// Step 1 - check for embedded data dumps
	for _, pattern := range dataPatterns {
		match := pattern.FindString(html)
		if match != "" {
			start := strings.Index(match, "{")
			if start != -1 {
				jsonStr := match[start:]
				if isValidJSON(jsonStr) {
					return FetchResult{URL: url, HTML: jsonStr, Layer: "layer2"}
				}
			}
		}
	}

	// Step 2 - check for JSON-LD structured data
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		var jsonLD string
		doc.Find(`script[type="application/ld+json"]`).Each(func(i int, s *goquery.Selection) {
			if jsonLD == "" {
				jsonLD = strings.TrimSpace(s.Text())
			}
		})
		if jsonLD != "" && isValidJSON(jsonLD) {
			return FetchResult{URL: url, HTML: jsonLD, Layer: "layer2"}
		}
	}

	// Step 3 - scan for API endpoints
	endpoints := extractEndpoints(html, url)

	// Step 4 - check WordPress REST API
	if strings.Contains(url, "wp-") || doc.Find(`meta[name="generator"]`).Length() > 0 {
		wpAPI := getBaseURL(url) + "/wp-json/wp/v2/posts"
		endpoints = append([]string{wpAPI}, endpoints...)
	}

	// Step 5 - probe each endpoint
	for _, endpoint := range endpoints {
		result := probeEndpoint(endpoint, rotate)
		if result != "" {
			return FetchResult{URL: url, HTML: result, Layer: "layer2"}
		}
	}

	return FetchResult{URL: url, Error: fmt.Errorf("no API found"), Layer: "layer2"}
}

func extractEndpoints(html string, baseURL string) []string {
	var endpoints []string
	seen := map[string]bool{}

	for _, pattern := range apiPatterns {
		matches := pattern.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				endpoint := resolveURL(baseURL, match[len(match)-1])
				if endpoint != "" && !seen[endpoint] {
					seen[endpoint] = true
					endpoints = append(endpoints, endpoint)
				}
			}
		}
	}

	return endpoints
}

func probeEndpoint(url string, rotate bool) string {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ""
	}

	ua := antibot.RandomUserAgent()
	antibot.HumanHeaders(req, ua)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	if isValidJSON(string(body)) && len(body) > 100 {
		return string(body)
	}

	return ""
}

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}
