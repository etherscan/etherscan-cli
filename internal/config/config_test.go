package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDeleteAPIKeyClearsPlaintext(t *testing.T) {
	cfg := File{APIKey: "PLAINTEXTKEY"}
	// The key is stored plaintext in the config file; removal clears that field.
	if !DeleteAPIKey(&cfg) {
		t.Fatal("expected DeleteAPIKey to report a removed key")
	}
	if cfg.APIKey != "" {
		t.Fatalf("plaintext api_key not cleared: %q", cfg.APIKey)
	}
	// Second call: nothing left to remove.
	if DeleteAPIKey(&cfg) {
		t.Fatal("expected no key to remove on second call")
	}
}

func TestDefaultChainIsEthereumID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultChain != "1" {
		t.Fatalf("default chain = %q, want 1", cfg.DefaultChain)
	}
}

func TestSetDefaultChainRequiresNumericID(t *testing.T) {
	cfg := File{}
	if err := Set(&cfg, "default_chain=008453"); err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultChain != "8453" {
		t.Fatalf("normalized default chain = %q, want 8453", cfg.DefaultChain)
	}
	for _, value := range []string{"base", "0", "0x2105", ""} {
		if err := Set(&cfg, "default_chain="+value); err == nil {
			t.Errorf("accepted invalid default chain %q", value)
		}
	}
}

// Save replaces the file atomically: the result is always 0600 (OpenFile's mode
// only applied on create, so a 0644 file kept its mode with the key in it), and no
// temp files are left behind.
func TestSaveTightensExistingPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix file modes")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("default_chain = \"1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil { // defeat umask
		t.Fatal(err)
	}
	if _, err := Save(File{DefaultChain: "1", APIKey: "SECRETKEY"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("config mode = %o, want 600", mode)
	}
	assertOnlyConfigInDir(t, path)
}

// A save that fails at the final step reports the error and leaves no temp file.
// The previous config is only ever replaced by that rename, so it stays intact;
// before, O_TRUNC emptied it before encoding started.
func TestFailedSaveCleansUpTempFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, _ := DefaultPath()
	// Rename cannot replace a non-empty directory, so the temp file is written and
	// then the rename fails.
	if err := os.MkdirAll(filepath.Join(path, "blocker"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(File{DefaultChain: "1", APIKey: "NEWKEY"}); err == nil {
		t.Fatal("expected Save to fail when the config path is a directory")
	}
	if _, err := os.Stat(filepath.Join(path, "blocker")); err != nil {
		t.Fatalf("existing config path was disturbed: %v", err)
	}
	assertOnlyConfigInDir(t, path)
}

// A config symlinked from elsewhere (e.g. a dotfiles repo) is updated at its
// target; the link itself must survive the rename.
func TestSaveWritesThroughSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, _ := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "dotfiles-config.toml")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := Save(File{DefaultChain: "1", APIKey: "LINKEDKEY"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink was replaced: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), "LINKEDKEY") {
		t.Fatalf("symlink target not updated: %q, %v", data, err)
	}
}

// A dotfiles setup that keeps the API-key file out of the repo leaves a dangling link
// on a fresh machine; saving must create the target rather than replace the link.
func TestSaveWritesThroughDanglingSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, _ := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "dotfiles-config.toml")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	assertSavedThroughLink(t, path, target)
}

func TestSaveWritesThroughDanglingRelativeSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	path, _ := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	dotfiles := filepath.Join(xdg, "dotfiles")
	if err := os.MkdirAll(dotfiles, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "dotfiles", "config.toml"), path); err != nil {
		t.Fatal(err)
	}
	assertSavedThroughLink(t, path, filepath.Join(dotfiles, "config.toml"))
}

// ".." in a relative link target must be resolved the way the filesystem does: after
// following any symlinked directory before it, not by trimming the path as text.
func TestSaveResolvesRelativeLinksPhysically(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, tmp, configs string) (link string)
		want  string // relative to tmp
		wrong string // where lexical cleaning would have written
	}{
		{
			name: "config dir is a symlink",
			setup: func(t *testing.T, tmp, configs string) string {
				return filepath.Join("..", "target.toml")
			},
			want:  filepath.Join("real", "target.toml"),
			wrong: filepath.Join("xdg", "target.toml"),
		},
		{
			name: "symlinked dir then .. inside the target",
			setup: func(t *testing.T, tmp, configs string) string {
				deep := filepath.Join(tmp, "elsewhere", "deep", "dir")
				if err := os.MkdirAll(deep, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(deep, filepath.Join(configs, "sub")); err != nil {
					t.Fatal(err)
				}
				return "sub/../target.toml" // not filepath.Join: it would clean away the ".."
			},
			want:  filepath.Join("elsewhere", "deep", "target.toml"),
			wrong: filepath.Join("real", "configs", "target.toml"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			configs := filepath.Join(tmp, "real", "configs")
			if err := os.MkdirAll(configs, 0o700); err != nil {
				t.Fatal(err)
			}
			xdg := filepath.Join(tmp, "xdg")
			if err := os.MkdirAll(xdg, 0o700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("XDG_CONFIG_HOME", xdg)
			path, _ := DefaultPath() // <xdg>/etherscan/config.toml
			if err := os.Symlink(configs, filepath.Dir(path)); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(tc.setup(t, tmp, configs), path); err != nil {
				t.Fatal(err)
			}
			assertSavedThroughLink(t, path, filepath.Join(tmp, tc.want))
			if _, err := os.Lstat(filepath.Join(tmp, tc.wrong)); !os.IsNotExist(err) {
				t.Fatalf("config was written to the lexically cleaned path %s", tc.wrong)
			}
		})
	}
}

func assertSavedThroughLink(t *testing.T, path, target string) {
	t.Helper()
	if _, err := Save(File{DefaultChain: "1", APIKey: "LINKEDKEY"}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(path); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink was replaced: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil || !strings.Contains(string(data), "LINKEDKEY") {
		t.Fatalf("symlink target not created: %q, %v", data, err)
	}
	if info, err := os.Stat(target); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("target should be 0600: %v, %v", info, err)
	}
}

func TestSaveRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := File{APIKey: "ROUNDTRIP", DefaultChain: "8453", DefaultOutput: "table"}
	for i := 0; i < 2; i++ { // second save replaces an existing file
		if _, err := Save(want); err != nil {
			t.Fatal(err)
		}
	}
	got, path, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Load = %+v, want %+v", got, want)
	}
	assertOnlyConfigInDir(t, path)
}

func assertOnlyConfigInDir(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Errorf("leftover file in config dir: %s", e.Name())
		}
	}
}
