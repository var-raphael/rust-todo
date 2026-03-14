package crawler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/var-raphael/phantomcrawl/config"
)

func WithRetry(cfg config.RetryConfig, fn func() FetchResult) FetchResult {
	var result FetchResult

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result = fn()

		if result.Error == nil {
			return result
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		wait := backoffDuration(cfg.Backoff, attempt)
		time.Sleep(wait)
	}

	return result
}

func backoffDuration(strategy string, attempt int) time.Duration {
	switch strategy {
	case "exponential":
		seconds := 1 << attempt
		return time.Duration(seconds) * time.Second
	case "linear":
		return time.Duration(attempt*2) * time.Second
	default:
		return 2 * time.Second
	}
}

func HandleRateLimit(resp *http.Response, cfg config.RetryConfig) time.Duration {
	if resp == nil {
		return 60 * time.Second
	}

	if resp.StatusCode != 429 {
		return 0
	}

	if cfg.RespectRetryAfter {
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			var seconds int
			fmt.Sscanf(retryAfter, "%d", &seconds)
			if seconds > 0 {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	return 60 * time.Second
}