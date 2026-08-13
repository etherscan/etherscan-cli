package updater

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNPMPackageNameAllowlist(t *testing.T) {
	for _, name := range []string{NPMTransitionalPackage, NPMCanonicalPackage} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(NPMInstallPackageEnv, name)
			got, err := NPMPackageName()
			if err != nil || got != name {
				t.Fatalf("NPMPackageName() = %q, %v", got, err)
			}
		})
	}
	t.Setenv(NPMInstallPackageEnv, "some-other-package")
	if _, err := NPMPackageName(); err == nil {
		t.Fatal("untrusted package name was accepted")
	}
}

func TestNPMPackageNameWithoutMarkerFailsClosed(t *testing.T) {
	t.Setenv(NPMInstallPackageEnv, "")
	if _, err := NPMPackageName(); err == nil {
		t.Fatal("package name was guessed for a non-npm test executable")
	}
}

func TestNPMPackageFromExecutable(t *testing.T) {
	t.Setenv("ETHERSCAN_INSTALL_METHOD", "")
	tests := []struct {
		name       string
		executable string
		want       string
		ok         bool
	}{
		{
			name:       "canonical package",
			executable: filepath.Join("C:", "Users", "test", "node_modules", "@etherscan", "cli", "vendor", "etherscan.exe"),
			want:       NPMCanonicalPackage,
			ok:         true,
		},
		{
			name:       "transitional package",
			executable: filepath.Join("usr", "local", "lib", "node_modules", "@etherscan-npm", "cli", "vendor", "etherscan"),
			want:       NPMTransitionalPackage,
			ok:         true,
		},
		{
			name:       "lookalike package",
			executable: filepath.Join("usr", "local", "lib", "node_modules", "@etherscan", "cli-malicious", "vendor", "etherscan"),
		},
		{
			name:       "manual install",
			executable: filepath.Join("usr", "local", "bin", "etherscan"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := npmPackageFromExecutable(test.executable)
			if got != test.want || ok != test.ok {
				t.Fatalf("npmPackageFromExecutable(%q) = %q, %v; want %q, %v", test.executable, got, ok, test.want, test.ok)
			}

			service := NewService()
			service.Executable = func() (string, error) { return test.executable, nil }
			if detected := service.DetectMethod(); (detected == MethodNPM) != test.ok {
				t.Fatalf("DetectMethod() = %q; package classification success = %v", detected, test.ok)
			}
		})
	}
}

func TestHomebrewUninstallRemovesConfigAfterSuccess(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	configDir := filepath.Join(configRoot, "etherscan")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	var command string
	var args []string
	service := NewService()
	service.LookPath = func(name string) (string, error) { return "/opt/homebrew/bin/" + name, nil }
	service.runCommand = func(_ context.Context, name string, commandArgs []string, _, _ io.Writer, background bool) error {
		command, args = name, append([]string(nil), commandArgs...)
		if background {
			t.Fatal("Homebrew uninstall ran in background")
		}
		return nil
	}
	if background, err := service.Uninstall(context.Background(), MethodHomebrew, io.Discard, io.Discard); err != nil || background {
		t.Fatalf("Uninstall() = %v, %v", background, err)
	}
	if command != "brew" || strings.Join(args, " ") != "uninstall etherscan/etherscan-cli/etherscan" {
		t.Fatalf("unexpected command: %s %q", command, args)
	}
	if _, err := os.Stat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config still exists: %v", err)
	}
}

func TestManagerFailureRetainsConfig(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	configDir := filepath.Join(configRoot, "etherscan")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	service := NewService()
	service.LookPath = func(name string) (string, error) { return name, nil }
	service.runCommand = func(context.Context, string, []string, io.Writer, io.Writer, bool) error { return errors.New("failed") }
	if _, err := service.Uninstall(context.Background(), MethodHomebrew, io.Discard, io.Discard); err == nil {
		t.Fatal("expected manager failure")
	}
	if _, err := os.Stat(configDir); err != nil {
		t.Fatalf("config was removed after manager failure: %v", err)
	}
}

