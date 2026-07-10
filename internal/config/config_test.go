package config

import "testing"

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
