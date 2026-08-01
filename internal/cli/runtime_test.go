package cli

import (
	"errors"
	"testing"
	"time"

	"github.com/etherscan/etherscan-cli/internal/chains"
	"github.com/etherscan/etherscan-cli/internal/client"
	"github.com/etherscan/etherscan-cli/internal/config"
	"github.com/etherscan/etherscan-cli/internal/output"
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

func TestBuildRuntimeAllowsEmptyKeyForTUI(t *testing.T) {
	t.Setenv("ETHERSCAN_API_KEY", "")
	state := &globalState{timeout: 5 * time.Second, rate: 3}
	rt, err := buildRuntime(state, config.File{}, "")
	if err != nil {
		t.Fatalf("keyless TUI runtime failed: %v", err)
	}
	if rt.client == nil || rt.chain.ID != "1" {
		t.Fatalf("incomplete keyless runtime: client=%v chain=%+v", rt.client, rt.chain)
	}
}

func TestRebindRuntimeChainPreservesSession(t *testing.T) {
	ethereum, err := chains.Resolve("ethereum")
	if err != nil {
		t.Fatal(err)
	}
	rt := resolvedRuntime{
		client: client.New(client.Options{BaseURL: "https://example.test/v2/api", APIKey: "TESTKEY", ChainID: ethereum.ID, RateLimit: 3}),
		format: output.JSON,
		chain:  ethereum,
	}
	originalClient := rt.client

	chain, err := rebindRuntimeChain(&rt, "matic")
	if err != nil {
		t.Fatalf("rebindRuntimeChain(matic): %v", err)
	}
	if chain.ID != "137" || rt.chain.ID != chain.ID || rt.chain.Name != chain.Name {
		t.Fatalf("chain not rebound to polygon: %+v", rt.chain)
	}
	if rt.client == originalClient {
		t.Fatal("chain rebind must use an immutable client clone")
	}
	if rt.format != output.JSON {
		t.Fatalf("chain rebind changed output format to %q", rt.format)
	}

	currentClient, currentChainID := rt.client, rt.chain.ID
	if _, err := rebindRuntimeChain(&rt, "not-a-chain"); err == nil {
		t.Fatal("expected unknown-chain error")
	}
	if rt.client != currentClient || rt.chain.ID != currentChainID {
		t.Fatal("failed chain rebind mutated the runtime")
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
