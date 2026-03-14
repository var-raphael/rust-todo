package crawler

import "fmt"

func Layer3Fetch(url string, client BrowserClient) FetchResult {
	if client == nil {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("no browser client available - add Browserless keys or install Chrome"),
			Layer: "layer3",
		}
	}

	// Layer 3 uses full render via browser
	// BrowserClient.HijackFetch returns full HTML
	// after JS execution, scroll, and click handling
	html, err := client.HijackFetch(url)
	if err != nil {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("browser fetch failed: %w", err),
			Layer: "layer3",
		}
	}

	if html == "" {
		return FetchResult{
			URL:   url,
			Error: fmt.Errorf("browser returned empty content"),
			Layer: "layer3",
		}
	}

	return FetchResult{
		URL:   url,
		HTML:  html,
		Layer: "layer3",
	}
}