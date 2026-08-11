package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/chains"
)

const testContractAddress = "0xBB9bc244D798123fDe783fCc1C72d3Bb8C189413"

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
		if spec.Module != "contract" || spec.Action != "verifysourcecode" || !spec.Post || !spec.Sensitive || !spec.NoRetry {
			t.Fatalf("%s has incorrect request metadata: %+v", name, spec)
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
		name    string
		chain   string
		command string
		flags   []string
		want    map[string]string
	}{
		{
			name:    "solidity",
			chain:   "ethereum",
			command: "verify",
			flags:   []string{"--source-code", "contract C {}", "--codeformat", "solidity-single-file", "--contractname", "C", "--compilerversion", "v0.8.24", "--optimization-used", "0", "--constructor-arguments", "0x00aa", "--license-type", "3"},
			want:    map[string]string{"codeformat": "solidity-single-file", "constructorArguments": "00aa", "optimizationUsed": "0", "licenseType": "3"},
		},
		{
			name:    "abstract zk stack",
			chain:   "abstract",
			command: "verify-zksync",
			flags:   []string{"--source-code", `{}`, "--codeformat", "solidity-standard-json-input", "--contractname", "C.sol:C", "--compilerversion", "v0.8.24", "--zksolc-version", "v1.5.7"},
			want:    map[string]string{"codeformat": "solidity-standard-json-input", "zksolcVersion": "v1.5.7"},
		},
		{
			name:    "vyper",
			chain:   "ethereum",
			command: "verify-vyper",
			flags:   []string{"--source-code", `{}`, "--contractname", "C.vy:C", "--compilerversion", "vyper:0.4.0", "--optimization-used", "1"},
			want:    map[string]string{"codeformat": "vyper-json", "optimizationUsed": "1"},
		},
		{
			name:    "stylus",
			chain:   "arbitrum",
			command: "verify-stylus",
			flags:   []string{"--source-code", "https://github.com/example/project", "--contractname", "project", "--compilerversion", "stylus:0.5.3", "--license-type", "3"},
			want:    map[string]string{"codeformat": "stylus", "sourceCode": "https://github.com/example/project", "licenseType": "3"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got url.Values
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				got = r.Form
				fmt.Fprint(w, `{"status":"1","message":"OK","result":"guid"}`)
			}))
			defer server.Close()

			root := newRootCommand(BuildInfo{}, &fakeUpdateManager{})
			args := []string{"--api-key", "TESTKEY", "--base-url", server.URL, "--chain", test.chain, "--yes", "contract", test.command, testContractAddress}
			root.SetArgs(append(args, test.flags...))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got.Get("module") != "contract" || got.Get("action") != "verifysourcecode" {
				t.Fatalf("wire endpoint = %q/%q", got.Get("module"), got.Get("action"))
			}
			for name, want := range test.want {
				if value := got.Get(name); value != want {
					t.Errorf("%s = %q, want %q", name, value, want)
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
		"contract", "verify-zksync", testContractAddress,
		"--source-code", `{}`, "--codeformat", "solidity-standard-json-input",
		"--contractname", "C.sol:C", "--compilerversion", "v0.8.24", "--zksolc-version", "v1.5.7",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "Abstract") {
		t.Fatalf("error = %v, want Abstract restriction", err)
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
		"contract", "verify-stylus", testContractAddress,
		"--source-code", "https://github.com/example/stylus-contract",
		"--contractname", "C", "--compilerversion", "stylus:0.5.1",
	})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "Arbitrum") {
		t.Fatalf("error = %v, want Arbitrum restriction", err)
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
		{name: "unsupported format", params: map[string]string{"sourceCode": "x", "codeformat": "unknown"}, wantErr: "unsupported codeformat"},
		{name: "oversized source", params: map[string]string{"sourceCode": strings.Repeat("x", maxVerificationSourceBytes+1), "codeformat": "stylus"}, wantErr: "exceeds"},
	} {
		t.Run(test.name, func(t *testing.T) {
			spec := EndpointSpec{Use: "verify <address>", AllowedCodeFormats: []string{"solidity-single-file", "solidity-standard-json-input", "vyper-json", "stylus"}}
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
	cmd, _, err := root.Find([]string{"contract", "verify"})
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
