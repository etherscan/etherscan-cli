package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/etherscan/etherscan-cli/internal/config"
)

const latestReleaseURL = "https://api.github.com/repos/etherscan/etherscan-cli/releases/latest"

type Result struct {
	Current         string
	Latest          string
	ReleaseURL      string
	Checked         bool
	UpdateAvailable bool
}

type state struct {
	LastCheckDate  string `json:"last_check_date,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	ReleaseURL     string `json:"release_url,omitempty"`
	SkippedVersion string `json:"skipped_version,omitempty"`
}

type Service struct {
	HTTPClient       *http.Client
	LatestReleaseURL string
	StatePath        string
	Now              func() time.Time
	Executable       func() (string, error)
	GOOS             string
	LookPath         func(string) (string, error)
	InstallerURL     func(string, string) string
	RemoveFile       func(string) error
	runCommand       commandRunner
}

func NewService() *Service {
	return &Service{
		HTTPClient:       &http.Client{Timeout: 15 * time.Second},
		LatestReleaseURL: latestReleaseURL,
		Now:              time.Now,
		Executable:       os.Executable,
		GOOS:             runtimeGOOS,
		LookPath:         execLookPath,
		InstallerURL:     defaultInstallerURL,
		RemoveFile:       os.Remove,
		runCommand:       defaultCommandRunner,
	}
}

func (s *Service) removeFile(path string) error {
	if s.RemoveFile != nil {
		return s.RemoveFile(path)
	}
	return os.Remove(path)
}

func (s *Service) Check(ctx context.Context, current string, force bool) (Result, error) {
	currentText, currentVersion, err := canonicalVersion(current)
	if err != nil {
		return Result{}, err
	}
	result := Result{Current: currentText}
	if !force && os.Getenv("ETHERSCAN_NO_UPDATE_CHECK") != "" {
		return result, nil
	}
	path, err := s.statePath()
	if err != nil {
		return Result{}, err
	}
	st := loadState(path)
	today := s.now().Format("2006-01-02")
	if !force && st.LastCheckDate == today {
		return result, nil
	}

	// Record the attempt before making the request so a failed network does not
	// slow every interactive launch for the rest of the day.
	st.LastCheckDate = today
	_ = saveState(path, st)

	checkCtx := ctx
	if !force {
		var cancel context.CancelFunc
		checkCtx, cancel = context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
	}
	release, err := s.latestRelease(checkCtx)
	if err != nil {
		return Result{}, err
	}
	latestText, latestVersion, err := canonicalVersion(release.TagName)
	if err != nil {
		return Result{}, fmt.Errorf("GitHub returned an invalid release version %q", release.TagName)
	}
	st.LatestVersion = latestText
	st.ReleaseURL = release.HTMLURL
	_ = saveState(path, st)

	result.Checked = true
	result.Latest = latestText
	result.ReleaseURL = release.HTMLURL
	result.UpdateAvailable = compareVersions(latestVersion, currentVersion) > 0 && (force || st.SkippedVersion != latestText)
	return result, nil
}

func (s *Service) Skip(version string) error {
	version, _, err := canonicalVersion(version)
	if err != nil {
		return err
	}
	path, err := s.statePath()
	if err != nil {
		return err
	}
	st := loadState(path)
	st.SkippedVersion = version
	return saveState(path, st)
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

func (s *Service) latestRelease(ctx context.Context) (githubRelease, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.LatestReleaseURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "etherscan-cli-updater")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token := os.Getenv("ETHERSCAN_GITHUB_TOKEN"); token != "" && isGitHubHost(request.URL.Hostname()) {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := s.client().Do(request)
	if err != nil {
		return githubRelease{}, fmt.Errorf("check GitHub releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return githubRelease{}, fmt.Errorf("check GitHub releases: HTTP %s", response.Status)
	}
	var release githubRelease
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode GitHub release: %w", err)
	}
	if release.TagName == "" || release.Draft || release.Prerelease {
		return githubRelease{}, errors.New("GitHub did not return a stable release")
	}
	if release.HTMLURL == "" {
		release.HTMLURL = "https://github.com/etherscan/etherscan-cli/releases/latest"
	}
	return release, nil
}

func (s *Service) client() *http.Client {
	if s.HTTPClient != nil {
		return s.HTTPClient
	}
	return http.DefaultClient
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) statePath() (string, error) {
	if s.StatePath != "" {
		return s.StatePath, nil
	}
	configPath, err := config.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(configPath), "update-state.json"), nil
}

func loadState(path string) state {
	f, err := os.Open(path)
	if err != nil {
		return state{}
	}
	defer f.Close()
	var st state
	if json.NewDecoder(io.LimitReader(f, 64<<10)).Decode(&st) != nil {
		return state{}
	}
	return st
}

func saveState(path string, st state) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// Write to a sibling temp file and rename over the target so a concurrent CLI
	// launch (e.g. two terminals on the same day) can never observe a torn file.
	f, err := os.CreateTemp(dir, ".update-state-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	err = json.NewEncoder(f).Encode(st)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
