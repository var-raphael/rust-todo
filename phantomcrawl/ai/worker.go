package ai

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/var-raphael/phantomcrawl/storage"
)

const (
	// Minimum content length to bother AI cleaning.
	// Pages with less than this are likely empty, redirects, or near-duplicate language variants.
	minContentLength = 300
)

type Worker struct {
	cleaner   *Cleaner
	db        *storage.DB
	outputDir string
	semaphore chan struct{}
	Queue     chan string
	wg        sync.WaitGroup
}

func NewWorker(cleaner *Cleaner, db *storage.DB, outputDir string, maxConcurrent int) *Worker {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}
	return &Worker{
		cleaner:   cleaner,
		db:        db,
		outputDir: outputDir,
		semaphore: make(chan struct{}, maxConcurrent),
		Queue:     make(chan string, 1000),
	}
}

func (w *Worker) Start() {
	go func() {
		for url := range w.Queue {
			w.semaphore <- struct{}{} // block until a slot is free
			w.wg.Add(1)
			go func(u string) {
				defer w.wg.Done()
				defer func() { <-w.semaphore }()
				w.cleanURL(u)
			}(url)
		}
		// drain semaphore - wait for all in-flight workers to finish
		for i := 0; i < cap(w.semaphore); i++ {
			w.semaphore <- struct{}{}
		}
	}()
}

func (w *Worker) Enqueue(url string) {
	if !w.cleaner.cfg.Enabled {
		return
	}
	if !w.cleaner.cfg.SaveCleaned {
		return
	}
	if w.db.IsCleaned(url) {
		return
	}
	defer func() { recover() }() // prevent panic if queue closed
	w.Queue <- url
}

func (w *Worker) Wait() {
	w.wg.Wait()
}

func (w *Worker) cleanURL(targetURL string) {
	rawPath := w.findRawJSON(targetURL)
	if rawPath == "" {
		fmt.Printf("  clean error: could not find raw.json for %s\n", targetURL)
		return
	}

	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		fmt.Printf("  clean error reading %s: %s\n", targetURL, err)
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawBytes, &data); err != nil {
		fmt.Printf("  clean error parsing %s: %s\n", targetURL, err)
		return
	}

	// Use extracted text content for cleaning, not raw HTML
	content, ok := data["content"].(string)
	if !ok || content == "" {
		fmt.Printf("  clean skip: no content for %s\n", targetURL)
		return
	}

	// Skip AI cleaning if content is too short - not worth the tokens
	trimmed := strings.TrimSpace(content)
	if len(trimmed) < minContentLength {
		fmt.Printf("  clean skip: content too short (%d chars) for %s\n", len(trimmed), targetURL)
		w.db.MarkCleaned(targetURL) // mark as done so it does not retry
		return
	}

	// Pre-clean content before sending to AI to reduce token usage
	content = preClean(content)

	// Small delay before hitting AI API to avoid bursting
	time.Sleep(500 * time.Millisecond)

	cleaned, err := w.cleaner.Clean(content)
	if err != nil {
		fmt.Printf("  clean failed %s: %s\n", targetURL, err)
		return
	}

	// Build cleaned.json alongside raw.json
	cleanedData := map[string]interface{}{
		"url":        data["url"],
		"title":      data["title"],
		"crawled_at": data["crawled_at"],
		"cleaned":    cleaned,
		"links":      data["links"],
		"images":     data["images"],
		"videos":     data["videos"],
		"documents":  data["documents"],
		"emails":     data["emails"],
		"phones":     data["phones"],
		"metadata":   data["metadata"],
	}

	cleanedBytes, err := json.MarshalIndent(cleanedData, "", "  ")
	if err != nil {
		fmt.Printf("  clean marshal error %s: %s\n", targetURL, err)
		return
	}

	cleanedPath := filepath.Join(filepath.Dir(rawPath), "cleaned.json")
	if err := os.WriteFile(cleanedPath, cleanedBytes, 0644); err != nil {
		fmt.Printf("  clean write error %s: %s\n", targetURL, err)
		return
	}

	w.db.MarkCleaned(targetURL)
	fmt.Printf("  cleaned %s\n", targetURL)
}

// preClean reduces token usage before sending content to the AI.
// Removes short lines (nav items, labels), deduplicates repeated lines,
// and trims excess whitespace. Typical reduction: 50-70% fewer tokens.
func preClean(content string) string {
	lines := strings.Split(content, "\n")
	seen := map[string]bool{}
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip very short lines - likely nav items, labels, or noise
		if len(trimmed) < 20 {
			continue
		}

		// Skip duplicate lines - common on sites with repeated nav/footer
		lower := strings.ToLower(trimmed)
		if seen[lower] {
			continue
		}

		seen[lower] = true
		result = append(result, trimmed)
	}

	return strings.Join(result, "\n")
}

func (w *Worker) findRawJSON(targetURL string) string {
	domain := extractDomain(targetURL)
	domainDir := filepath.Join(expandHome(w.outputDir), domain)

	var found string
	filepath.Walk(domainDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if info.IsDir() || info.Name() != "raw.json" {
			return nil
		}

		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var data map[string]interface{}
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil
		}

		if u, ok := data["url"].(string); ok && u == targetURL {
			found = path
		}
		return nil
	})

	return found
}

func extractDomain(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
