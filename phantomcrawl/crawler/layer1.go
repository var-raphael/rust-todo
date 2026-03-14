package crawler

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/var-raphael/phantomcrawl/antibot"
	"github.com/var-raphael/phantomcrawl/config"
)

type FetchResult struct {
	URL   string
	HTML  string
	Layer string
	Error error
}

func Layer1Fetch(targetURL string, rotate bool, proxyCfg config.ProxyConfig) FetchResult {
	var client *http.Client

	if proxyCfg.Enabled && len(proxyCfg.URLs) > 0 {
		rotator := antibot.NewProxyRotator(proxyCfg.URLs, proxyCfg.KeyRotation)
		proxy := rotator.Next()
		client = antibot.NewTLSClientWithProxy(proxy)
	} else {
		client = antibot.NewTLSClient()
	}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return FetchResult{URL: targetURL, Error: fmt.Errorf("request creation failed: %w", err), Layer: "layer1"}
	}

	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36"
	if rotate {
		ua = antibot.RandomUserAgent()
	}
	antibot.HumanHeaders(req, ua)

	resp, err := client.Do(req)
	if err != nil {
		return FetchResult{URL: targetURL, Error: fmt.Errorf("request failed: %w", err), Layer: "layer1"}
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return FetchResult{URL: targetURL, Error: fmt.Errorf("rate limited (429)"), Layer: "layer1"}
	}

	if resp.StatusCode >= 400 {
		return FetchResult{URL: targetURL, Error: fmt.Errorf("HTTP %d", resp.StatusCode), Layer: "layer1"}
	}

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return FetchResult{URL: targetURL, Error: fmt.Errorf("gzip error: %w", err), Layer: "layer1"}
		}
		defer gz.Close()
		reader = gz
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return FetchResult{URL: targetURL, Error: fmt.Errorf("read failed: %w", err), Layer: "layer1"}
	}

	html := string(body)

	if !isMeaningful(html) {
		return FetchResult{URL: targetURL, HTML: html, Error: fmt.Errorf("content not meaningful"), Layer: "layer1"}
	}

	return FetchResult{URL: targetURL, HTML: html, Layer: "layer1"}
}

func isMeaningful(html string) bool {
	if len(html) < 5 {
		return false
	}

	trimmed := strings.TrimSpace(html)

	// Accept plain text responses (not HTML) — e.g. IP addresses, JSON, plain API responses
	if !strings.Contains(trimmed, "<") {
		return len(trimmed) > 0
	}

	// For HTML responses, require minimum size
	if len(html) < 200 {
		return false
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return false
	}

	body := doc.Find("body")
	bodyText := strings.TrimSpace(body.Text())
	if len(bodyText) < 50 {
		// Also accept short responses with pre/code (JSON, API responses)
		preText := strings.TrimSpace(doc.Find("pre, code").Text())
		if len(preText) < 10 {
			return false
		}
	}

	return true
}
