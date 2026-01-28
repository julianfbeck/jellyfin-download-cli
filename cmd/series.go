package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	seriesLimit int
)

var seriesCmd = &cobra.Command{
	Use:   "series",
	Short: "List series",
}

var seriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all series",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := getClient(true)
		if err != nil {
			return err
		}

		items, err := client.SearchItems(ctx, "", []string{"Series"}, seriesLimit)
		if err != nil {
			return exitError(4, err)
		}

		if jsonOutput {
			outputJSON(items)
			return nil
		}
		for _, item := range items {
			fmt.Printf("%s\t%s\n", item.Id, item.Name)
		}
		return nil
	},
}

func init() {
	seriesListCmd.Flags().IntVar(&seriesLimit, "limit", 100, "Max results")
	seriesCmd.AddCommand(seriesListCmd)
	rootCmd.AddCommand(seriesCmd)
}
