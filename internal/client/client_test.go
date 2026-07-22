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

func TestRPCDecodeEnvelopeError(t *testing.T) {
	// Pre-dispatch guard errors (rate limit, invalid key, unsupported chainid, throttle) reach
	// proxy endpoints in the standard Etherscan envelope, not JSON-RPC. decodeRPC must surface
	// them verbatim as errors, matching decodeEnvelope, instead of returning them as a result.
	cases := []struct {
		name string
		body string
		want string
	}{
		{"invalid-key", `{"status":"0","message":"NOTOK","result":"Invalid API Key (#err2)"}`, "Invalid API Key (#err2)"},
		{"rate-limit", `{"status":"0","message":"NOTOK","result":"Max rate limit reached, please use API Key for higher rate limit"}`, "Max rate limit reached, please use API Key for higher rate limit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeRPC([]byte(tc.body))
			if err == nil {
				t.Fatal("expected error, got nil (envelope error leaked as success)")
			}
			if err.Error() != tc.want {
				t.Fatalf("message = %q, want %q", err.Error(), tc.want)
			}
		})
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

func TestClientForChainPreservesSession(t *testing.T) {
	c := New(Options{BaseURL: "https://example.test/v2/api", APIKey: "secret", ChainID: "1", RateLimit: 3})
	switched := c.ForChain("137")

	if switched == c {
		t.Fatal("ForChain must return a distinct client")
	}
	if c.chainID != "1" || switched.chainID != "137" {
		t.Fatalf("chain IDs changed incorrectly: original=%q switched=%q", c.chainID, switched.chainID)
	}
	if switched.limiter != c.limiter {
		t.Fatal("chain switch must preserve the session rate limiter")
	}
	if switched.http != c.http {
		t.Fatal("chain switch must preserve the HTTP transport")
	}
	if switched.baseURL != c.baseURL || switched.apiKey != c.apiKey || switched.retries != c.retries {
		t.Fatal("chain switch changed resolved client settings")
	}
}

func TestWithAPIKeyClonesWithoutMutatingOriginal(t *testing.T) {
	var queries []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.RawQuery)
		fmt.Fprint(w, `{"status":"1","message":"OK","result":"1"}`)
	}))
	defer srv.Close()

	original := New(Options{BaseURL: srv.URL, ChainID: "1", RateLimit: 1000})
	withKey := original.WithAPIKey("TESTKEY")
	if _, err := original.Get(context.Background(), "account", "balance", nil, false); err != nil {
		t.Fatal(err)
	}
	if _, err := withKey.Get(context.Background(), "account", "balance", nil, false); err != nil {
		t.Fatal(err)
	}
	if len(queries) != 2 || strings.Contains(queries[0], "apikey=") || !strings.Contains(queries[1], "apikey=TESTKEY") {
		t.Fatalf("unexpected original/clone queries: %v", queries)
	}
}

func TestChainList(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		fmt.Fprint(w, `{"comments":"ok","totalcount":2,"result":[{"chainname":"Ethereum Mainnet","chainid":"1","blockexplorer":"https://etherscan.io","apiurl":"https://api.etherscan.io/v2/api?chainid=1","status":1},{"chainname":"Base Mainnet","chainid":"8453","blockexplorer":"https://basescan.org","apiurl":"https://api.etherscan.io/v2/api?chainid=8453","status":1}]}`)
	}))
	defer srv.Close()

	// Mirror the real base URL shape: <host>/v2/api -> chainlist at <host>/v2/chainlist.
	c := New(Options{BaseURL: srv.URL + "/v2/api", APIKey: "secret", ChainID: "1", RateLimit: 1000})
	res, err := c.ChainList(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v2/chainlist" {
		t.Fatalf("request path = %q, want /v2/chainlist", gotPath)
	}
	// The endpoint takes no parameters: no module/action/apikey/chainid may be sent.
	if gotQuery != "" {
		t.Fatalf("chainlist must send no query params, got %q", gotQuery)
	}
	chains, err := DecodeResult[[]map[string]any](res.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(chains) != 2 || chains[0]["chainname"] != "Ethereum Mainnet" || chains[1]["chainid"] != "8453" {
		t.Fatalf("unexpected chainlist rows: %v", chains)
	}
}

func TestChainListDecodeErrors(t *testing.T) {
	if _, err := decodeChainList([]byte(`not json`)); err == nil {
		t.Fatal("expected error on malformed body")
	}
	if _, err := decodeChainList([]byte(`{"comments":"x","totalcount":0}`)); err == nil {
		t.Fatal("expected error on missing result field")
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
