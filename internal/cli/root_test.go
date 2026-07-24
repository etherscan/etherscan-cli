package cli

import (
	"bytes"
	"strings"
	"testing"
)

// A bare `etherscan` invocation prints the Quick Start guide (and does not launch
// the explorer or run an update check).
func TestBareInvocationPrintsQuickStart(t *testing.T) {
	manager := &fakeUpdateManager{}
	root := newRootCommand(BuildInfo{Version: "1.1.0"}, manager)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	got := out.String()
	for _, want := range []string{"Quick Start", "etherscan login", "docs.etherscan.io"} {
		if !strings.Contains(got, want) {
			t.Errorf("bare invocation output missing %q; got:\n%s", want, got)
		}
	}
	// A bare invocation must not trigger the update flow.
	if manager.upgradedVersion != "" || manager.skipped != "" {
		t.Errorf("bare invocation should not touch the updater, got %+v", manager)
	}
}
