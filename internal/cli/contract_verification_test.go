package cli

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/chains"
)

const (
	testContractAddress       = "0xBB9bc244D798123fDe783fCc1C72d3Bb8C189413"
	testImplementationAddress = "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
)

func TestContractVerificationRegistry(t *testing.T) {
	want := map[string]struct {
		format string
		file   bool
	}{
		"verify":        {file: true},
		"verify-zksync": {file: true},
		"verify-vyper":  {format: "vyper-json", file: true},
		"verify-stylus": {format: "stylus"},
	}
	for _, spec := range endpoints() {
		name := strings.Fields(spec.Use)[0]
		expected, ok := want[name]
		if !ok {
			continue
		}
		// Module stays "contract" (the wire module) while the CLI files the
		// command under the contractverification group.
		if spec.Module != "contract" || spec.Action != "verifysourcecode" || !spec.Post || !spec.Sensitive || !spec.NoRetry {
			t.Fatalf("%s has incorrect request metadata: %+v", name, spec)
		}
		if spec.Group != "contractverification" {
			t.Fatalf("%s CLI group = %q, want contractverification", name, spec.Group)
		}
		if spec.FixedParams["codeformat"] != expected.format || spec.AcceptsFile != expected.file {
			t.Fatalf("%s format/file metadata = %q/%v, want %q/%v", name, spec.FixedParams["codeformat"], spec.AcceptsFile, expected.format, expected.file)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing verification commands: %v", want)
	}
}

func TestVerificationVariantsSendExpectedForm(t *testing.T) {
	tests := []struct {
		name        string
		chain       string
		wantChainID string
		command     string
		flags       []string
		want        map[string]string
	}{
		{
			name:        "solidity",
			chain:       "sepolia",
			wantChainID: "11155111",
			command:     "verify",
			flags:       []string{"--source-code", "contract C {}", "--codeformat", "solidity-single-file", "--contractname", "C", "--compilerversion", "v0.8.24", "--optimization-used", "0", "--constructor-arguments", "0x00aa", "--license-type", "3"},
			want:        map[string]string{"codeformat": "solidity-single-file", "constructorArguments": "00aa", "optimizationUsed": "0", "licenseType": "3"},
		},
		{
			name:        "solidity on mainnet",
			chain:       "ethereum",
			wantChainID: "1",
			command:     "verify",
			flags:       []string{"--source-code", "contract C {}", "--codeformat", "solidity-single-file", "--contractname", "C", "--compilerversion", "v0.8.24", "--optimization-used", "0"},
			want:        map[string]string{"codeformat": "solidity-single-file", "optimizationUsed": "0"},
		},
		{
			name:        "abstract zk stack",
			chain:       "abstract",
			wantChainID: "2741",
			command:     "verify-zksync",
			flags:       []string{"--source-code", `{}`, "--codeformat", "solidity-standard-json-input", "--contractname", "C.sol:C", "--compilerversion", "v0.8.24", "--zksolc-version", "v1.5.7"},
			want:        map[string]string{"codeformat": "solidity-standard-json-input", "zksolcVersion": "v1.5.7"},
		},
		{
			name:        "vyper",
			chain:       "ethereum",
			wantChainID: "1",
			command:     "verify-vyper",
			flags:       []string{"--source-code", `{}`, "--contractname", "C.vy:C", "--compilerversion", "vyper:0.4.0", "--optimization-used", "1"},
			want:        map[string]string{"codeformat": "vyper-json", "optimizationUsed": "1"},
		},
		{
			name:        "stylus",
			chain:       "arbitrum-sepolia",
			wantChainID: "421614",
			command:     "verify-stylus",
			flags:       []string{"--source-code", "https://github.com/example/project", "--contractname", "project", "--compilerversion", "stylus:0.5.3", "--license-type", "3"},
			want:        map[string]string{"codeformat": "stylus", "sourceCode": "https://github.com/example/project", "licenseType": "3"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Read the query and body separately. r.Form merges them on a POST, so
			// an assertion built on it cannot tell which side a parameter is on —
			// which is why chainid landing in the body went unnoticed.
			var got url.Values
			var query url.Values
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				query = r.URL.Query()
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				got, err = url.ParseQuery(string(raw))
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprint(w, `{"status":"1","message":"OK","result":"guid"}`)
			}))
			defer server.Close()

			root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
			args := []string{"--api-key", "TESTKEY", "--base-url", server.URL, "--chain", test.chain, "--yes", "contractverification", test.command, testContractAddress}
			root.SetArgs(append(args, test.flags...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			// module/action/chainid route the request and belong in the URL.
			if query.Get("module") != "contract" || query.Get("action") != "verifysourcecode" {
				t.Fatalf("wire endpoint = %q/%q, want contract/verifysourcecode in the query", query.Get("module"), query.Get("action"))
			}
			// Etherscan routes on the query chainid; in the body it is invisible to
			// the router and the submission is rejected on every non-default chain.
			if got := query.Get("chainid"); got != test.wantChainID {
				t.Errorf("query chainid = %q, want %q", got, test.wantChainID)
			}
			for name, want := range test.want {
				if value := got.Get(name); value != want {
					t.Errorf("body %s = %q, want %q", name, value, want)
				}
			}
		})
	}
}

