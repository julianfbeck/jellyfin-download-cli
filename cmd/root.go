package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/julianfbeck/jellyfin-download-cli/internal/api"
	"github.com/julianfbeck/jellyfin-download-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	plainOutput bool
	quietMode  bool
	verbose    bool
	noColor    bool
	noInput    bool
	storeDir   string
	serverFlag string
	timeout    time.Duration
	version    = "dev"
	ctx        = context.Background()
)

var rootCmd = &cobra.Command{
	Use:           "jellyfin-download",
	Short:         "Download movies and episodes from Jellyfin",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		handleError(err)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&plainOutput, "plain", false, "Output as plain text")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false, "Suppress non-essential output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolVar(&noInput, "no-input", false, "Disable interactive prompts")
	rootCmd.PersistentFlags().StringVar(&storeDir, "store", "", "Store directory (default: ~/.jellyfin-download)")
	rootCmd.PersistentFlags().StringVar(&serverFlag, "server", "", "Override Jellyfin server URL")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "API request timeout")

	cobra.OnInitialize(func() {
		if jsonOutput && plainOutput {
			plainOutput = false
		}
	})
}

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	return e.Err.Error()
}

func exitError(code int, err error) error {
	return ExitError{Code: code, Err: err}
}

func handleError(err error) {
	var exit ExitError
	if errors.As(err, &exit) {
		printError("%v\n", exit.Err)
		os.Exit(exit.Code)
	}
	printError("%v\n", err)
	os.Exit(1)
}

func outputJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func printInfo(format string, args ...interface{}) {
	if !quietMode {
		fmt.Printf(format, args...)
	}
}

func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
}

func resolveStoreDir() (string, error) {
	return config.ResolveStoreDir(storeDir)
}

func loadConfig() (*config.Config, string, error) {
	store, err := resolveStoreDir()
	if err != nil {
		return nil, "", err
	}

	cfg, err := config.Load(store)
	if err != nil {
		return nil, "", err
	}
	config.ApplyEnv(cfg)

	if serverFlag != "" {
		cfg.Server = serverFlag
	}
	if cfg.Server != "" {
		cfg.Server = config.NormalizeServerURL(cfg.Server)
	}

	if cfg.DeviceID == "" {
		cfg.DeviceID = uuid.NewString()
		if cfg.DeviceName == "" {
			cfg.DeviceName = "jellyfin-download"
		}
		if err := config.Save(store, cfg); err != nil {
			return nil, "", err
		}
	}

	return cfg, store, nil
}

func getClient(requireAuth bool) (*api.Client, *config.Config, string, error) {
	cfg, store, err := loadConfig()
	if err != nil {
		return nil, nil, "", err
	}

	if requireAuth {
		if err := cfg.ValidateAuth(); err != nil {
			return nil, nil, "", exitError(3, err)
		}
	}

	client := api.NewClient(cfg.Server, cfg.Token, cfg.UserID, cfg.DeviceID, cfg.DeviceName, timeout)
	return client, cfg, store, nil
}
