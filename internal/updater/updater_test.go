package updater

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestDailyCheckMakesOneRequest(t *testing.T) {
	t.Setenv("ETHERSCAN_GITHUB_TOKEN", "secret-test-token")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("token was forwarded to a non-GitHub host: %q", got)
		}
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.2.0", HTMLURL: "https://example.test/release"})
	}))
	defer server.Close()

	service := NewService()
	service.LatestReleaseURL = server.URL
	service.StatePath = filepath.Join(t.TempDir(), "state.json")
	service.Now = func() time.Time { return time.Date(2026, 7, 22, 8, 0, 0, 0, time.Local) }

	first, err := service.Check(context.Background(), "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Checked || !first.UpdateAvailable || first.Latest != "1.2.0" {
		t.Fatalf("unexpected first result: %+v", first)
	}
	second, err := service.Check(context.Background(), "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if second.Checked || second.UpdateAvailable {
		t.Fatalf("unexpected cached result: %+v", second)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestFailedDailyCheckDoesNotRetryUntilTomorrow(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	service := NewService()
	service.LatestReleaseURL = server.URL
	service.StatePath = filepath.Join(t.TempDir(), "state.json")
	service.Now = func() time.Time { return time.Date(2026, 7, 22, 8, 0, 0, 0, time.Local) }
	if _, err := service.Check(context.Background(), "1.1.0", false); err == nil {
		t.Fatal("expected the first check to fail")
	}
	if _, err := service.Check(context.Background(), "1.1.0", false); err != nil {
		t.Fatalf("second check should use the recorded attempt: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1", requests)
	}
}

func TestAutomaticCheckCanBeDisabled(t *testing.T) {
	t.Setenv("ETHERSCAN_NO_UPDATE_CHECK", "1")
	service := NewService()
	service.StatePath = filepath.Join(t.TempDir(), "state.json")
	service.LatestReleaseURL = "http://127.0.0.1:1/should-not-be-requested"
	result, err := service.Check(context.Background(), "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Checked || result.UpdateAvailable {
		t.Fatalf("disabled check returned %+v", result)
	}
}

func TestSkipSuppressesAutomaticCheckButNotManualCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(githubRelease{TagName: "v1.2.0", HTMLURL: "https://example.test/release"})
	}))
	defer server.Close()

	day := time.Date(2026, 7, 22, 8, 0, 0, 0, time.Local)
	service := NewService()
	service.LatestReleaseURL = server.URL
	service.StatePath = filepath.Join(t.TempDir(), "state.json")
	service.Now = func() time.Time { return day }
	if err := service.Skip("1.2.0"); err != nil {
		t.Fatal(err)
	}
	automatic, err := service.Check(context.Background(), "1.1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	if automatic.UpdateAvailable {
		t.Fatal("skipped version was offered automatically")
	}
	manual, err := service.Check(context.Background(), "1.1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if !manual.UpdateAvailable {
		t.Fatal("manual check should ignore a skipped version")
	}
}
