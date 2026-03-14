package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/var-raphael/phantomcrawl/storage"
)

func humanTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show crawl statistics and URL records",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Init()
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
			os.Exit(1)
		}
		defer db.Close()

		total, failed, err := db.GetStats()
		if err != nil {
			fmt.Println(red("Error: " + err.Error()))
			os.Exit(1)
		}
		cleaned, pending, _ := db.GetCleanStats()

		fmt.Println(bold(green("PhantomCrawl Stats")))
		fmt.Println(dim("------------------"))

		if total == 0 && failed == 0 {
			fmt.Println(yellow("No crawl has been run yet. Run: phantomcrawl start"))
			return
		}

		fmt.Printf("Total crawled : %s\n", green(fmt.Sprintf("%d", total+failed)))
		fmt.Printf("Successful    : %s\n", green(fmt.Sprintf("%d", total)))
		if failed > 0 {
			fmt.Printf("Failed        : %s\n", red(fmt.Sprintf("%d", failed)))
		} else {
			fmt.Printf("Failed        : %s\n", green("0"))
		}
		fmt.Printf("AI cleaned    : %s\n", green(fmt.Sprintf("%d", cleaned)))
		if pending > 0 {
			fmt.Printf("Clean pending : %s\n", yellow(fmt.Sprintf("%d", pending)))
		} else {
			fmt.Printf("Clean pending : %s\n", green("0"))
		}

		records, err := db.GetAllRecords()
		if err != nil || len(records) == 0 {
			return
		}

		// Crawled URLs
		fmt.Println("\n" + bold("Crawled URLs"))
		fmt.Println(dim("------------"))
		for _, r := range records {
			if r.Status != "crawled" {
				continue
			}
			cleanStatus := dim("[ raw ]")
			if r.Cleaned {
				cleanedAgo := ""
				if r.CleanedAt != nil {
					cleanedAgo = dim(" cleaned " + humanTime(*r.CleanedAt))
				}
				cleanStatus = green("[ cleaned ]") + cleanedAgo
			}
			fmt.Printf(
				"  %s %s %s %s\n",
				green("✓"),
				r.URL,
				dim("("+r.LayerUsed+") "+humanTime(r.CrawledAt)),
				cleanStatus,
			)
		}

		// Pending clean
		hasPending := false
		for _, r := range records {
			if r.Status == "crawled" && !r.Cleaned {
				if !hasPending {
					fmt.Println("\n" + bold(yellow("Pending AI Clean")))
					fmt.Println(dim("----------------"))
					hasPending = true
				}
				fmt.Printf("  %s %s %s\n",
					yellow("⏳"),
					r.URL,
					dim("crawled "+humanTime(r.CrawledAt)),
				)
			}
		}

		// Failed URLs
		hasFailed := false
		for _, r := range records {
			if r.Status == "failed" {
				if !hasFailed {
					fmt.Println("\n" + bold(red("Failed URLs")))
					fmt.Println(dim("-----------"))
					hasFailed = true
				}
				reason := ""
				if r.FailureReason != "" {
					reason = dim(" — " + r.FailureReason)
				}
				fmt.Printf("  %s %s %s%s\n",
					red("✗"),
					r.URL,
					dim(humanTime(r.CrawledAt)),
					reason,
				)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
