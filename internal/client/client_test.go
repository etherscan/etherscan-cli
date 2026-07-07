package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEnvelopeDecodeVariants(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		empty bool
		err   bool
	}{
		{"array", `{"status":"1","message":"OK","result":[{"hash":"0x1"}]}`, false, false},
		{"string", `{"status":"1","message":"OK","result":"123"}`, false, false},
		{"object", `{"status":"1","message":"OK","result":{"SafeGasPrice":"1"}}`, false, false},
		{"empty", `{"status":"0","message":"No transactions found","result":[]}`, true, false},
		{"empty-blank-result", `{"status":"0","message":"No transactions found","result":""}`, true, false},
		{"error", `{"status":"0","message":"NOTOK","result":"Invalid API Key"}`, false, true},
		{"error-blank-result", `{"status":"0","message":"NOTOK","result":""}`, false, true},
		{"rate-limit", `{"status":"0","message":"NOTOK","result":"Max rate limit reached, please use API Key for higher rate limit"}`, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeEnvelope([]byte(tt.body))
			if tt.err && err == nil {
				t.Fatal("expected error")
			}
			if !tt.err && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Empty != tt.empty {
				t.Fatalf("empty=%v, want %v", got.Empty, tt.empty)
			}
		})
	}
}

func TestEnvelopeErrorMessageVerbatim(t *testing.T) {
	// Etherscan's documented error message lives in the `result` field. It must be surfaced
	// verbatim: no "NOTOK" status-word prefix and no leftover JSON quotes.
	cases := []struct {
		name string
		body string
		want string
	}{
		{"rate-limit", `{"status":"0","message":"NOTOK","result":"Max rate limit reached, please use API Key for higher rate limit"}`, "Max rate limit reached, please use API Key for higher rate limit"},
		{"verification", `{"status":"0","message":"NOTOK","result":"Unable to locate ContractCode at 0x123"}`, "Unable to locate ContractCode at 0x123"},
		{"invalid-key", `{"status":"0","message":"NOTOK","result":"Invalid API Key"}`, "Invalid API Key"},
		{"blank-result-falls-back", `{"status":"0","message":"NOTOK","result":""}`, "NOTOK"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeEnvelope([]byte(tc.body))
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tc.want {
				t.Fatalf("message = %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

func TestRPCDecode(t *testing.T) {
	ok, err := decodeRPC([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(ok.Raw) != `"0x1"` {
		t.Fatalf("raw=%s", ok.Raw)
	}
	if _, err := decodeRPC([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"bad"}}`)); err == nil {
		t.Fatal("expected rpc error")
	}
}

func TestClientBuildsRedactedV2Request(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.RawQuery
		fmt.Fprint(w, `{"status":"1","message":"OK","result":"1"}`)
	}))
	defer srv.Close()

	c := New(Options{BaseURL: srv.URL, APIKey: "secret", ChainID: "8453", RateLimit: 1000})
	if _, err := c.Get(context.Background(), "account", "balance", map[string]string{"address": "0x0000000000000000000000000000000000000000"}, true); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"module=account", "action=balance", "chainid=8453", "apikey=secret"} {
		if !strings.Contains(seen, want) {
			t.Fatalf("query %q missing %q", seen, want)
		}
	}
	redacted := RedactURL(srv.URL + "/?apikey=secret")
	if strings.Contains(redacted, "secret") || !strings.Contains(redacted, "REDACTED") {
		t.Fatalf("not redacted: %s", redacted)
	}
	errText := RedactSecrets(`Get "https://api.etherscan.io/v2/api?apikey=secret&module=account": dial failed`)
	if strings.Contains(errText, "secret") || !strings.Contains(errText, "apikey=REDACTED") {
		t.Fatalf("error text not redacted: %s", errText)
	}
}

func TestValidators(t *testing.T) {
	if err := ValidateAddress("address", "0x0000000000000000000000000000000000000000"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAddress("address", "bad"); err == nil {
		t.Fatal("expected bad address error")
	}
	if err := ValidateTxHash("txhash", "0x"+strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateCommaAddresses("addresses", strings.Join([]string{
		"0x0000000000000000000000000000000000000000",
		"0x0000000000000000000000000000000000000001",
	}, ","), 1); err == nil {
		t.Fatal("expected max list error")
	}
	if err := ValidateCommaAddresses("addresses", strings.Join([]string{
		"0x0000000000000000000000000000000000000000",
		"0x0000000000000000000000000000000000000001",
	}, ","), 20); err != nil {
		t.Fatalf("valid address list rejected: %v", err)
	}
	for _, bad := range []string{
		"0x0000000000000000000000000000000000000000,",
		"0x0000000000000000000000000000000000000000,,0x0000000000000000000000000000000000000001",
		",",
	} {
		if err := ValidateCommaAddresses("addresses", bad, 20); err == nil {
			t.Fatalf("expected empty-entry error for %q", bad)
		}
	}
	if err := ValidateDate("startdate", "2026-06-19"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSort("sideways"); err == nil {
		t.Fatal("expected sort error")
	}
}

func TestLimiterRespectsContext(t *testing.T) {
	limiter := NewLimiter(1)
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := limiter.Wait(ctx); err == nil {
		t.Fatal("expected context timeout")
	}
}
