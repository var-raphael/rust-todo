package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "phantomcrawl",
	Short: "A powerful four layer web crawler",
	Long:  `PhantomCrawl - Intelligent web crawler with AI cleaning and anti-bot evasion.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}