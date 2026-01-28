package cmd

import (
	"github.com/julianfbeck/jellyfin-download-cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored Jellyfin credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, store, err := loadConfig()
		if err != nil {
			return err
		}
		cfg.Token = ""
		cfg.UserID = ""
		if err := config.Save(store, cfg); err != nil {
			return err
		}
		printInfo("Logged out\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