func TestVerifyZkSyncOnlyAllowsAbstractChains(t *testing.T) {
	var spec EndpointSpec
	for _, candidate := range endpoints() {
		if strings.HasPrefix(candidate.Use, "verify-zksync ") {
			spec = candidate
			break
		}
	}
	for _, name := range []string{"abstract", "abstract-sepolia"} {
		chain, err := chains.Resolve(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateEndpointChain(spec, chain); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
	for _, name := range []string{"ethereum", "arbitrum"} {
		chain, err := chains.Resolve(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateEndpointChain(spec, chain); err == nil || !strings.Contains(err.Error(), "Abstract") {
			t.Errorf("%s error = %v, want Abstract restriction", name, err)
		}
	}
}

func TestVerifyZkSyncRejectsOtherChainBeforeSubmission(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprint(w, `{"status":"1","message":"OK","result":"guid"}`)
	}))
	defer server.Close()

	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	root.SetArgs([]string{
		"--api-key", "TESTKEY", "--base-url", server.URL, "--chain", "ethereum",
		"contractverification", "verify-zksync", testContractAddress,
		"--source-code", `{}`, "--codeformat", "solidity-standard-json-input",
		"--contractname", "C.sol:C", "--compilerversion", "v0.8.24", "--zksolc-version", "v1.5.7",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "Abstract") {
		t.Fatalf("error = %v, want Abstract restriction", err)
	}
	// The message must name the command's real path. It used to hardcode
	// "etherscan contract", which survived the group split silently because the
	// assertion above only matches the chain name.
	if want := "etherscan contractverification verify-zksync"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to name %q", err, want)
	}
	if requests != 0 {
		t.Fatalf("restricted request reached server %d times", requests)
	}
}

func TestVerifyStylusOnlyAllowsArbitrumChains(t *testing.T) {
	var spec EndpointSpec
	for _, candidate := range endpoints() {
		if strings.HasPrefix(candidate.Use, "verify-stylus ") {
			spec = candidate
			break
		}
	}
	for _, name := range []string{"arbitrum", "arbitrum-sepolia"} {
		chain, err := chains.Resolve(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateEndpointChain(spec, chain); err != nil {
			t.Errorf("%s rejected: %v", name, err)
		}
	}
	for _, name := range []string{"ethereum", "abstract"} {
		chain, err := chains.Resolve(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateEndpointChain(spec, chain); err == nil || !strings.Contains(err.Error(), "Arbitrum") {
			t.Errorf("%s error = %v, want Arbitrum restriction", name, err)
		}
	}
}

func TestVerifyStylusRejectsOtherChainBeforeSubmission(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprint(w, `{"status":"1","message":"OK","result":"guid"}`)
	}))
	defer server.Close()

	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	root.SetArgs([]string{
		"--api-key", "TESTKEY", "--base-url", server.URL, "--chain", "ethereum",
		"contractverification", "verify-stylus", testContractAddress,
		"--source-code", "https://github.com/example/stylus-contract",
		"--contractname", "C", "--compilerversion", "stylus:0.5.1",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "Arbitrum") {
		t.Fatalf("error = %v, want Arbitrum restriction", err)
	}
	if want := "etherscan contractverification verify-stylus"; !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to name %q", err, want)
	}
	if requests != 0 {
		t.Fatalf("restricted request reached server %d times", requests)
	}
}

func TestVerificationValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		params  map[string]string
		wantErr string
	}{
		{name: "single file needs optimization", params: map[string]string{"sourceCode": "x", "codeformat": "solidity-single-file"}, wantErr: "optimizationUsed"},
		{name: "standard json does not", params: map[string]string{"sourceCode": "{}", "codeformat": "solidity-standard-json-input"}},
		// Asserts the whole message, not just the prefix: the spec below carries a
		// real Module/Group, so a regression in the command path renders here.
		{name: "unsupported format", params: map[string]string{"sourceCode": "x", "codeformat": "unknown"}, wantErr: `unsupported codeformat "unknown" for etherscan contractverification verify`},
		{name: "oversized source", params: map[string]string{"sourceCode": strings.Repeat("x", maxVerificationSourceBytes+1), "codeformat": "stylus"}, wantErr: "exceeds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := EndpointSpec{Module: "contract", Group: "contractverification", Use: "verify <address>", AllowedCodeFormats: []string{"solidity-single-file", "solidity-standard-json-input", "vyper-json", "stylus"}}
			err := validateSourceVerification(spec, test.params)
			if test.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("error = %v, want %q", err, test.wantErr)
			}
		})
	}

	for _, value := range []string{"0", "1"} {
		if err := validateParams(EndpointSpec{Params: []ParamSpec{req("optimizationUsed", "", KindZeroOne)}}, map[string]string{"optimizationUsed": value}); err != nil {
			t.Errorf("optimization %q rejected: %v", value, err)
		}
	}
	zksyncSpec := EndpointSpec{Use: "verify-zksync <address>", AllowedCodeFormats: []string{"solidity-single-file", "solidity-standard-json-input"}}
	if err := validateSourceVerification(zksyncSpec, map[string]string{"sourceCode": `{}`, "codeformat": "vyper-json", "optimizationUsed": "0"}); err == nil {
		t.Fatal("verify-zksync accepted a Vyper code format")
	}
	if err := validateParams(EndpointSpec{Params: []ParamSpec{req("optimizationUsed", "", KindZeroOne)}}, map[string]string{"optimizationUsed": "yes"}); err == nil {
		t.Fatal("invalid optimization value accepted")
	}
	for _, value := range []string{"1", "14"} {
		if err := validateLicenseType(value); err != nil {
			t.Errorf("license %q rejected: %v", value, err)
		}
	}
	for _, value := range []string{"0", "15", "MIT"} {
		if err := validateLicenseType(value); err == nil {
			t.Errorf("invalid license %q accepted", value)
		}
	}
}

func TestConstructorArgumentNormalization(t *testing.T) {
	for input, want := range map[string]string{"00aa": "00aa", "0x00aa": "00aa", "0X00AA": "00AA", "": ""} {
		got, err := normalizeConstructorArguments(input)
		if err != nil || got != want {
			t.Errorf("normalizeConstructorArguments(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	for _, input := range []string{"abc", "zz"} {
		if _, err := normalizeConstructorArguments(input); err == nil {
			t.Errorf("invalid constructor arguments %q accepted", input)
		}
	}
	if _, err := normalizeConstructorArguments(strings.Repeat("a", maxConstructorArgumentChars+2)); err == nil {
		t.Fatal("oversized constructor arguments accepted")
	}
}

func TestVerificationFileLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	params := map[string]string{}
	if err := populateFileParam(&globalState{file: path}, params); err != nil || params["sourceCode"] != "{}" {
		t.Fatalf("small file = %q, %v", params["sourceCode"], err)
	}

	path = filepath.Join(t.TempDir(), "large.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxVerificationSourceBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := populateFileParam(&globalState{file: path}, map[string]string{}); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized file error = %v", err)
	}
}

