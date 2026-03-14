package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/var-raphael/phantomcrawl/storage"
)

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Wipe state and start fresh",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := storage.Init()
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		defer db.Close()

		if err := db.Reset(); err != nil {
			fmt.Println("Error resetting state:", err)
			os.Exit(1)
		}

		fmt.Println("State wiped. Ready for a fresh crawl.")
	},
}

func init() {
	rootCmd.AddCommand(resetCmd)
}