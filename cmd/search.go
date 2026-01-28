package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/julianfbeck/jellyfin-download-cli/internal/api"
	"github.com/julianfbeck/jellyfin-download-cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	searchType  string
	searchLimit int
	searchInteractive bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search movies and series",
	Args: func(cmd *cobra.Command, args []string) error {
		if searchInteractive {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("query required")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, _, err := getClient(true)
		if err != nil {
			return err
		}

		query := strings.Join(args, " ")
		types := parseSearchTypes(searchType)

		if searchInteractive {
			selection, err := ui.InteractiveSearch(ctx, "Jellyfin Search", query, func(ctx context.Context, q string) ([]ui.SearchResult, error) {
				items, err := client.SearchItems(ctx, q, types, searchLimit)
				if err != nil {
					return nil, err
				}
				results := make([]ui.SearchResult, 0, len(items))
				for _, item := range items {
					results = append(results, ui.SearchResult{
						ID:    item.Id,
						Name:  item.Name,
						Type:  item.Type,
						Extra: formatItemLabel(item),
					})
				}
				return results, nil
			})
			if err != nil {
				return exitError(2, err)
			}
			if selection == nil {
				return nil
			}
			fmt.Printf("%s\t%s\t%s\n", selection.ID, selection.Name, selection.Type)
			return nil
		}

		items, err := client.SearchItems(ctx, query, types, searchLimit)
		if err != nil {
			return exitError(4, err)
		}

		if jsonOutput {
			outputJSON(items)
			return nil
		}
		if plainOutput {
			for _, item := range items {
				fmt.Printf("%s\t%s\t%s\n", item.Id, item.Name, item.Type)
			}
			return nil
		}

		for _, item := range items {
			fmt.Printf("%s  %s (%s)\n", item.Id, item.Name, item.Type)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Item type filter: movie, series, episode")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "Max results")
	searchCmd.Flags().BoolVarP(&searchInteractive, "interactive", "i", false, "Interactive search UI")
	rootCmd.AddCommand(searchCmd)
}

func parseSearchTypes(value string) []string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return []string{"Movie", "Series"}
	}

	parts := strings.Split(value, ",")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		switch part {
		case "movie", "movies":
			out = append(out, "Movie")
		case "series", "show", "tv":
			out = append(out, "Series")
		case "episode", "episodes":
			out = append(out, "Episode")
		}
	}
	if len(out) == 0 {
		return []string{"Movie", "Series"}
	}
	return out
}

func formatItemLabel(item api.Item) string {
	label := item.Name
	if item.ProductionYear != 0 {
		label = fmt.Sprintf("%s (%d)", label, item.ProductionYear)
	}
	if item.Type != "" {
		label = fmt.Sprintf("%s [%s]", label, item.Type)
	}
	return label
}
