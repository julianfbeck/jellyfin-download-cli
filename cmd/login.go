package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/julianfbeck/jellyfin-download-cli/internal/api"
	"github.com/julianfbeck/jellyfin-download-cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	loginUser          string
	loginPasswordStdin bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your Jellyfin server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, store, err := loadConfig()
		if err != nil {
			return err
		}

		server := cfg.Server
		if serverFlag != "" {
			server = config.NormalizeServerURL(serverFlag)
		}
		if server == "" {
			return exitError(2, fmt.Errorf("server is required. Use --server or set JELLYFIN_SERVER"))
		}

		username := loginUser
		if username == "" && !noInput {
			username = promptLine(fmt.Sprintf("Username [%s]: ", cfg.LastUsername), cfg.LastUsername)
		}
		if username == "" {
			return exitError(2, fmt.Errorf("username is required"))
		}

		password, err := readPassword(loginPasswordStdin)
		if err != nil {
			return err
		}
		if password == "" {
			return exitError(2, fmt.Errorf("password is required"))
		}

		client := newUnauthedClient(server, cfg, timeout)
		resp, err := client.AuthenticateByName(ctx, username, password)
		if err != nil {
			return exitError(4, err)
		}

		cfg.Server = server
		cfg.Token = resp.AccessToken
		cfg.UserID = resp.User.Id
		cfg.LastUsername = username
		if err := config.Save(store, cfg); err != nil {
			return err
		}

		printInfo("Logged in as %s\n", resp.User.Name)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginUser, "user", "", "Username")
	loginCmd.Flags().BoolVar(&loginPasswordStdin, "password-stdin", false, "Read password from stdin")
	rootCmd.AddCommand(loginCmd)
}

func readPassword(fromStdin bool) (string, error) {
	if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}
	if noInput {
		return "", exitError(2, fmt.Errorf("password required; use --password-stdin"))
	}
	fmt.Print("Password: ")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(pw)), nil
}

func promptLine(label, fallback string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(label)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback
	}
	return line
}

func newUnauthedClient(server string, cfg *config.Config, timeout time.Duration) *api.Client {
	return api.NewClient(server, "", "", cfg.DeviceID, cfg.DeviceName, timeout)
}
