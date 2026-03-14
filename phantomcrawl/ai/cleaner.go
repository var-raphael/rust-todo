package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/var-raphael/phantomcrawl/antibot"
	"github.com/var-raphael/phantomcrawl/config"
)

const (
	maxChunkRunes = 6000 // ~8k tokens safe limit per chunk
	maxRetries    = 4
)

var rateLimitRe = regexp.MustCompile(`try again in ([0-9]+(?:\.[0-9]+)?)s`)

type Cleaner struct {
	cfg     config.AIConfig
	rotator *antibot.KeyRotator
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CleanRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type CleanResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func New(cfg config.AIConfig) *Cleaner {
	return &Cleaner{
		cfg:     cfg,
		rotator: antibot.NewKeyRotator(cfg.Keys, cfg.KeyRotation),
	}
}

// Clean cleans content - chunks automatically if too large
func (c *Cleaner) Clean(raw string) (string, error) {
	if !c.cfg.Enabled {
		return "", fmt.Errorf("AI cleaning disabled")
	}

	if !c.rotator.HasKeys() {
		return "", fmt.Errorf("no AI keys configured")
	}

	// If content fits in one chunk, clean directly
	if utf8.RuneCountInString(raw) <= maxChunkRunes {
		return c.cleanChunk(raw)
	}

	// Split into chunks and clean each
	chunks := splitIntoChunks(raw, maxChunkRunes)
	var cleaned []string

	for i, chunk := range chunks {
		result, err := c.cleanChunk(fmt.Sprintf("[Part %d/%d]\n%s", i+1, len(chunks), chunk))
		if err != nil {
			return "", fmt.Errorf("chunk %d failed: %w", i+1, err)
		}
		cleaned = append(cleaned, result)
	}

	return strings.Join(cleaned, "\n\n"), nil
}

func (c *Cleaner) cleanChunk(content string) (string, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := c.doRequest(content)
		if err == nil {
			return result, nil
		}

		errMsg := err.Error()

		// Check if it's a rate limit error
		if strings.Contains(errMsg, "Rate limit") || strings.Contains(errMsg, "rate_limit") {
			wait := parseWaitDuration(errMsg)
			if wait < time.Second {
				wait = time.Second // minimum 1s wait
			}
			// Add a small buffer on top of what Groq says
			wait += 500 * time.Millisecond
			fmt.Printf("  rate limited, retrying in %.1fs (attempt %d/%d)...\n", wait.Seconds(), attempt+1, maxRetries)
			time.Sleep(wait)
			lastErr = err
			continue
		}

		// Non-rate-limit error, fail immediately
		return "", err
	}

	return "", fmt.Errorf("rate limit retries exhausted: %w", lastErr)
}

func (c *Cleaner) doRequest(content string) (string, error) {
	key, err := c.rotator.Next()
	if err != nil {
		return "", fmt.Errorf("key rotation failed: %w", err)
	}

	prompt := c.cfg.Prompt
	if prompt == "" {
		prompt = "Remove noise and boilerplate. Extract the main content, normalize it, and return clean structured text."
	}

	endpoint := c.getEndpoint()

	reqBody := CleanRequest{
		Model:     c.cfg.Model,
		MaxTokens: 4096,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt + "\n\nContent:\n" + content,
			},
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

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read failed: %w", err)
	}

	var cleanResp CleanResponse
	if err := json.Unmarshal(body, &cleanResp); err != nil {
		return "", fmt.Errorf("unmarshal failed: %w", err)
	}

	if cleanResp.Error != nil {
		return "", fmt.Errorf("AI error: %s", cleanResp.Error.Message)
	}

	if len(cleanResp.Choices) == 0 {
		return "", fmt.Errorf("empty AI response")
	}

	return cleanResp.Choices[0].Message.Content, nil
}

// parseWaitDuration extracts the wait time from Groq's rate limit error message
func parseWaitDuration(errMsg string) time.Duration {
	matches := rateLimitRe.FindStringSubmatch(errMsg)
	if len(matches) < 2 {
		return 5 * time.Second // default fallback
	}
	secs, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 5 * time.Second
	}
	return time.Duration(secs * float64(time.Second))
}

func splitIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	runes := []rune(text)

	for len(runes) > 0 {
		end := chunkSize
		if end > len(runes) {
			end = len(runes)
		}

		// Try to break at a newline near the chunk boundary
		if end < len(runes) {
			for i := end; i > end-200 && i > 0; i-- {
				if runes[i] == '\n' {
					end = i
					break
				}
			}
		}

		chunks = append(chunks, string(runes[:end]))
		runes = runes[end:]
	}

	return chunks
}

func (c *Cleaner) getEndpoint() string {
	switch c.cfg.Provider {
	case "groq":
		return "https://api.groq.com/openai/v1/chat/completions"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	default:
		return "https://api.groq.com/openai/v1/chat/completions"
	}
}