func TestVerificationLegacyFlagAliases(t *testing.T) {
	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	cmd, _, err := root.Find([]string{"contractverification", "verify"})
	if err != nil {
		t.Fatal(err)
	}
	for legacy, canonical := range map[string]string{
		"optimizationUsed":     "optimization-used",
		"constructorArguments": "constructor-arguments",
		"evmVersion":           "evm-version",
		"licenseType":          "license-type",
	} {
		if cmd.Flags().Lookup(canonical) == nil {
			t.Errorf("canonical flag --%s missing", canonical)
		}
		alias := cmd.Flags().Lookup(legacy)
		if alias == nil || !alias.Hidden || alias.Deprecated == "" {
			t.Errorf("legacy flag --%s is not a hidden deprecated alias: %+v", legacy, alias)
		}
		if err := cmd.Flags().Set(legacy, "1"); err != nil {
			t.Fatal(err)
		}
		if got, err := cmd.Flags().GetString(canonical); err != nil || got != "1" {
			t.Errorf("--%s did not bind --%s: %q, %v", legacy, canonical, got, err)
		}
	}
}

// TestContractGroupSplit pins the command-tree split: contract is data retrieval
// only, contractverification owns every submission and status poll, and the move
// is CLI-only — all seven keep sending module=contract.
func TestContractGroupSplit(t *testing.T) {
	want := map[string][]string{
		"contract": {"getabi", "getcontractcreation", "getsourcecode"},
		"contractverification": {
			"check-proxy", "check-status", "verify", "verify-proxy",
			"verify-stylus", "verify-vyper", "verify-zksync",
		},
	}
	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	for group, expected := range want {
		cmd, _, err := root.Find([]string{group})
		if err != nil {
			t.Fatalf("group %q not found: %v", group, err)
		}
		if cmd.Name() != group {
			t.Fatalf("group %q resolved to %q", group, cmd.Name())
		}
		var got []string
		for _, sub := range cmd.Commands() {
			got = append(got, sub.Name())
		}
		sort.Strings(got)
		if strings.Join(got, " ") != strings.Join(expected, " ") {
			t.Fatalf("%s subcommands = %v, want %v", group, got, expected)
		}
	}

	// The verification group is a CLI grouping only: the wire module must stay
	// contract, or the requests break and the validation gate in
	// validateParams (keyed on Module) silently stops firing.
	moved := 0
	for _, spec := range endpoints() {
		if spec.Group != "contractverification" {
			continue
		}
		moved++
		if spec.Module != "contract" {
			t.Fatalf("%s wire module = %q, want contract", strings.Fields(spec.Use)[0], spec.Module)
		}
	}
	if moved != 7 {
		t.Fatalf("contractverification spec count = %d, want 7", moved)
	}
}

// TestProxyAndPollWireForms pins the exact requests for the three commands that
// TestVerificationVariantsSendExpectedForm does not cover. Without these, the
// action names and param names of verify-proxy, check-status and check-proxy are
// unasserted, so a typo in any of them ships silently.
func TestProxyAndPollWireForms(t *testing.T) {
	const guid = "abcd1234"
	for _, test := range []struct {
		name       string
		args       []string
		post       bool
		wantAction string
		wantParams map[string]string
	}{
		{
			name:       "verify-proxy",
			args:       []string{"verify-proxy", testContractAddress, "--expectedimplementation", testImplementationAddress},
			post:       true,
			wantAction: "verifyproxycontract",
			wantParams: map[string]string{"address": testContractAddress, "expectedimplementation": testImplementationAddress},
		},
		{
			name:       "check-status",
			args:       []string{"check-status", guid},
			wantAction: "checkverifystatus",
			wantParams: map[string]string{"guid": guid},
		},
		{
			name:       "check-proxy",
			args:       []string{"check-proxy", guid},
			wantAction: "checkproxyverification",
			wantParams: map[string]string{"guid": guid},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			// got holds the endpoint parameters, which live in the body on a POST and
			// the query on a GET. query always holds the routing parameters. Reading
			// them apart is deliberate: r.Form merges body and query on a POST, and
			// r.PostForm cannot see the query at all.
			var got url.Values
			var query url.Values
			var method string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				query = r.URL.Query()
				if r.Method == http.MethodPost {
					raw, err := io.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}
					if got, err = url.ParseQuery(string(raw)); err != nil {
						t.Fatal(err)
					}
				} else {
					got = query
				}
				fmt.Fprint(w, `{"status":"1","message":"OK","result":"ok"}`)
			}))
			defer server.Close()

			root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
			// A non-mainnet chain on purpose: chainid in the body reaches the default
			// upstream, which serves mainnet, so a mainnet case passes even unfixed.
			args := []string{"--api-key", "TESTKEY", "--base-url", server.URL, "--chain", "sepolia", "--yes", "contractverification"}
			root.SetArgs(append(args, test.args...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			wantMethod := http.MethodGet
			if test.post {
				wantMethod = http.MethodPost
			}
			if method != wantMethod {
				t.Errorf("method = %s, want %s", method, wantMethod)
			}
			// The wire module stays "contract" even though the CLI group does not.
			// module, action and chainid route the request, so both verbs put them in
			// the query.
			if query.Get("module") != "contract" || query.Get("action") != test.wantAction {
				t.Fatalf("wire endpoint = %q/%q, want contract/%s in the query", query.Get("module"), query.Get("action"), test.wantAction)
			}
			if chainID := query.Get("chainid"); chainID != "11155111" {
				t.Errorf("query chainid = %q, want 11155111", chainID)
			}
			for name, want := range test.wantParams {
				if value := got.Get(name); value != want {
					t.Errorf("%s = %q, want %q", name, value, want)
				}
			}
		})
	}
}

