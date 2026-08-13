package updater

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectMethod(t *testing.T) {
	service := NewService()
	service.Executable = func() (string, error) {
		return filepath.Join(string(filepath.Separator), "opt", "homebrew", "Cellar", "etherscan", "1.2.0", "bin", "etherscan"), nil
	}
	if got := service.DetectMethod(); got != MethodHomebrew {
		t.Fatalf("DetectMethod() = %q, want %q", got, MethodHomebrew)
	}
	service.Executable = func() (string, error) { return filepath.Join(t.TempDir(), "etherscan"), nil }
	if got := service.DetectMethod(); got != MethodScript {
		t.Fatalf("DetectMethod() = %q, want %q", got, MethodScript)
	}
	service.Executable = func() (string, error) {
		return filepath.Join(string(filepath.Separator), "usr", "lib", "node_modules", "@etherscan", "cli", "vendor", "etherscan"), nil
	}
	if got := service.DetectMethod(); got != MethodNPM {
		t.Fatalf("DetectMethod() = %q, want %q", got, MethodNPM)
	}
	service.Executable = func() (string, error) {
		return filepath.Join(string(filepath.Separator), "usr", "lib", "node_modules", "@etherscan-npm", "cli", "vendor", "etherscan"), nil
	}
	if got := service.DetectMethod(); got != MethodNPM {
		t.Fatalf("DetectMethod() for transitional scope = %q, want %q", got, MethodNPM)
	}
	// The current layout: the binary lives in a platform package, not the umbrella.
	service.Executable = func() (string, error) {
		return filepath.Join(string(filepath.Separator), "usr", "lib", "node_modules", "@etherscan-npm", "cli-linux-x64", "etherscan"), nil
	}
	if got := service.DetectMethod(); got != MethodNPM {
		t.Fatalf("DetectMethod() for platform package = %q, want %q", got, MethodNPM)
	}
	t.Setenv("ETHERSCAN_INSTALL_METHOD", MethodNPM)
	service.Executable = func() (string, error) { return filepath.Join(t.TempDir(), "etherscan"), nil }
	if got := service.DetectMethod(); got != MethodNPM {
		t.Fatalf("DetectMethod() with npm marker = %q, want %q", got, MethodNPM)
	}
}

func TestNPMUpgradeReturnsPackageManagerInstruction(t *testing.T) {
	service := NewService()
	_, err := service.Upgrade(context.Background(), MethodNPM, "1.2.0", &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "npm install -g @etherscan-npm/cli@latest") {
		t.Fatalf("Upgrade() error = %v, want npm install instruction", err)
	}
}

func TestScriptUpgradeDispatch(t *testing.T) {
	t.Setenv("ETHERSCAN_GITHUB_TOKEN", "secret-test-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("token was forwarded to a non-GitHub host: %q", got)
		}
		io.WriteString(w, "#!/bin/sh\nexit 0\n")
	}))
	defer server.Close()

	var name string
	var args []string
	var background bool
	installDir := filepath.Join(t.TempDir(), "bin")
	service := NewService()
	service.GOOS = "linux"
	service.Executable = func() (string, error) { return filepath.Join(installDir, "etherscan"), nil }
	service.InstallerURL = func(string, string) string { return server.URL }
	service.runCommand = func(_ context.Context, command string, commandArgs []string, _, _ io.Writer, bg bool) error {
		name, args, background = command, append([]string(nil), commandArgs...), bg
		return nil
	}
	if _, err := service.Upgrade(context.Background(), MethodScript, "1.2.0", &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if name != "sh" || background || len(args) != 6 {
		t.Fatalf("unexpected dispatch: name=%q args=%q background=%v", name, args, background)
	}
	if args[1] != "--version" || args[2] != "v1.2.0" || args[3] != "--install-dir" || args[4] != installDir || args[5] != "--no-path-update" {
		t.Fatalf("unexpected installer arguments: %q", args)
	}
}

func TestWindowsScriptUpgradeRunsAfterCurrentProcessExits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Write-Host update\n")
	}))
	defer server.Close()

	installDir := filepath.Join(t.TempDir(), "bin")
	var args []string
	var background bool
	service := NewService()
	service.GOOS = "windows"
	service.Executable = func() (string, error) { return filepath.Join(installDir, "etherscan.exe"), nil }
	service.InstallerURL = func(string, string) string { return server.URL }
	service.runCommand = func(_ context.Context, command string, commandArgs []string, _, _ io.Writer, bg bool) error {
		if command != "powershell.exe" {
			t.Fatalf("command = %q, want powershell.exe", command)
		}
		args, background = append([]string(nil), commandArgs...), bg
		return nil
	}
	backgroundResult, err := service.Upgrade(context.Background(), MethodScript, "1.2.0", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !background || !backgroundResult || !strings.Contains(joined, "-WaitForProcessId") || !strings.Contains(joined, "-CleanupScript") || !strings.Contains(joined, installDir) {
		t.Fatalf("unexpected Windows dispatch: args=%q background=%v result=%v", args, background, backgroundResult)
	}
}

func TestHomebrewUpgradeDispatch(t *testing.T) {
	var name string
	var args []string
	service := NewService()
	service.LookPath = func(file string) (string, error) { return "/opt/homebrew/bin/" + file, nil }
	service.runCommand = func(_ context.Context, command string, commandArgs []string, _, _ io.Writer, background bool) error {
		name, args = command, append([]string(nil), commandArgs...)
		if background {
			t.Fatal("Homebrew update must run in the foreground")
		}
		return nil
	}
	if _, err := service.Upgrade(context.Background(), MethodHomebrew, "1.2.0", &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if name != "brew" || strings.Join(args, " ") != "upgrade etherscan/etherscan-cli/etherscan" {
		t.Fatalf("unexpected command: %s %q", name, args)
	}
}
