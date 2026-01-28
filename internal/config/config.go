package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"net/url"
	"strings"
)

const (
	defaultStoreDirName = ".jellyfin-download"
)

type Config struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Token        string `json:"token"`
	DeviceID     string `json:"device_id"`
	DeviceName   string `json:"device_name"`
	DefaultRate  string `json:"default_rate"`
	LastUsername string `json:"last_username"`
}

func ResolveStoreDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	if env := os.Getenv("JELLYFIN_STORE"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}
	return filepath.Join(home, defaultStoreDirName), nil
}

func ConfigPath(storeDir string) string {
	return filepath.Join(storeDir, "config.json")
}

func Load(storeDir string) (*Config, error) {
	path := ConfigPath(storeDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

func Save(storeDir string, cfg *Config) error {
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return fmt.Errorf("creating store dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(ConfigPath(storeDir), data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

func ApplyEnv(cfg *Config) {
	if env := os.Getenv("JELLYFIN_SERVER"); env != "" {
		cfg.Server = env
	}
	if env := os.Getenv("JELLYFIN_TOKEN"); env != "" {
		cfg.Token = env
	}
	if env := os.Getenv("JELLYFIN_USER_ID"); env != "" {
		cfg.UserID = env
	}
	if env := os.Getenv("JELLYFIN_RATE"); env != "" {
		cfg.DefaultRate = env
	}
}

func (c *Config) ValidateAuth() error {
	if c.Server == "" {
		return fmt.Errorf("server not set. Run 'jellyfin-download login' or set JELLYFIN_SERVER")
	}
	if c.Token == "" || c.UserID == "" {
		return fmt.Errorf("not authenticated. Run 'jellyfin-download login'")
	}
	return nil
}

func NormalizeServerURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimRight(raw, "/")
	}
	if parsed.Scheme == "" && parsed.Host == "" && parsed.Path != "" {
		parsed, err = url.Parse("https://" + raw)
		if err != nil {
			return strings.TrimRight(raw, "/")
		}
	}

	parsed.Fragment = ""
	parsed.RawQuery = ""

	if strings.Contains(parsed.Path, "/web") {
		parts := strings.Split(parsed.Path, "/web")
		parsed.Path = parts[0]
	}

	normalized := parsed.String()
	return strings.TrimRight(normalized, "/")
}
