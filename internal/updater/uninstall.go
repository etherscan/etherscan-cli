package updater

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/etherscan/etherscan-cli/internal/config"
)

const (
	installMarkerName    = ".etherscan-cli-path-added"
	installMarkerContent = "etherscan-cli:path-added:v1"
)

// Uninstall removes the package-managed installation or the exact running executable.
// It returns true when Windows has scheduled the work for after the current process exits.
func (s *Service) Uninstall(ctx context.Context, method string, stdout, stderr io.Writer) (background bool, err error) {
	if method == "" {
		method = s.DetectMethod()
	}
	if !ValidMethod(method) {
		return false, fmt.Errorf("unsupported uninstall method %q", method)
	}

	configDir, err := uninstallConfigDir()
	if err != nil {
		return false, err
	}
	goos := s.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}

	switch method {
	case MethodHomebrew:
		if _, err := s.lookPath()("brew"); err != nil {
			return false, errorsWithHint(err, "Homebrew was detected but brew is not on PATH")
		}
		if err := s.runner()(ctx, "brew", []string{"uninstall", "etherscan/etherscan-cli/etherscan"}, stdout, stderr, false); err != nil {
			return false, err
		}
		return false, removeConfigDir(configDir)

	case MethodNPM:
		packageName, err := NPMPackageName()
		if err != nil {
			return false, err
		}
		npmPath, err := s.lookPath()("npm")
		if err != nil {
			return false, errorsWithHint(err, "npm installation was detected but npm is not on PATH")
		}
		if goos == "windows" {
			wrapperPID := 0
			if value := strings.TrimSpace(os.Getenv(NPMWrapperPIDEnv)); value != "" {
				parsed, parseErr := strconv.Atoi(value)
				if parseErr != nil || parsed <= 0 {
					return false, fmt.Errorf("invalid npm wrapper process ID %q", value)
				}
				wrapperPID = parsed
			}
			return s.scheduleWindowsUninstall(ctx, windowsUninstallOptions{
				Action:     "npm",
				NPMPath:    npmPath,
				NPMPackage: packageName,
				ParentPID:  wrapperPID,
				ConfigDir:  configDir,
				Stdout:     stdout,
				Stderr:     stderr,
			})
		}
		if err := s.runner()(ctx, npmPath, []string{"uninstall", "-g", packageName}, stdout, stderr, false); err != nil {
			return false, err
		}
		return false, removeConfigDir(configDir)
	}

	executable, err := s.executable()
	if err != nil {
		return false, fmt.Errorf("locate current executable: %w", err)
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return false, fmt.Errorf("resolve current executable: %w", err)
	}
	if strings.ContainsAny(executable, "\r\n") {
		return false, fmt.Errorf("executable path contains a line break")
	}

	if goos == "windows" {
		return s.scheduleWindowsUninstall(ctx, windowsUninstallOptions{
			Action:     "binary",
			Executable: executable,
			ConfigDir:  configDir,
			Stdout:     stdout,
			Stderr:     stderr,
		})
	}
	if goos != "darwin" && goos != "linux" {
		return false, fmt.Errorf("uninstall is not supported on %s", goos)
	}

	if err := s.removeFile(executable); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("remove %s: %w; remove it manually with: %s", executable, err, manualRemoveCommand(goos, executable, errors.Is(err, os.ErrPermission)))
	}
	if err := cleanupUnixInstallerPath(filepath.Dir(executable)); err != nil {
		return false, fmt.Errorf("binary removed, but PATH cleanup failed: %w", err)
	}
	if err := removeConfigDir(configDir); err != nil {
		return false, fmt.Errorf("binary removed, but configuration cleanup failed: %w", err)
	}
	return false, nil
}

func uninstallConfigDir() (string, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return "", fmt.Errorf("resolve configuration directory: %w", err)
	}
	return filepath.Dir(path), nil
}

func removeConfigDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove configuration %s: %w", dir, err)
	}
	return nil
}

func manualRemoveCommand(goos, path string, privileged bool) string {
	if goos == "windows" {
		return "Remove-Item -LiteralPath " + powershellQuote(path) + " -Force"
	}
	if privileged {
		return "sudo rm -- " + shellQuote(path)
	}
	return "rm -- " + shellQuote(path)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func powershellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func cleanupUnixInstallerPath(installDir string) error {
	marker := filepath.Join(installDir, installMarkerName)
	markerValid := validInstallMarker(marker)
	profiles := unixProfiles()
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`").Replace(installDir)
	posixLine := `export PATH="` + escaped + `:$PATH"`
	fishLine := `fish_add_path "` + escaped + `"`
	legacy := false
	for _, item := range profiles {
		if hasExactProfileBlock(item.path, item.line(posixLine, fishLine)) {
			legacy = true
			break
		}
	}
	if !markerValid && !legacy {
		return nil
	}
	empty, err := directoryEmptyExceptMarker(installDir, marker, markerValid)
	if err != nil || !empty {
		return err
	}
	for _, item := range profiles {
		if err := removeExactProfileBlock(item.path, item.line(posixLine, fishLine)); err != nil {
			return err
		}
	}
	if markerValid {
		if err := os.Remove(marker); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.Remove(installDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

type unixProfile struct {
	path string
	fish bool
}

func (p unixProfile) line(posix, fish string) string {
	if p.fish {
		return fish
	}
	return posix
}

func unixProfiles() []unixProfile {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil
		}
	}
	return []unixProfile{
		{path: filepath.Join(home, ".zshrc")},
		{path: filepath.Join(home, ".bashrc")},
		{path: filepath.Join(home, ".profile")},
		{path: filepath.Join(home, ".config", "fish", "config.fish"), fish: true},
	}
}

func validInstallMarker(path string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	data, err := os.ReadFile(path)
	return err == nil && string(data) == installMarkerContent
}

func directoryEmptyExceptMarker(dir, marker string, ignoreMarker bool) (bool, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if ignoreMarker && filepath.Join(dir, entry.Name()) == marker {
			continue
		}
		return false, nil
	}
	return true, nil
}

func hasExactProfileBlock(path, target string) bool {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	for i := 0; i+1 < len(lines); i++ {
		if lines[i] == "# Etherscan CLI" && lines[i+1] == target {
			return true
		}
	}
	return false
}

func removeExactProfileBlock(path, target string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) || (err == nil && !info.Mode().IsRegular()) {
		return nil
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	newline := "\n"
	if bytes.Contains(data, []byte("\r\n")) {
		newline = "\r\n"
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	changed := false
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if i+1 < len(lines) && lines[i] == "# Etherscan CLI" && lines[i+1] == target {
			i++
			changed = true
			continue
		}
		out = append(out, lines[i])
	}
	if !changed {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".etherscan-profile-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return err
	}
	if _, err := io.WriteString(tmp, strings.Join(out, newline)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
