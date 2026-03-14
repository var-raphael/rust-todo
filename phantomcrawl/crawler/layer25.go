package crawler

import (
	"fmt"
	"strings"
)

type HijackResult struct {
	URL      string
	Data     string
	Layer    string
	Error    error
}

// BrowserClient interface so both Browserless
// and go-rod implement the same contract
type BrowserClient interface {
	HijackFetch(url string) (string, error)
	Close()
}

func Layer25Fetch(url string, client BrowserClient) FetchResult {
	if client == nil {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("no browser client available"),
			Layer: "layer2.5",
		}
	}

	data, err := client.HijackFetch(url)
	if err != nil {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("hijack failed: %w", err),
			Layer: "layer2.5",
		}
	}

	// Must be meaningful JSON
	if data == "" || !isValidJSON(data) {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("no clean JSON intercepted"),
			Layer: "layer2.5",
		}
	}

	// Filter out tiny responses
	if len(strings.TrimSpace(data)) < 100 {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("intercepted data too small"),
			Layer: "layer2.5",
		}
	}

	return FetchResult{
		URL:   url,
		HTML:  data,
		Layer: "layer2.5",
	}
}