package browser

import (
	"fmt"
	"os/exec"

	"github.com/go-rod/rod"
)

type RodClient struct{}

func NewRodClient() *RodClient {
	return &RodClient{}
}

func (r *RodClient) HasKeys() bool {
	return true // no keys needed, uses local Chrome
}

func (r *RodClient) HijackFetch(url string) (string, error) {
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("")
	defer page.MustClose()

	if err := page.Navigate(url); err != nil {
		return "", fmt.Errorf("rod navigate failed: %w", err)
	}

	if err := page.WaitLoad(); err != nil {
		return "", fmt.Errorf("rod wait failed: %w", err)
	}

	html, err := page.HTML()
	if err != nil {
		return "", fmt.Errorf("rod html failed: %w", err)
	}

	return html, nil
}

func (r *RodClient) Close() {}

// ChromeAvailable checks if Chrome or Chromium is installed on the system
func ChromeAvailable() bool {
	candidates := []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"chrome",
	}
	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil && path != "" {
			return true
		}
	}
	return false
}
