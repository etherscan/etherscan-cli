package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/zalando/go-keyring"
)

const serviceName = "etherscan-cli"
const keyringAccount = "default"

type File struct {
	APIKey        string `toml:"api_key,omitempty"`
	DefaultChain  string `toml:"default_chain,omitempty"`
	DefaultOutput string `toml:"default_output,omitempty"`
	BaseURL       string `toml:"base_url,omitempty"`
}

func DefaultPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "etherscan", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".etherscan", "config.toml"), nil
}

func Load() (File, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return File{}, "", err
	}
	cfg := File{DefaultChain: "ethereum", DefaultOutput: "table"}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, path, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, path, err
	}
	if cfg.DefaultChain == "" {
		cfg.DefaultChain = "ethereum"
	}
	if cfg.DefaultOutput == "" {
		cfg.DefaultOutput = "table"
	}
	return cfg, path, nil
}

func Save(cfg File) (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return path, toml.NewEncoder(f).Encode(cfg)
}

func GetAPIKey(fallback File) (string, string) {
	if key := os.Getenv("ETHERSCAN_API_KEY"); key != "" {
		return key, "env"
	}
	if key, err := keyring.Get(serviceName, keyringAccount); err == nil && key != "" {
		return key, "keyring"
	}
	if fallback.APIKey != "" {
		return fallback.APIKey, "config"
	}
	return "", ""
}

func StoreAPIKey(key string, cfg *File) string {
	if err := keyring.Set(serviceName, keyringAccount, key); err == nil {
		cfg.APIKey = ""
		return "keyring"
	}
	cfg.APIKey = key
	return "config"
}

// DeleteAPIKey removes the stored key from both the OS keyring and any plaintext
// fallback in the config file. It cannot affect the ETHERSCAN_API_KEY env var.
func DeleteAPIKey(cfg *File) bool {
	removed := false
	if err := keyring.Delete(serviceName, keyringAccount); err == nil {
		removed = true
	}
	if cfg.APIKey != "" {
		cfg.APIKey = ""
		removed = true
	}
	return removed
}

func Redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "REDACTED"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func Set(cfg *File, assignment string) error {
	var key, value string
	for i, r := range assignment {
		if r == '=' {
			key, value = assignment[:i], assignment[i+1:]
			break
		}
	}
	if key == "" {
		return fmt.Errorf("setting must be key=value")
	}
	switch key {
	case "api_key":
		cfg.APIKey = value
	case "default_chain":
		cfg.DefaultChain = value
	case "default_output":
		cfg.DefaultOutput = value
	case "base_url":
		cfg.BaseURL = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
