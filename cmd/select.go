package cmd

import (
	"fmt"
	"strings"

	"github.com/julianfbeck/jellyfin-download-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	selectType  string
	selectLimit int
	selectMulti bool
)

var selectCmd = &cobra.Command{
	Use:   "select <query>",
	Short: "Interactively select movies or series",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if noInput {
			return exitError(2, fmt.Errorf("interactive selection disabled by --no-input"))
		}
		client, _, _, err := getClient(true)
		if err != nil {
			return err
		}

		query := strings.Join(args, " ")
		types := parseSearchTypes(selectType)
		items, err := client.SearchItems(ctx, query, types, selectLimit)
		if err != nil {
			return exitError(4, err)
		}
		if len(items) == 0 {
			printInfo("No results\n")
			return nil
		}

		labels := make([]string, len(items))
		for i, item := range items {
			labels[i] = formatItemLabel(item)
		}

		indices, err := ui.PromptSelectIndices("Select item(s):", labels, selectMulti)
		if err != nil {
			return exitError(2, err)
		}

		selected := make([]interface{}, 0, len(indices))
		for _, idx := range indices {
			if idx < 0 || idx >= len(items) {
				continue
			}
			selected = append(selected, items[idx])
		}

		if jsonOutput {
			outputJSON(selected)
			return nil
		}
		for _, idx := range indices {
			item := items[idx]
			fmt.Printf("%s\t%s\t%s\n", item.Id, item.Name, item.Type)
		}
		return nil
	},
}

func init() {
	selectCmd.Flags().StringVar(&selectType, "type", "", "Item type filter: movie, series")
	selectCmd.Flags().IntVar(&selectLimit, "limit", 20, "Max results")
	selectCmd.Flags().BoolVar(&selectMulti, "multi", false, "Allow selecting multiple items")
	rootCmd.AddCommand(selectCmd)
}
