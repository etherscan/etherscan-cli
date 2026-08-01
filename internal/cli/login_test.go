package cli

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

// newLoginTest points the config at a temp dir and stands up a fake API so
// validateKeyLive can succeed or fail on demand without touching the network.
// ETHERSCAN_PLAIN_PROMPT forces the readSecret branch: `go test` has no TTY so
// keyPromptTTY would be false anyway, but pinning it keeps the test honest if
// it is ever run from a terminal.
func newLoginTest(t *testing.T, validKey bool) (configPath string, baseURL string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ETHERSCAN_PLAIN_PROMPT", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if validKey {
			_, _ = w.Write([]byte(`{"status":"1","message":"OK","result":{"creditsUsed":1,"creditLimit":100000}}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"0","message":"NOTOK","result":"Invalid API Key"}`))
	}))
	t.Cleanup(srv.Close)

	return filepath.Join(dir, "etherscan", "config.toml"), srv.URL
}

func runLogin(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newRootCommand(BuildInfo{Version: "1.1.0"}, &fakeUpdateManager{})
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"login"}, args...))
	err = root.Execute()
	return out.String(), errOut.String(), err
}

// Piped stdin must keep working: this is the automation path and it predates the
// branded prompt.
func TestLoginReadsPipedKey(t *testing.T) {
	cfgPath, baseURL := newLoginTest(t, true)

	stdout, _, err := runLogin(t, "SomeApiKeyValue123\n", "--base-url", baseURL)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if !strings.Contains(stdout, "API Key saved to") {
		t.Errorf("missing success line, got %q", stdout)
	}
	if strings.Contains(stdout, "SomeApiKeyValue123") {
		t.Errorf("success line leaked the raw key: %q", stdout)
	}
	if !strings.Contains(stdout, maskKey("SomeApiKeyValue123")) {
		t.Errorf("success line should show the masked key, got %q", stdout)
	}
	data, readErr := os.ReadFile(cfgPath)
	if readErr != nil {
		t.Fatalf("config not written: %v", readErr)
	}
	if !strings.Contains(string(data), "SomeApiKeyValue123") {
		t.Errorf("key not persisted, config is:\n%s", data)
	}
}

// --api-key short-circuits both prompts; stdin must never be touched.
func TestLoginFlagKeySkipsPrompt(t *testing.T) {
	cfgPath, baseURL := newLoginTest(t, true)

	root := newRootCommand(BuildInfo{Version: "1.1.0"}, &fakeUpdateManager{})
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(iotest.ErrReader(errors.New("stdin must not be read")))
	root.SetArgs([]string{"login", "--api-key", "FlagSuppliedKey99", "--base-url", baseURL})
	if err := root.Execute(); err != nil {
		t.Fatalf("login with --api-key: %v", err)
	}
	if !strings.Contains(out.String(), "API Key saved to") {
		t.Errorf("missing success line, got %q", out.String())
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if !strings.Contains(string(data), "FlagSuppliedKey99") {
		t.Errorf("flag key not persisted, config is:\n%s", data)
	}
}

// A key the API rejects must leave nothing behind: no config file, no partial write.
func TestLoginValidationFailureWritesNothing(t *testing.T) {
	cfgPath, baseURL := newLoginTest(t, false)

	stdout, _, err := runLogin(t, "BadKeyValue123\n", "--base-url", baseURL)
	if err == nil {
		t.Fatal("expected login to fail against a rejecting API")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("unexpected error %v", err)
	}
	if strings.Contains(stdout, "API Key saved") {
		t.Errorf("failed login should print no success line, got %q", stdout)
	}
	if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
		t.Errorf("failed login must not create %s", cfgPath)
	}
}

func TestLoginRejectsEmptyAndMalformedKeys(t *testing.T) {
	for name, stdin := range map[string]string{
		"empty":      "\n",
		"whitespace": "AB CD\n",
	} {
		t.Run(name, func(t *testing.T) {
			cfgPath, baseURL := newLoginTest(t, true)
			if _, _, err := runLogin(t, stdin, "--base-url", baseURL); err == nil {
				t.Fatal("expected an error")
			}
			if _, statErr := os.Stat(cfgPath); !os.IsNotExist(statErr) {
				t.Errorf("rejected key must not create %s", cfgPath)
			}
		})
	}
}

func TestReadSecretNonTTY(t *testing.T) {
	var out bytes.Buffer
	got, err := readSecret("key: ", strings.NewReader("  PipedKey  \n"), &out)
	if err != nil {
		t.Fatalf("readSecret: %v", err)
	}
	if got != "PipedKey" {
		t.Errorf("readSecret = %q; want %q", got, "PipedKey")
	}
	if out.String() != "key: " {
		t.Errorf("prompt should be written to out, got %q", out.String())
	}
}

// The plain form must stay byte-identical to previous releases so anything
// scraping it keeps working; colour is additive only.
func TestLoginSavedLine(t *testing.T) {
	plain := loginSavedLine("/tmp/config.toml", "abc***xyz", false)
	if plain != "API Key saved to /tmp/config.toml! Key: abc***xyz" {
		t.Errorf("plain saved line changed: %q", plain)
	}
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("plain saved line must carry no escapes: %q", plain)
	}

	colored := loginSavedLine("/tmp/config.toml", "abc***xyz", true)
	if !strings.Contains(colored, "\x1b[") {
		t.Error("colored saved line should carry escapes")
	}
	if stripANSI(colored) != "✓ API Key saved to /tmp/config.toml! Key: abc***xyz" {
		t.Errorf("colored saved line reads %q", stripANSI(colored))
	}
}

// The branded prompt must not be attempted when it cannot be drawn, or login
// would hang against a pipe.
func TestKeyPromptTTYRespectsOverrides(t *testing.T) {
	t.Setenv("ETHERSCAN_PLAIN_PROMPT", "1")
	if keyPromptTTY() {
		t.Error("ETHERSCAN_PLAIN_PROMPT must force the plain reader")
	}
	t.Setenv("ETHERSCAN_PLAIN_PROMPT", "")
	t.Setenv("TERM", "dumb")
	if keyPromptTTY() {
		t.Error("TERM=dumb must force the plain reader")
	}
}
