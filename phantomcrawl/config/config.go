package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type RetryConfig struct {
	MaxAttempts       int    `json:"max_attempts"`
	Backoff           string `json:"backoff"`
	RespectRetryAfter bool   `json:"respect_retry_after"`
}

type ScrapeConfig struct {
	Text         bool   `json:"text"`
	Links        bool   `json:"links"`
	Images       string `json:"images"`
	Videos       string `json:"videos"`
	Documents    string `json:"documents"`
	Emails       bool   `json:"emails"`
	PhoneNumbers bool   `json:"phone_numbers"`
	Metadata     bool   `json:"metadata"`
}

type AIConfig struct {
	Enabled             bool     `json:"enabled"`
	SaveRaw             bool     `json:"save_raw"`
	SaveCleaned         bool     `json:"save_cleaned"`
	Prompt              string   `json:"prompt"`
	Model               string   `json:"model"`
	Provider            string   `json:"provider"`
	KeyRotation         string   `json:"key_rotation"`
	Keys                []string `json:"keys"`
	MaxConcurrentCleans int      `json:"max_concurrent_cleans"`
}

type Layer3Config struct {
	KeyRotation string   `json:"key_rotation"`
	Keys        []string `json:"keys"`
}

type APIConfig struct {
	Port   int  `json:"port"`
	Stream bool `json:"stream"`
}

type ProxyConfig struct {
	Enabled     bool     `json:"enabled"`
	KeyRotation string   `json:"key_rotation"`
	URLs        []string `json:"urls"`
}

type CaptchaConfig struct {
	Enabled     bool     `json:"enabled"`
	Provider    string   `json:"provider"`
	KeyRotation string   `json:"key_rotation"`
	Keys        []string `json:"keys"`
}

type AntiBotConfig struct {
	RotateUserAgents   bool          `json:"rotate_user_agents"`
	RequestDelayJitter bool          `json:"request_delay_jitter"`
	Proxy              ProxyConfig   `json:"proxy"`
	CaptchaSolver      CaptchaConfig `json:"captcha_solver"`
}

type Config struct {
	URLsFile         string        `json:"urls_file"`
	BatchSize        int           `json:"batch_size"`
	Throttle         int           `json:"throttle"`
	Depth            int           `json:"depth"`
	DepthLimit       int           `json:"depth_limit"`
	StayOnDomain     bool          `json:"stay_on_domain"`
	Output           string        `json:"output"`
	Retry            RetryConfig   `json:"retry"`
	Scrape           ScrapeConfig  `json:"scrape"`
	AI               AIConfig      `json:"ai"`
	Layer3           Layer3Config  `json:"layer3"`
	API              APIConfig     `json:"api"`
	AntiBot          AntiBotConfig `json:"anti_bot"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid crawl.json: %w", err)
	}

	// Resolve $ENV_VAR references in all key slices
	cfg.AI.Keys = resolveEnvVars(cfg.AI.Keys)
	cfg.Layer3.Keys = resolveEnvVars(cfg.Layer3.Keys)
	cfg.AntiBot.Proxy.URLs = resolveEnvVars(cfg.AntiBot.Proxy.URLs)
	cfg.AntiBot.CaptchaSolver.Keys = resolveEnvVars(cfg.AntiBot.CaptchaSolver.Keys)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// resolveEnvVars replaces $VAR_NAME entries with their env values, dropping empty ones
func resolveEnvVars(keys []string) []string {
	var resolved []string
	for _, k := range keys {
		if strings.HasPrefix(k, "$") {
			val := os.Getenv(strings.TrimPrefix(k, "$"))
			if val != "" {
				resolved = append(resolved, val)
			}
		} else if k != "" {
			resolved = append(resolved, k)
		}
	}
	return resolved
}

func validate(cfg *Config) error {
	if cfg.URLsFile == "" {
		return fmt.Errorf("urls_file is required in crawl.json")
	}
	if cfg.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be greater than 0")
	}
	if cfg.Throttle < 0 {
		return fmt.Errorf("throttle cannot be negative")
	}
	if cfg.Depth < 0 {
		return fmt.Errorf("depth must be 0 or greater")
	}
	return nil
}
