package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/etherscan/etherscan-cli/internal/updater"
)

type fakeUpdateManager struct {
	result            updater.Result
	checkErr          error
	method            string
	skipped           string
	upgradedMethod    string
	upgradedVersion   string
	background        bool
	uninstalledMethod string
	uninstallCalls    int
	uninstallErr      error
}

func (f *fakeUpdateManager) Check(context.Context, string, bool) (updater.Result, error) {
	return f.result, f.checkErr
}

func (f *fakeUpdateManager) Skip(version string) error {
	f.skipped = version
	return nil
}

func (f *fakeUpdateManager) DetectMethod() string { return f.method }

func (f *fakeUpdateManager) Upgrade(_ context.Context, method, version string, _, _ io.Writer) (bool, error) {
	f.upgradedMethod = method
	f.upgradedVersion = version
	return f.background, nil
}

func (f *fakeUpdateManager) Uninstall(_ context.Context, method string, _, _ io.Writer) (bool, error) {
	f.uninstallCalls++
	f.uninstalledMethod = method
	return f.background, f.uninstallErr
}

func TestOfferUpdateChoices(t *testing.T) {
	result := updater.Result{
		Current:         "1.1.0",
		Latest:          "1.2.0",
		ReleaseURL:      "https://example.test/release",
		Checked:         true,
		UpdateAvailable: true,
	}

	t.Run("later", func(t *testing.T) {
		manager := &fakeUpdateManager{result: result, method: updater.MethodScript}
		var out bytes.Buffer
		exit, err := offerUpdate(context.Background(), manager, "1.1.0", strings.NewReader("2\n"), &out, &bytes.Buffer{})
		if err != nil || exit || manager.skipped != "" || manager.upgradedVersion != "" {
			t.Fatalf("unexpected result: exit=%v err=%v manager=%+v", exit, err, manager)
		}
	})

	t.Run("skip", func(t *testing.T) {
		manager := &fakeUpdateManager{result: result, method: updater.MethodScript}
		exit, err := offerUpdate(context.Background(), manager, "1.1.0", strings.NewReader("3\n"), &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil || exit || manager.skipped != "1.2.0" {
			t.Fatalf("unexpected result: exit=%v err=%v manager=%+v", exit, err, manager)
		}
	})

	t.Run("enter defaults to later", func(t *testing.T) {
		manager := &fakeUpdateManager{result: result, method: updater.MethodScript}
		exit, err := offerUpdate(context.Background(), manager, "1.1.0", strings.NewReader("\n"), &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil || exit || manager.skipped != "" || manager.upgradedVersion != "" {
			t.Fatalf("empty input should default to Later: exit=%v err=%v manager=%+v", exit, err, manager)
		}
	})

	t.Run("update", func(t *testing.T) {
		manager := &fakeUpdateManager{result: result, method: updater.MethodHomebrew}
		exit, err := offerUpdate(context.Background(), manager, "1.1.0", strings.NewReader("1\n"), &bytes.Buffer{}, &bytes.Buffer{})
		if err != nil || !exit || manager.upgradedMethod != updater.MethodHomebrew || manager.upgradedVersion != "1.2.0" {
			t.Fatalf("unexpected result: exit=%v err=%v manager=%+v", exit, err, manager)
		}
	})

	t.Run("npm update instruction", func(t *testing.T) {
		manager := &fakeUpdateManager{result: result, method: updater.MethodNPM}
		var out bytes.Buffer
		exit, err := offerUpdate(context.Background(), manager, "1.1.0", strings.NewReader("1\n"), &out, &bytes.Buffer{})
		if err != nil || !exit || manager.upgradedVersion != "" {
			t.Fatalf("unexpected result: exit=%v err=%v manager=%+v", exit, err, manager)
		}
		if !strings.Contains(out.String(), "npm install -g @etherscan/cli@latest") {
			t.Fatalf("output = %q, want npm install instruction", out.String())
		}
	})
}

func TestUpdateCommandUsesRequestedMethod(t *testing.T) {
	manager := &fakeUpdateManager{
		result: updater.Result{Current: "1.1.0", Latest: "1.2.0", Checked: true, UpdateAvailable: true},
		method: updater.MethodScript,
	}
	root := newRootCommand(BuildInfo{Version: "1.1.0"}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"update", "--method", "homebrew"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if manager.upgradedMethod != updater.MethodHomebrew || manager.upgradedVersion != "1.2.0" {
		t.Fatalf("unexpected update: %+v", manager)
	}
}

func TestUpdateCommandShowsNPMInstruction(t *testing.T) {
	manager := &fakeUpdateManager{
		result: updater.Result{Current: "1.1.0", Latest: "1.2.0", Checked: true, UpdateAvailable: true},
		method: updater.MethodNPM,
	}
	root := newRootCommand(BuildInfo{Version: "1.1.0"}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"update"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if manager.upgradedVersion != "" {
		t.Fatalf("npm update invoked updater: %+v", manager)
	}
	if !strings.Contains(out.String(), "npm install -g @etherscan/cli@latest") {
		t.Fatalf("output = %q, want npm install instruction", out.String())
	}
}

func TestUpdateCommandCannotForceScriptForNPMInstallation(t *testing.T) {
	manager := &fakeUpdateManager{
		result: updater.Result{Current: "1.1.0", Latest: "1.2.0", Checked: true, UpdateAvailable: true},
		method: updater.MethodNPM,
	}
	root := newRootCommand(BuildInfo{Version: "1.1.0"}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"update", "--method", "script"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if manager.upgradedVersion != "" {
		t.Fatalf("npm update invoked forced script updater: %+v", manager)
	}
	if !strings.Contains(out.String(), "npm install -g @etherscan/cli@latest") {
		t.Fatalf("output = %q, want npm install instruction", out.String())
	}
}
