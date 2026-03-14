package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a crawl.json template",
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := os.Stat("crawl.json"); err == nil {
			fmt.Println("crawl.json already exists")
			return
		}

		template := `{
  "urls_file": "./urls.txt",
  "batch_size": 3,
  "throttle": 5,
  "depth": 0,
  "depth_limit": 10,
  "stay_on_domain": true,
  "output": "~/phantomcrawl/scraped",
  "retry": {
    "max_attempts": 3,
    "backoff": "exponential",
    "respect_retry_after": true
  },
  "scrape": {
    "text": true,
    "links": true,
    "images": "links_only",
    "videos": "links_only",
    "documents": "links_only",
    "emails": false,
    "phone_numbers": false,
    "metadata": true
  },
  "ai": {
    "enabled": false,
    "save_raw": true,
    "save_cleaned": false,
    "max_concurrent_cleans": 1,
    "prompt": "You are a web content extractor. You will receive extracted text from a webpage. Your job:\n1. Remove nav menus, footers, cookie banners, and boilerplate\n2. Extract only meaningful content: headings, paragraphs, lists\n3. Return plain text only — no HTML, no JSON, no commentary\n4. If the page has no meaningful content, return: NO_CONTENT\nReturn only the extracted text. No commentary, no notes, no explanations.",
    "model": "llama-3.3-70b-versatile",
    "provider": "groq",
    "key_rotation": "random",
    "keys": []
  },
  "layer3": {
    "key_rotation": "random",
    "keys": []
  },
  "api": {
    "port": 4000,
    "stream": true
  },
  "anti_bot": {
    "rotate_user_agents": true,
    "request_delay_jitter": true,
    "proxy": {
      "enabled": false,
      "key_rotation": "random",
      "urls": []
    },
    "captcha_solver": {
      "enabled": false,
      "provider": "2captcha",
      "key_rotation": "random",
      "keys": []
    }
  }
}`

		err := os.WriteFile("crawl.json", []byte(template), 0644)
		if err != nil {
			fmt.Println("Error creating crawl.json:", err)
			return
		}

		fmt.Println("crawl.json created. Edit it and add your URLs to urls.txt")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
