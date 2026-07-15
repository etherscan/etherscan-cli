package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/etherscan/etherscan-cli/internal/config"
)

// TestRuntimeRequiresKey guards the API-key-required policy: with no key from
// flag, env, or config, runtime() must fail fast with errNoAPIKey instead of
// building a client that sends keyless (server-throttled) requests.
func TestRuntimeRequiresKey(t *testing.T) {
	// Isolate from the developer's real env and config file.
	t.Setenv("ETHERSCAN_API_KEY", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	state := &globalState{timeout: 5 * time.Second, rate: 3}
	_, err := runtime(state)
	if !errors.Is(err, errNoAPIKey) {
		t.Fatalf("expected errNoAPIKey, got %v", err)
	}

	// The --api-key flag alone must satisfy the requirement.
	state.apiKey = "TESTKEY"
	if _, err := runtime(state); err != nil {
		t.Fatalf("runtime with --api-key failed: %v", err)
	}

	// So must the environment variable.
	state.apiKey = ""
	t.Setenv("ETHERSCAN_API_KEY", "ENVKEY")
	if _, err := runtime(state); err != nil {
		t.Fatalf("runtime with env key failed: %v", err)
	}
}

// TestRuntimeWithChainOverride: an explicit chain override (from the TUI switcher)
// wins over the flag/env/config precedence; an empty override keeps the default.
func TestRuntimeWithChainOverride(t *testing.T) {
	t.Setenv("ETHERSCAN_API_KEY", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	state := &globalState{apiKey: "TESTKEY", timeout: 5 * time.Second, rate: 3}

	rt, err := runtimeWithChain(state, "polygon")
	if err != nil {
		t.Fatalf("runtimeWithChain(polygon): %v", err)
	}
	if rt.chain.ID != "137" {
		t.Fatalf("override ignored: got %s (%s)", rt.chain.Name, rt.chain.ID)
	}

	rt, err = runtimeWithChain(state, "")
	if err != nil {
		t.Fatalf("runtimeWithChain(default): %v", err)
	}
	if rt.chain.ID != "1" {
		t.Fatalf("empty override should default to ethereum: got %s (%s)", rt.chain.Name, rt.chain.ID)
	}
}

func TestResolveKeyPrecedence(t *testing.T) {
	t.Setenv("ETHERSCAN_API_KEY", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Flag beats everything.
	t.Setenv("ETHERSCAN_API_KEY", "ENVKEY")
	if got := resolveKey(&globalState{apiKey: "FLAGKEY"}, cfg); got != "FLAGKEY" {
		t.Fatalf("flag should win, got %q", got)
	}
	// Env beats config.
	cfg.APIKey = "CFGKEY"
	if got := resolveKey(&globalState{}, cfg); got != "ENVKEY" {
		t.Fatalf("env should beat config, got %q", got)
	}
	// Config is the last resort.
	t.Setenv("ETHERSCAN_API_KEY", "")
	if got := resolveKey(&globalState{}, cfg); got != "CFGKEY" {
		t.Fatalf("config fallback failed, got %q", got)
	}
	// Nothing set -> empty.
	cfg.APIKey = ""
	if got := resolveKey(&globalState{}, cfg); got != "" {
		t.Fatalf("expected empty key, got %q", got)
	}
}
