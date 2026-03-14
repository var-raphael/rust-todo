package browser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/var-raphael/phantomcrawl/antibot"
	"github.com/var-raphael/phantomcrawl/crawler"
)

type BrowserlessClient struct {
	rotator *antibot.KeyRotator
}

type browserlessRequest struct {
	URL     string                 `json:"url"`
	Options map[string]interface{} `json:"gotoOptions,omitempty"`
}

func NewBrowserlessClient(keys []string, rotation string) *BrowserlessClient {
	return &BrowserlessClient{
		rotator: antibot.NewKeyRotator(keys, rotation),
	}
}

func (b *BrowserlessClient) HasKeys() bool {
	return b.rotator.HasKeys()
}

func (b *BrowserlessClient) HijackFetch(url string) (string, error) {
	key, err := b.rotator.Next()
	if err != nil {
		return "", fmt.Errorf("no browserless keys available")
	}

	// Check if BROWSERLESS_URL is set in env (dev mode)
	endpoint := os.Getenv("BROWSERLESS_URL")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://production-sfo.browserless.io/content?token=%s", key)
	}

	reqBody := browserlessRequest{
		URL: url,
		Options: map[string]interface{}{
			"waitUntil": "networkidle2",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal failed: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request creation failed: %w", err)
	}

	ua := antibot.RandomUserAgent()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", ua)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("browserless request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("browserless rate limited")
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("browserless HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	return string(body), nil
}

func (b *BrowserlessClient) Close() {
	// Nothing to close for REST API
}

// Verify BrowserlessClient implements crawler.BrowserClient
var _ crawler.BrowserClient = (*BrowserlessClient)(nil)
