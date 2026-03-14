# PhantomCrawl

> A 4-layer web crawler with AI cleaning, TLS fingerprinting, and anti-bot evasion. Scraped Cloudflare.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-BSL-green.svg)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-var--raphael-black?logo=github)](https://github.com/var-raphael)

---

## What is PhantomCrawl?

PhantomCrawl is a Go-based web crawler built for developers who need to scrape the real web - not just static HTML. It uses a 4-layer escalation engine that adapts to whatever a site throws at it, from simple HTML to JS-heavy SPAs protected by enterprise anti-bot systems.

It scraped Cloudflare.com - a site protected by Cloudflare - at depth 1, across 100+ pages, with zero blocks. All Layer 1.

---

## How It Works

PhantomCrawl tries the cheapest, fastest method first and only escalates when needed:

| Layer | Method | Use Case |
|-------|--------|----------|
| **Layer 1** | Direct HTTP with TLS fingerprinting (utls HelloChrome_120) | Most of the web - SSR, static, Next.js, etc. |
| **Layer 2** | Network hijacking - extracts embedded JSON, API endpoints, JSON-LD | SPAs that expose data in `window.__NEXT_DATA__` etc. |
| **Layer 2.5** | Browserless REST API | Cloud-based JS rendering without local Chrome |
| **Layer 3** | go-rod headless Chrome (auto-detected) | Full browser rendering for complex SPAs |

Each layer falls back to the next automatically. Most sites never leave Layer 1.

---

## Installation

### Download Binary (Recommended)

Download the pre-built binary for your platform from [GitHub Releases](https://github.com/var-raphael/PhantomCrawl/releases):

```bash
# Linux (64-bit)
wget https://github.com/var-raphael/PhantomCrawl/releases/latest/download/phantomcrawl-linux-amd64
chmod +x phantomcrawl-linux-amd64
sudo mv phantomcrawl-linux-amd64 /usr/local/bin/phantomcrawl

# Linux ARM / Android Termux
wget https://github.com/var-raphael/PhantomCrawl/releases/latest/download/phantomcrawl-linux-arm64
chmod +x phantomcrawl-linux-arm64
mv phantomcrawl-linux-arm64 $PREFIX/bin/phantomcrawl

# Mac (Apple Silicon)
wget https://github.com/var-raphael/PhantomCrawl/releases/latest/download/phantomcrawl-darwin-arm64
chmod +x phantomcrawl-darwin-arm64
sudo mv phantomcrawl-darwin-arm64 /usr/local/bin/phantomcrawl

# Mac (Intel)
wget https://github.com/var-raphael/PhantomCrawl/releases/latest/download/phantomcrawl-darwin-amd64
chmod +x phantomcrawl-darwin-amd64
sudo mv phantomcrawl-darwin-amd64 /usr/local/bin/phantomcrawl

# Windows
# Download phantomcrawl-windows-amd64.exe from releases
# Move it to a folder in your PATH, e.g. C:\Windows\System32\
# Or add its folder to your PATH environment variable
```

Once installed, run from anywhere:
```bash
phantomcrawl init
phantomcrawl start
```

### Build From Source

Requires Go 1.21+

```bash
git clone https://github.com/var-raphael/PhantomCrawl.git
cd PhantomCrawl
go build -ldflags="-s -w" -o phantomcrawl .
```

---

## Quickstart

```bash
# 1. Generate config
phantomcrawl init

# 2. Add your URLs
echo "https://example.com" > urls.txt

# 3. Start crawling
phantomcrawl start
```

Output is saved to `~/phantomcrawl/scraped/` as JSON.

---

## Configuration

Run `phantomcrawl init` to generate a `crawl.json` template, or use the [Config Generator UI](https://github.com/var-raphael/PhantomCrawl/blob/main/ui.html) - open it in any browser, fill the form, download your files.

### Full Reference

```json
{
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
    "enabled": true,
    "save_raw": true,
    "save_cleaned": true,
    "max_concurrent_cleans": 1,
    "prompt": "...",
    "model": "llama-3.3-70b-versatile",
    "provider": "groq",
    "key_rotation": "random",
    "keys": ["$GROQ_KEY_1", "$GROQ_KEY_2"]
  },
  "layer3": {
    "key_rotation": "random",
    "keys": ["$BROWSERLESS_KEY"]
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
      "urls": ["$PROXY_1"]
    }
  }
}
```

| Field | Description |
|-------|-------------|
| `batch_size` | URLs crawled concurrently per batch |
| `throttle` | Seconds to wait between batches (with jitter) |
| `depth` | How deep to follow links from seed URLs. `0` = seed only |
| `depth_limit` | Max links to follow per parent URL. `0` = unlimited |
| `stay_on_domain` | Only follow links within the seed domain |

---

## AI Cleaning

PhantomCrawl uses an LLM to strip boilerplate and extract clean text from raw HTML. Supports **Groq** (free tier) and **OpenAI**.

### Setup

1. Get a free API key from [console.groq.com](https://console.groq.com)
2. Create a `.env` file next to your `crawl.json`:

```env
GROQ_KEY_1=gsk_your_key_here
GROQ_KEY_2=gsk_another_key_here
```

3. Reference keys in `crawl.json`:

```json
"keys": ["$GROQ_KEY_1", "$GROQ_KEY_2"]
```

Multiple keys are rotated automatically to spread token usage across accounts.

### Output

Each crawled URL produces two files:

```
scraped/
  example.com/
    page-title/
      raw.json      ← full extracted data: links, images, metadata, raw HTML
      cleaned.json  ← AI-cleaned text + structured data
```

`cleaned.json` contains the AI-extracted text alongside links, images, emails, and metadata - everything you need in one file.

---

## Proxy Support

Route requests through proxies to avoid IP bans on large crawls.

```env
PROXY_1=http://user:pass@proxy1.example.com:8080
PROXY_2=http://proxy2.example.com:8080
```

```json
"proxy": {
  "enabled": true,
  "key_rotation": "random",
  "urls": ["$PROXY_1", "$PROXY_2"]
}
```

Proxies are tunneled at the TCP level through the utls TLS transport - not just HTTP-level proxying.

---

## Layer 3 - Headless Browser

For JS-heavy SPAs that Layer 1 and 2 can't handle.

**Chrome installed locally** - go-rod is used automatically. No config needed.

**No Chrome (mobile, server)** - use [Browserless](https://browserless.io):

```env
BROWSERLESS_KEY=your_key
```

```json
"layer3": {
  "key_rotation": "random",
  "keys": ["$BROWSERLESS_KEY"]
}
```

---

## Commands

```bash
phantomcrawl init     # Generate crawl.json template
phantomcrawl start    # Start crawling
phantomcrawl reset    # Wipe crawl state (keeps scraped files)
phantomcrawl stats    # Show crawl statistics and URL records
```

### Resuming

PhantomCrawl automatically resumes interrupted crawls. If AI cleaning hits a token quota mid-run, just run `phantomcrawl start` again without resetting - it skips already-crawled URLs and retries only the pending cleans.

---

## Output Format

Output is JSON. To convert to other formats use `jq`, Python's `pandas`, or any JSON parser:

```python
import json, csv

with open('scraped/example.com/page/cleaned.json') as f:
    data = json.load(f)

# Access fields
print(data['cleaned'])   # AI-cleaned text
print(data['links'])     # All extracted links
print(data['images'])    # All image URLs
print(data['metadata'])  # Page metadata
```

---

## License

BSL (Business Source License) - free for personal and non-commercial use. See [LICENSE](LICENSE).

---

## Author

Built by **Raphael Samuel**, 18, Lagos, Nigeria.

Self-taught. Started coding on a phone. No bootcamp, no degree, just code.

- Portfolio: [var-raphael.vercel.app](https://var-raphael.vercel.app)
- GitHub: [github.com/var-raphael](https://github.com/var-raphael)

> *"The kid from Nigeria who built a better Firecrawl."*

---

*If PhantomCrawl helped you, star the repo ⭐*
