package cmd

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/var-raphael/phantomcrawl/ai"
	"github.com/var-raphael/phantomcrawl/api"
	"github.com/var-raphael/phantomcrawl/antibot"
	"github.com/var-raphael/phantomcrawl/browser"
	"github.com/var-raphael/phantomcrawl/config"
	"github.com/var-raphael/phantomcrawl/crawler"
	"github.com/var-raphael/phantomcrawl/extractor"
	"github.com/var-raphael/phantomcrawl/output"
	"github.com/var-raphael/phantomcrawl/storage"
)

var splash = []string{
	"Built by Raphael Samuel, 18, Nigeria.",
	"Started coding on a phone. No laptop. No excuses.",
	"PhantomCrawl is his 7th shipped product.",
	"Four layer engine. Most scrapers have one.",
	"Self taught. No bootcamp. No degree. Just code.",
	"From Lagos to the world.",
	"Portfolio: var-raphael.vercel.app",
	"GitHub: github.com/var-raphael",
	"If this tool helped you, star the repo.",
	"PhantomCrawl covers 90-95% of the internet.",
	"Layer 2.5 network hijacking. Try finding that elsewhere.",
	"Built in public. BSL licensed. Free forever for personal use.",
	"The kid from Nigeria who built a better Firecrawl.",
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start crawling",
	Run: func(cmd *cobra.Command, args []string) {
		godotenv.Load()

		fmt.Println(bold(green("PhantomCrawl v1.0.0")))
		fmt.Println(dim("-------------------"))
		rand.Seed(time.Now().UnixNano())
		fmt.Printf(dim("  %s\n"), splash[rand.Intn(len(splash))])
		fmt.Println(dim("-------------------"))

		fmt.Println(dim("Loading crawl.json..."))
		cfg, err := config.Load("crawl.json")
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
			os.Exit(1)
		}

		fmt.Println(dim("Initializing state database..."))
		db, err := storage.Init()
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
			os.Exit(1)
		}
		defer db.Close()

		fmt.Printf(dim("Reading %s...\n"), cfg.URLsFile)
		urls, err := readURLs(cfg.URLsFile)
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
			os.Exit(1)
		}
		fmt.Printf(green("✓")+" %d URLs found\n", len(urls))

		// Resume detection
		if cfg.AI.Enabled && cfg.AI.SaveCleaned {
			_, pending, _ := db.GetCleanStats()
			if pending > 0 {
				fmt.Printf(yellow("⚡")+" Resuming — %d URLs pending AI clean from previous run.\n", pending)
			}
		}

		writer := output.New(cfg.Output)
		cleaner := ai.New(cfg.AI)

		aiWorker := ai.NewWorker(cleaner, db, cfg.Output, cfg.AI.MaxConcurrentCleans)
		aiWorker.Start()

		var browserClient crawler.BrowserClient

		if browser.ChromeAvailable() {
			fmt.Println(green("✓") + " Chrome detected. Using go-rod for Layer 3.")
			browserClient = browser.NewRodClient()
		} else {
			bl := browser.NewBrowserlessClient(cfg.Layer3.Keys, cfg.Layer3.KeyRotation)
			if bl.HasKeys() {
				fmt.Println(green("✓") + " Browserless client initialized.")
				browserClient = bl
			} else {
				fmt.Println(yellow("⚠") + " No Chrome or Browserless keys found. Layer 3 unavailable.")
			}
		}

		if cfg.API.Stream {
			server := api.New(cfg, db)
			go func() {
				if err := server.Start(); err != nil {
					fmt.Println(red("API error: " + err.Error()))
				}
			}()
		}

		total := len(urls)
		batches := makeBatches(urls, cfg.BatchSize)
		fmt.Printf("\nStarting %d batches...\n\n", len(batches))

		for batchNum, batch := range batches {
			fmt.Printf(bold("Batch %d/%d\n"), batchNum+1, len(batches))

			var batchWg sync.WaitGroup
			for _, url := range batch {
				batchWg.Add(1)
				go func(u string) {
					defer batchWg.Done()
					fmt.Printf(dim("  crawling %s ...\n"), u)

					c := crawler.New(cfg, db, browserClient)

					for result := range c.CrawlAll(u) {
						if result.Error != nil {
							continue
						}

						data := extractor.Extract(result.URL, result.HTML, result.Layer)

						data.Raw = result.HTML

						if err := writer.Save(data); err != nil {
							fmt.Printf(red("  save error: %s\n"), err)
							continue
						}

						api.BroadcastEvent(fmt.Sprintf(`{"url":"%s","layer":"%s","status":"done"}`, result.URL, result.Layer))
						fmt.Printf(green("  ✓")+" (%s) %s\n", cyan(result.Layer), result.URL)

						if cfg.AI.Enabled && cfg.AI.SaveCleaned {
							fmt.Printf(dim("  queuing AI clean for %s...\n"), result.URL)
							aiWorker.Enqueue(result.URL)
						}
					}
				}(url)
			}
			batchWg.Wait()

			if batchNum < len(batches)-1 {
				wait := antibot.Jitter(cfg.Throttle)
				fmt.Printf(dim("\nWaiting %s before next batch...\n\n"), wait.Round(time.Millisecond))
				time.Sleep(wait)
			}
		}

		// Close queue and wait for all cleanups to finish
		if cfg.AI.Enabled && cfg.AI.SaveCleaned {
			close(aiWorker.Queue)

			done := make(chan struct{})
			go func() {
				aiWorker.Wait()
				close(done)
			}()

			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
		loop:
			for {
				select {
				case <-done:
					break loop
				case <-ticker.C:
					cleaned, pending, _ := db.GetCleanStats()
					fmt.Printf(dim("  AI cleaning... %d cleaned, %d pending\n"), cleaned, pending)
				}
			}

			_, pending2, _ := db.GetCleanStats()
			if pending2 == 0 {
				fmt.Println(green("  ✓ AI cleaning complete."))
			} else {
				fmt.Println(yellow("  ⚠ AI cleaning finished with pending items."))
			}
		}

		total2, failed, _ := db.GetStats()
		cleaned, pending, _ := db.GetCleanStats()

		fmt.Println("\n" + dim("-------------------"))
		fmt.Println(bold("Crawl complete."))
		fmt.Printf("Total crawled : %s (from %d seed URLs)\n", green(fmt.Sprintf("%d", total2)), total)
		if failed > 0 {
			fmt.Printf("Failed        : %s\n", red(fmt.Sprintf("%d", failed)))
		} else {
			fmt.Printf("Failed        : %s\n", green("0"))
		}
		if cfg.AI.Enabled && cfg.AI.SaveCleaned {
			fmt.Printf("AI cleaned    : %s\n", green(fmt.Sprintf("%d", cleaned)))
			if pending > 0 {
				fmt.Printf("Clean pending : %s\n", yellow(fmt.Sprintf("%d", pending)))
				fmt.Println(yellow("⚠") + dim(" Token quota likely reached. Run again when quota resets to resume."))
				uncleaned, _ := db.GetUncleaned()
				for _, u := range uncleaned {
					fmt.Printf(dim("  - %s\n"), u)
				}
			} else {
				fmt.Printf("Clean pending : %s\n", green("0"))
			}
		}
		fmt.Printf("Output        : %s\n", cyan(cfg.Output))
	},
}

func readURLs(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open %s: %w", path, err)
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("no URLs found in %s", path)
	}

	return urls, nil
}

func makeBatches(urls []string, size int) [][]string {
	var batches [][]string
	for i := 0; i < len(urls); i += size {
		end := i + size
		if end > len(urls) {
			end = len(urls)
		}
		batches = append(batches, urls[i:end])
	}
	return batches
}

func init() {
	rootCmd.AddCommand(startCmd)
}
