package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	MethodHomebrew = "homebrew"
	MethodNPM      = "npm"
	MethodScript   = "script"
)

var runtimeGOOS = runtime.GOOS

type commandRunner func(context.Context, string, []string, io.Writer, io.Writer, bool) error

func (s *Service) DetectMethod() string {
	if strings.EqualFold(os.Getenv("ETHERSCAN_INSTALL_METHOD"), MethodNPM) {
		return MethodNPM
	}
	executable, err := s.executable()
	if err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
			executable = resolved
		}
		normalized := strings.ToLower(filepath.ToSlash(executable))
		if strings.Contains(normalized, "/node_modules/@etherscan/cli/") {
			return MethodNPM
		}
		if strings.Contains(normalized, "/cellar/etherscan/") || strings.Contains(normalized, "/linuxbrew/.linuxbrew/cellar/etherscan/") {
			return MethodHomebrew
		}
	}
	return MethodScript
}

func ValidMethod(method string) bool {
	return method == MethodHomebrew || method == MethodNPM || method == MethodScript
}

// Upgrade installs a stable release. The returned background value is true on
// Windows, where the installer waits for this running executable to exit before
// replacing it.
func (s *Service) Upgrade(ctx context.Context, method, version string, stdout, stderr io.Writer) (background bool, err error) {
	version, _, err = canonicalVersion(version)
	if err != nil {
		return false, err
	}
	if method == "" {
		method = s.DetectMethod()
	}
	if !ValidMethod(method) {
		return false, fmt.Errorf("unsupported update method %q (use homebrew, npm, or script)", method)
	}
	if method == MethodNPM {
		return false, fmt.Errorf("npm manages this installation; run npm install -g @etherscan/cli@latest")
	}
	if method == MethodHomebrew {
		if _, err := s.lookPath()("brew"); err != nil {
			return false, errorsWithHint(err, "Homebrew was detected but brew is not on PATH")
		}
		return false, s.runner()(ctx, "brew", []string{"upgrade", "etherscan/etherscan-cli/etherscan"}, stdout, stderr, false)
	}

	executable, err := s.executable()
	if err != nil {
		return false, fmt.Errorf("locate current executable: %w", err)
	}
	installDir := filepath.Dir(executable)
	if strings.ContainsAny(installDir, "\r\n") {
		return false, fmt.Errorf("installation directory contains a line break")
	}
	goos := s.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	if goos != "windows" && goos != "darwin" && goos != "linux" {
		return false, fmt.Errorf("script updates are not supported on %s", goos)
	}

	extension := ".sh"
	if goos == "windows" {
		extension = ".ps1"
	}
	installerURL := s.installerURL()(goos, version)
	installerPath, err := s.downloadInstaller(ctx, installerURL, extension)
	if err != nil {
		return false, err
	}

	if goos == "windows" {
		args := []string{
			"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
			"-File", installerPath,
			"-Version", "v" + version,
			"-InstallDir", installDir,
			"-NoPathUpdate",
			"-WaitForProcessId", strconv.Itoa(os.Getpid()),
			"-CleanupScript",
		}
		if err := s.runner()(ctx, "powershell.exe", args, stdout, stderr, true); err != nil {
			_ = os.Remove(installerPath)
			return false, err
		}
		return true, nil
	}

	defer os.Remove(installerPath)
	if err := os.Chmod(installerPath, 0o700); err != nil {
		return false, err
	}
	args := []string{installerPath, "--version", "v" + version, "--install-dir", installDir, "--no-path-update"}
	return false, s.runner()(ctx, "sh", args, stdout, stderr, false)
}

func (s *Service) executable() (string, error) {
	if s.Executable != nil {
		return s.Executable()
	}
	return os.Executable()
}

func (s *Service) runner() commandRunner {
	if s.runCommand != nil {
		return s.runCommand
	}
	return defaultCommandRunner
}

func (s *Service) lookPath() func(string) (string, error) {
	if s.LookPath != nil {
		return s.LookPath
	}
	return exec.LookPath
}

func (s *Service) installerURL() func(string, string) string {
	if s.InstallerURL != nil {
		return s.InstallerURL
	}
	return defaultInstallerURL
}

func defaultInstallerURL(goos, version string) string {
	extension := ".sh"
	if goos == "windows" {
		extension = ".ps1"
	}
	return "https://raw.githubusercontent.com/etherscan/etherscan-cli/v" + version + "/scripts/install" + extension
}

var execLookPath = exec.LookPath

func (s *Service) downloadInstaller(ctx context.Context, installerURL, extension string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, installerURL, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "etherscan-cli-updater")
	if token := os.Getenv("ETHERSCAN_GITHUB_TOKEN"); token != "" && isGitHubHost(request.URL.Hostname()) {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := s.client().Do(request)
	if err != nil {
		return "", fmt.Errorf("download installer: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download installer: HTTP %s", response.Status)
	}
	contents, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil {
		return "", fmt.Errorf("download installer: %w", err)
	}
	if len(contents) == 0 || len(contents) > 1<<20 {
		return "", fmt.Errorf("download installer: invalid script size")
	}
	f, err := os.CreateTemp("", "etherscan-update-*"+extension)
	if err != nil {
		return "", err
	}
	path := f.Name()
	if _, err := f.Write(contents); err != nil {
		f.Close()
		os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", err
	}
	return path, nil
}

func defaultCommandRunner(ctx context.Context, name string, args []string, stdout, stderr io.Writer, background bool) error {
	var cmd *exec.Cmd
	if background {
		cmd = exec.Command(name, args...)
		configureBackgroundCommand(cmd)
	} else {
		cmd = exec.CommandContext(ctx, name, args...)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if !background {
		return cmd.Run()
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func errorsWithHint(err error, hint string) error {
	return fmt.Errorf("%s: %w", hint, err)
}

func isGitHubHost(host string) bool {
	switch strings.ToLower(host) {
	case "github.com", "api.github.com", "raw.githubusercontent.com":
		return true
	default:
		return false
	}
}
