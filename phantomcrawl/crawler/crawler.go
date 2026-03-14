package crawler

import (
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/var-raphael/phantomcrawl/config"
	"github.com/var-raphael/phantomcrawl/storage"
)

type Crawler struct {
	cfg       *config.Config
	db        *storage.DB
	browser   BrowserClient
	results   chan FetchResult
	semaphore chan struct{}
}

func New(cfg *config.Config, db *storage.DB, browser BrowserClient) *Crawler {
	return &Crawler{
		cfg:       cfg,
		db:        db,
		browser:   browser,
		semaphore: make(chan struct{}, 2),
	}
}

func (c *Crawler) CrawlAll(url string) <-chan FetchResult {
	c.results = make(chan FetchResult, 1000)

	var wg sync.WaitGroup
	wg.Add(1)
	go c.crawlWithDepth(url, "", 0, &wg)

	go func() {
		wg.Wait()
		close(c.results)
	}()

	return c.results
}

func (c *Crawler) crawlWithDepth(url string, parentURL string, currentDepth int, wg *sync.WaitGroup) {
	defer wg.Done()

	url = normalizeURL(url)

	c.semaphore <- struct{}{}
	defer func() { <-c.semaphore }()

	if currentDepth > c.cfg.Depth {
		return
	}

	if c.db.IsCrawled(url) {
		return
	}

	// Layer 1
	result := WithRetry(c.cfg.Retry, func() FetchResult {
		return Layer1Fetch(url, c.cfg.AntiBot.RotateUserAgents, c.cfg.AntiBot.Proxy)
	})

	if result.Error == nil {
		c.db.MarkCrawled(url, result.Layer)
		c.results <- result
		if currentDepth < c.cfg.Depth {
			c.followLinks(result.HTML, url, currentDepth, wg)
		}
		return
	}

	// Layer 2
	if result.HTML != "" {
		result2 := Layer2Fetch(url, result.HTML, c.cfg.AntiBot.RotateUserAgents)
		if result2.Error == nil {
			c.db.MarkCrawled(url, result2.Layer)
			c.results <- result2
			if currentDepth < c.cfg.Depth {
				c.followLinks(result2.HTML, url, currentDepth, wg)
			}
			return
		}
	}

	// Layer 2.5
	result25 := Layer25Fetch(url, c.browser)
	if result25.Error == nil {
		c.db.MarkCrawled(url, result25.Layer)
		c.results <- result25
		return
	}

	// Layer 3
	result3 := Layer3Fetch(url, c.browser)
	if result3.Error == nil {
		c.db.MarkCrawled(url, result3.Layer)
		c.results <- result3
		if currentDepth < c.cfg.Depth {
			c.followLinks(result3.HTML, url, currentDepth, wg)
		}
		return
	}

	// All layers failed
	c.db.MarkFailed(url, result3.Error.Error(), "layer3")
	fmt.Printf("  ✗ all layers failed: %s\n", url)
}

// followLinks extracts links from HTML and spawns crawl goroutines,
// respecting depth_limit (0 = unlimited)
func (c *Crawler) followLinks(html string, baseURL string, currentDepth int, wg *sync.WaitGroup) {
	links := extractLinks(html, baseURL)

	// Filter out non-crawlable assets
	skipExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".zip", ".tar", ".gz", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".mp4", ".mp3", ".csv"}
	var crawlable []string
	for _, link := range links {
		skip := false
		lower := strings.ToLower(link)
		for _, ext := range skipExts {
			if strings.HasSuffix(lower, ext) || strings.Contains(lower, ext+"?") {
				skip = true
				break
			}
		}
		if !skip {
			crawlable = append(crawlable, link)
		}
	}
	links = crawlable

	// Filter by domain
	if c.cfg.StayOnDomain {
		var filtered []string
		for _, link := range links {
			if sameDomain(link, baseURL) {
				filtered = append(filtered, link)
			}
		}
		links = filtered
	}

	// Apply per-parent depth limit
	if c.cfg.DepthLimit > 0 && len(links) > c.cfg.DepthLimit {
		links = links[:c.cfg.DepthLimit]
	}

	for _, link := range links {
		wg.Add(1)
		go c.crawlWithDepth(link, baseURL, currentDepth+1, wg)
	}
}

func extractLinks(html string, baseURL string) []string {
	var links []string
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return links
	}

	normalizedBase := normalizeURL(baseURL)
	seen := map[string]bool{}
	seen[normalizedBase] = true // skip links that resolve back to the parent
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		href = strings.TrimSpace(href)
		if href == "" || href == "#" || strings.HasPrefix(href, "javascript:") {
			return
		}
		resolved := resolveURL(baseURL, href)
		if resolved == "" {
			return
		}
		normalized := normalizeURL(resolved)
		if normalized != "" && !seen[normalized] {
			seen[normalized] = true
			links = append(links, normalized)
		}
	})

	return links
}

func normalizeURL(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		return u
	}
	// Strip fragment (#section) — same page, different scroll position
	parsed.Fragment = ""
	// Strip trailing slash unless it's just the root path
	if len(parsed.Path) > 1 {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}
	return parsed.String()
}

func sameDomain(url1, url2 string) bool {
	return getBaseURL(url1) == getBaseURL(url2)
}
