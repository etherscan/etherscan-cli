package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/updater"
)

func TestUninstallWithYesUsesDetectedMethod(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager := &fakeUpdateManager{method: updater.MethodScript}
	root := newRootCommand(BuildInfo{Version: "1.2.0"}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"uninstall", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if manager.uninstallCalls != 1 || manager.uninstalledMethod != updater.MethodScript {
		t.Fatalf("unexpected uninstall: %+v", manager)
	}
	if !strings.Contains(out.String(), "uninstalled") || !strings.Contains(out.String(), "aliases or symlinks") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestUninstallConfirmationDeclinedMakesNoChanges(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager := &fakeUpdateManager{method: updater.MethodHomebrew}
	root := newRootCommand(BuildInfo{}, manager)
	var out bytes.Buffer
	root.SetIn(strings.NewReader("no\n"))
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"uninstall"})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("Execute() error = %v", err)
	}
	if manager.uninstallCalls != 0 {
		t.Fatalf("uninstall ran %d times", manager.uninstallCalls)
	}
	if !strings.Contains(out.String(), "brew uninstall") || !strings.Contains(out.String(), "config") {
		t.Fatalf("prompt did not list targets: %q", out.String())
	}
}

func TestUninstallConfirmationAccepted(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager := &fakeUpdateManager{method: updater.MethodHomebrew}
	root := newRootCommand(BuildInfo{}, manager)
	root.SetIn(strings.NewReader("yes\n"))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"uninstall"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if manager.uninstallCalls != 1 {
		t.Fatalf("uninstall ran %d times", manager.uninstallCalls)
	}
}

func TestUninstallFailureIsNotReportedAsSuccess(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager := &fakeUpdateManager{method: updater.MethodScript, uninstallErr: errors.New("permission denied")}
	root := newRootCommand(BuildInfo{}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"uninstall", "--yes"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected uninstall failure")
	}
	if strings.Contains(out.String(), "CLI uninstalled") {
		t.Fatalf("failure reported success: %q", out.String())
	}
}

func TestUninstallNPMRejectsUntrustedPackageBeforeConfirmation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(updater.NPMInstallPackageEnv, "malicious-package")
	manager := &fakeUpdateManager{method: updater.MethodNPM}
	root := newRootCommand(BuildInfo{}, manager)
	root.SetIn(strings.NewReader("yes\n"))
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"uninstall"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected untrusted package error")
	}
	if manager.uninstallCalls != 0 {
		t.Fatal("uninstall ran for untrusted npm package")
	}
}