// TestGroupRejectsUnknownSubcommand: cobra reports an unknown command only at the
// root — under a group it prints help to stdout and exits 0, which would make a
// script still calling a moved or renamed command succeed with help text in its
// output. groupArgs turns that into a real error.
func TestGroupRejectsUnknownSubcommand(t *testing.T) {
	for _, test := range []struct{ name, group, sub, wantHint string }{
		{name: "moved verify", group: "contract", sub: "verify", wantHint: "contractverification"},
		{name: "renamed poll", group: "contract", sub: "verify-status", wantHint: "contractverification"},
		{name: "plain unknown", group: "account", sub: "definitelynotacommand"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
			var stdout, stderr strings.Builder
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			root.SetArgs([]string{test.group, test.sub, "0xabc"})
			err := root.Execute()
			if err == nil {
				t.Fatalf("expected an error, got nil (stdout: %q)", stdout.String())
			}
			if !strings.Contains(err.Error(), `unknown command "`+test.sub+`"`) {
				t.Errorf("error = %q, want it to name the unknown command", err)
			}
			// Help must not be written to stdout on an error path.
			if strings.Contains(stdout.String(), "Available Commands") {
				t.Errorf("help leaked to stdout: %q", stdout.String())
			}
			if test.wantHint != "" && !strings.Contains(err.Error(), test.wantHint) {
				t.Errorf("error = %q, want it to point at %q", err, test.wantHint)
			}
		})
	}
}

// TestGroupWithNoArgsStillPrintsHelp: making the group commands runnable must not
// break the bare "etherscan contract" invocation.
func TestGroupWithNoArgsStillPrintsHelp(t *testing.T) {
	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	var stdout strings.Builder
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"contract"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Available Commands", "getabi", "contractverification"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("bare group help missing %q: %q", want, stdout.String())
		}
	}
}

// TestGroupShortDescriptions pins the parent-group help wording, including the
// default fallback for groups with no override.
func TestGroupShortDescriptions(t *testing.T) {
	for group, want := range map[string]string{
		"contract":             "Etherscan contract data commands",
		"contractverification": "Etherscan contract verification commands",
		"account":              "Etherscan account commands",
	} {
		if got := groupShort(group); got != want {
			t.Errorf("groupShort(%q) = %q, want %q", group, got, want)
		}
	}
}

// TestContractHelpPointsAtVerificationGroup: cobra answers an unknown subcommand
// under a group by printing that group's help, so "etherscan contract verify"
// (the pre-split path) must land on text naming the new group.
func TestContractHelpPointsAtVerificationGroup(t *testing.T) {
	root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
	cmd, _, err := root.Find([]string{"contract"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd.Long, "contractverification") {
		t.Fatalf("contract group help does not point at the verification group: %q", cmd.Long)
	}
}
