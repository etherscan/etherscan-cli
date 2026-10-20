package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/etherscan/etherscan-cli/internal/chains"
)

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
	// DefaultOutput is deliberately left empty rather than seeded: Save writes the whole
	// struct, so seeding it would persist an implicit choice into every user's config.toml
	// (as it once did with "table") and pin them to it across future default changes.
	// Callers resolve an empty value against output.DefaultFormat instead.
	cfg := File{DefaultChain: "1"}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, path, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, path, err
	}
	if cfg.DefaultChain == "" {
		cfg.DefaultChain = "1"
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
	return path, writeFileAtomic(path, cfg)
}

// writeFileAtomic encodes cfg to a private temp file beside the target and renames
// it into place. The config holds the API key, so the result is always 0600 even
// when an older file was more permissive (OpenFile's mode only applies on create),
// and a failed write leaves the previous config intact instead of truncated. A
// symlinked config (e.g. from a dotfiles repo) is updated at its target, so the
// link survives.
func writeFileAtomic(path string, cfg File) (err error) {
	path, err = resolveLink(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".config-*.toml")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
		}
	}()
	if err = tmp.Chmod(0o600); err != nil {
		return err
	}
	if err = toml.NewEncoder(tmp).Encode(cfg); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// resolveLink follows a symlink chain to the file it names, even when that file does
// not exist yet. filepath.EvalSymlinks fails on a dangling link, and renaming over the
// link itself would replace it with a regular file; a dotfiles setup that keeps the
// API-key file out of the repo leaves exactly such a link on a fresh machine.
//
// Relative targets are appended to the link's directory as raw text, never via
// filepath.Join or filepath.Dir: those clean ".." lexically, but when the directory
// before it is itself a symlink the filesystem resolves ".." from the link's target.
// A path that is not a symlink is returned untouched, so plain configs never reach
// EvalSymlinks (which can fail on some Windows network and subst drives).
func resolveLink(path string) (string, error) {
	linked := false
	for range 40 {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) || (err == nil && info.Mode()&os.ModeSymlink == 0) {
			if !linked {
				return path, nil
			}
			return physicalPath(path)
		}
		if err != nil {
			return "", err
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		linked = true
		switch {
		case filepath.IsAbs(target):
			path = target
		case target != "" && os.IsPathSeparator(target[0]): // rooted without a drive (Windows)
			path = filepath.VolumeName(path) + target
		default:
			dir, _ := splitRaw(path)
			path = dir + target
		}
	}
	return "", fmt.Errorf("%s: too many levels of symbolic links", path)
}

// physicalPath turns a path reached through symlinks into a clean one naming the same
// file, so the temp file is created in the directory the rename will land in.
// EvalSymlinks walks the directory part as the filesystem does, ".." included.
func physicalPath(path string) (string, error) {
	dir, base := splitRaw(path)
	if base == "" || base == "." || base == ".." {
		return "", fmt.Errorf("%s: config path does not name a file", path)
	}
	if dir == "" {
		dir = "."
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(realDir, base), nil
}

// splitRaw splits path after its last separator without cleaning either half.
func splitRaw(path string) (dir, base string) {
	i := len(path) - 1
	for i >= len(filepath.VolumeName(path)) && !os.IsPathSeparator(path[i]) {
		i--
	}
	return path[:i+1], path[i+1:]
}

func GetAPIKey(fallback File) (string, string) {
	if key := os.Getenv("ETHERSCAN_API_KEY"); key != "" {
		return key, "env"
	}
	if fallback.APIKey != "" {
		return fallback.APIKey, "config"
	}
	return "", ""
}

// StoreAPIKey records the key in the config struct. It is persisted to the
// plaintext config file by a subsequent Save.
func StoreAPIKey(key string, cfg *File) {
	cfg.APIKey = key
}

// DeleteAPIKey removes the stored key from the config file. It cannot affect the
// ETHERSCAN_API_KEY env var.
func DeleteAPIKey(cfg *File) bool {
	if cfg.APIKey != "" {
		cfg.APIKey = ""
		return true
	}
	return false
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
		chainID, ok := chains.NormalizeID(value)
		if !ok {
			return errors.New("default_chain must be a positive decimal chain ID; run 'etherscan chains' to list supported IDs")
		}
		cfg.DefaultChain = chainID
	case "default_output":
		cfg.DefaultOutput = value
	case "base_url":
		cfg.BaseURL = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}