func TestNPMUninstallUsesValidatedPackageManager(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(NPMInstallPackageEnv, "@etherscan/cli")
	service := NewService()
	service.GOOS = "linux"
	service.LookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	var command string
	var args []string
	service.runCommand = func(_ context.Context, name string, commandArgs []string, _, _ io.Writer, background bool) error {
		command, args = name, append([]string(nil), commandArgs...)
		if background {
			t.Fatal("Unix npm uninstall ran in background")
		}
		return nil
	}
	if _, err := service.Uninstall(context.Background(), MethodNPM, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if command != "/usr/bin/npm" || strings.Join(args, " ") != "uninstall -g @etherscan/cli" {
		t.Fatalf("unexpected npm command: %s %q", command, args)
	}
}

func TestUnmanagedUninstallRemovesExactExecutableOnly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	installDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	running := filepath.Join(installDir, "renamed")
	sibling := filepath.Join(installDir, "etherscan")
	if err := os.WriteFile(running, []byte("running"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("unrelated"), 0o700); err != nil {
		t.Fatal(err)
	}
	service := NewService()
	service.GOOS = "linux"
	service.Executable = func() (string, error) { return running, nil }
	if _, err := service.Uninstall(context.Background(), MethodScript, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(running); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("running executable remains: %v", err)
	}
	if data, err := os.ReadFile(sibling); err != nil || string(data) != "unrelated" {
		t.Fatalf("sibling changed: %q, %v", data, err)
	}
	if _, err := os.Stat(installDir); err != nil {
		t.Fatalf("unmanaged parent directory was removed: %v", err)
	}
}

func TestProtectedUnmanagedExecutableRetainsConfig(t *testing.T) {
	root := t.TempDir()
	configRoot := filepath.Join(root, "config")
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	configDir := filepath.Join(configRoot, "etherscan")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	service := NewService()
	service.GOOS = "linux"
	service.Executable = func() (string, error) { return "/usr/local/bin/etherscan", nil }
	service.RemoveFile = func(string) error { return os.ErrPermission }
	_, err := service.Uninstall(context.Background(), MethodScript, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "sudo rm -- ") || !strings.Contains(err.Error(), "etherscan") {
		t.Fatalf("error lacks safe manual instruction: %v", err)
	}
	if _, err := os.Stat(configDir); err != nil {
		t.Fatalf("config was removed: %v", err)
	}
}

func TestWindowsUninstallSchedulesLocalHelper(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	service := NewService()
	service.GOOS = "windows"
	service.Executable = func() (string, error) { return filepath.Join(t.TempDir(), "renamed.exe"), nil }
	var args []string
	service.runCommand = func(_ context.Context, name string, commandArgs []string, _, _ io.Writer, background bool) error {
		if name != "powershell.exe" || !background {
			t.Fatalf("unexpected dispatch: %s background=%v", name, background)
		}
		args = append([]string(nil), commandArgs...)
		return nil
	}
	background, err := service.Uninstall(context.Background(), MethodScript, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || !background {
		t.Fatalf("Uninstall() = %v, %v", background, err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-WaitForProcessId", "-Action binary", "renamed.exe", "-CleanupScript"} {
		if !strings.Contains(joined, want) {
			t.Errorf("arguments missing %q: %q", want, args)
		}
	}
	for i, arg := range args {
		if arg == "-File" && i+1 < len(args) {
			os.Remove(args[i+1])
		}
	}
}

func TestWindowsHelperLaunchFailureCleansScript(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	service := NewService()
	service.GOOS = "windows"
	service.Executable = func() (string, error) { return filepath.Join(t.TempDir(), "etherscan.exe"), nil }
	var scriptPath string
	service.runCommand = func(_ context.Context, _ string, args []string, _, _ io.Writer, _ bool) error {
		for i, arg := range args {
			if arg == "-File" {
				scriptPath = args[i+1]
			}
		}
		return errors.New("launch failed")
	}
	if _, err := service.Uninstall(context.Background(), MethodScript, io.Discard, io.Discard); err == nil {
		t.Fatal("expected launch failure")
	}
	if _, err := os.Stat(scriptPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("helper remains: %v", err)
	}
}

func TestWindowsNPMUninstallSchedulesManager(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(NPMInstallPackageEnv, "@etherscan-npm/cli")
	t.Setenv(NPMWrapperPIDEnv, "4242")
	service := NewService()
	service.GOOS = "windows"
	service.LookPath = func(name string) (string, error) { return `C:\Program Files\nodejs\npm.cmd`, nil }
	var args []string
	service.runCommand = func(_ context.Context, _ string, commandArgs []string, _, _ io.Writer, background bool) error {
		if !background {
			t.Fatal("Windows npm uninstall must run after process exit")
		}
		args = append([]string(nil), commandArgs...)
		return nil
	}
	background, err := service.Uninstall(context.Background(), MethodNPM, io.Discard, io.Discard)
	if err != nil || !background {
		t.Fatalf("Uninstall() = %v, %v", background, err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-Action npm", "-WaitForParentProcessId", `C:\Program Files\nodejs\npm.cmd`, "@etherscan-npm/cli"} {
		if !strings.Contains(joined, want) {
			t.Errorf("arguments missing %q: %q", want, args)
		}
	}
	for i, arg := range args {
		if arg == "-File" {
			os.Remove(args[i+1])
		}
	}
}

func TestWindowsNPMUninstallRejectsInvalidWrapperPID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(NPMInstallPackageEnv, "@etherscan/cli")
	t.Setenv(NPMWrapperPIDEnv, "not-a-pid")
	service := NewService()
	service.GOOS = "windows"
	service.LookPath = func(name string) (string, error) { return `C:\Program Files\nodejs\npm.cmd`, nil }
	service.runCommand = func(context.Context, string, []string, io.Writer, io.Writer, bool) error {
		t.Fatal("helper ran with an invalid wrapper PID")
		return nil
	}
	if _, err := service.Uninstall(context.Background(), MethodNPM, io.Discard, io.Discard); err == nil {
		t.Fatal("invalid wrapper PID was accepted")
	}
}

func TestWindowsHelperRemovesExactBinaryAndConfig(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell helper is Windows-specific")
	}
	root := t.TempDir()
	installDir := filepath.Join(root, "manual")
	configDir := filepath.Join(root, "config", "etherscan")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(installDir, "renamed.exe")
	sibling := filepath.Join(installDir, "etherscan.exe")
	if err := os.WriteFile(executable, []byte("remove"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sibling, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "helper.ps1")
	if err := os.WriteFile(script, []byte(windowsUninstallScript), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", script,
		"-WaitForProcessId", "2147483647", "-Action", "binary", "-Executable", executable, "-ConfigDir", configDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("helper failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(executable); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("exact executable remains: %v", err)
	}
	if _, err := os.Stat(configDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("config remains: %v", err)
	}
	if data, err := os.ReadFile(sibling); err != nil || string(data) != "keep" {
		t.Fatalf("sibling changed: %q, %v", data, err)
	}
}

func TestProfileCleanupRequiresAdjacentMarkerAndPreservesMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(installDir, installMarkerName)
	if err := os.WriteFile(marker, []byte(installMarkerContent), 0o600); err != nil {
		t.Fatal(err)
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(installDir)
	target := `export PATH="` + escaped + `:$PATH"`
	profile := filepath.Join(home, ".profile")
	content := "export EDITOR=vim\n# Etherscan CLI\n" + target + "\n" + target + "\n"
	if err := os.WriteFile(profile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupUnixInstallerPath(installDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), target) != 1 || !strings.Contains(string(data), "export EDITOR=vim") {
		t.Fatalf("profile changed incorrectly: %q", data)
	}
	info, err := os.Stat(profile)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("profile mode = %v", info.Mode().Perm())
	}
}

func TestUnixLegacyProfileBlockProvidesProvenance(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installDir := filepath.Join(home, "legacy-bin")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(installDir)
	target := `export PATH="` + escaped + `:$PATH"`
	profile := filepath.Join(home, ".profile")
	if err := os.WriteFile(profile, []byte("# Etherscan CLI\n"+target+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cleanupUnixInstallerPath(installDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy-owned directory remains: %v", err)
	}
	data, err := os.ReadFile(profile)
	if err != nil || strings.Contains(string(data), target) {
		t.Fatalf("legacy PATH block remains: %q, %v", data, err)
	}
}

func TestUnprovenancedEmptyDirectoryIsRetained(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	installDir := filepath.Join(home, "manual-bin")
	if err := os.MkdirAll(installDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := cleanupUnixInstallerPath(installDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(installDir); err != nil {
		t.Fatalf("unprovenanced directory was removed: %v", err)
	}
}
