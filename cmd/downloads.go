package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/julianfbeck/jellyfin-download-cli/internal/download"
	"github.com/julianfbeck/jellyfin-download-cli/internal/store"
	"github.com/spf13/cobra"
)

var downloadsCmd = &cobra.Command{
	Use:   "downloads",
	Short: "Manage download queue and progress",
}

var (
	listStatus string
)

var downloadsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, storeDir, err := getClient(false)
		if err != nil {
			return err
		}
		storeDB, err := store.Open(storeDir)
		if err != nil {
			return err
		}
		defer storeDB.Close()

		downloads, err := storeDB.ListDownloads(listStatus)
		if err != nil {
			return err
		}

		if jsonOutput {
			outputJSON(downloads)
			return nil
		}
		for _, d := range downloads {
			fmt.Printf("%d\t%s\t%s\t%s\n", d.ID, d.Status, d.ItemName, d.Path)
		}
		return nil
	},
}

var downloadsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a download record",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _, storeDir, err := getClient(false)
		if err != nil {
			return err
		}
		storeDB, err := store.Open(storeDir)
		if err != nil {
			return err
		}
		defer storeDB.Close()

		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return exitError(2, fmt.Errorf("invalid id"))
		}
		d, err := storeDB.GetDownload(id)
		if err != nil {
			return err
		}
		if d == nil {
			return exitError(2, fmt.Errorf("download not found"))
		}

		if jsonOutput {
			outputJSON(d)
			return nil
		}
		fmt.Printf("%d\t%s\t%s\t%s\n", d.ID, d.Status, d.ItemName, d.Path)
		return nil
	},
}

var downloadsResumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Resume queued or failed downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, cfg, storeDir, err := getClient(true)
		if err != nil {
			return err
		}
		storeDB, err := store.Open(storeDir)
		if err != nil {
			return err
		}
		defer storeDB.Close()

		statuses := []string{"queued", "failed", "downloading"}
		var toResume []store.Download
		for _, status := range statuses {
			items, err := storeDB.ListDownloads(status)
			if err != nil {
				return err
			}
			toResume = append(toResume, items...)
		}

		if len(toResume) == 0 {
			printInfo("No downloads to resume\n")
			return nil
		}

		limiter, err := download.ParseRateLimit(resolveRate(cfg.DefaultRate))
		if err != nil {
			return err
		}

		for _, rec := range toResume {
			item, err := client.GetItem(ctx, rec.ItemID)
			if err != nil {
				return exitError(4, err)
			}
			opts := downloadOptions{
				Rate:         resolveRate(cfg.DefaultRate),
				Output:       filepath.Dir(rec.Path),
				OverridePath: rec.Path,
				Series:       rec.SeriesID.String,
			}
			if err := downloadItem(client, storeDB, *item, filepath.Dir(rec.Path), limiter, opts); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	downloadsListCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (queued, downloading, done, failed)")

	downloadsCmd.AddCommand(downloadsListCmd)
	downloadsCmd.AddCommand(downloadsShowCmd)
	downloadsCmd.AddCommand(downloadsResumeCmd)
	rootCmd.AddCommand(downloadsCmd)
}
