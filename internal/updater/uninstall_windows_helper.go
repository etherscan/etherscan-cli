package updater

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
)

type windowsUninstallOptions struct {
	Action     string
	Executable string
	NPMPath    string
	NPMPackage string
	ParentPID  int
	ConfigDir  string
	Stdout     io.Writer
	Stderr     io.Writer
}

func (s *Service) scheduleWindowsUninstall(ctx context.Context, options windowsUninstallOptions) (bool, error) {
	script, err := os.CreateTemp("", "etherscan-uninstall-*.ps1")
	if err != nil {
		return false, err
	}
	path := script.Name()
	if _, err := io.WriteString(script, windowsUninstallScript); err != nil {
		script.Close()
		os.Remove(path)
		return false, err
	}
	if err := script.Close(); err != nil {
		os.Remove(path)
		return false, err
	}

	args := []string{
		"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-File", path,
		"-WaitForProcessId", strconv.Itoa(os.Getpid()),
		"-Action", options.Action,
		"-ConfigDir", options.ConfigDir,
		"-CleanupScript",
	}
	if options.Action == "npm" {
		if options.ParentPID > 0 {
			args = append(args, "-WaitForParentProcessId", strconv.Itoa(options.ParentPID))
		}
		args = append(args,
			"-NpmPath", options.NPMPath,
			"-NpmPackage", options.NPMPackage,
		)
	} else {
		args = append(args, "-Executable", options.Executable)
	}
	if err := s.runner()(ctx, "powershell.exe", args, options.Stdout, options.Stderr, true); err != nil {
		os.Remove(path)
		return false, err
	}
	return true, nil
}

var windowsUninstallScript = fmt.Sprintf(`[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][int]$WaitForProcessId,
	[int]$WaitForParentProcessId = 0,
    [Parameter(Mandatory=$true)][ValidateSet('binary','npm')][string]$Action,
    [string]$Executable,
    [string]$NpmPath,
    [string]$NpmPackage,
    [Parameter(Mandatory=$true)][string]$ConfigDir,
    [switch]$CleanupScript
)
$ErrorActionPreference = 'Stop'
$markerName = '%s'
$markerContent = "%s"

function Test-Marker([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    $item = Get-Item -LiteralPath $Path -Force
    if ($item.Attributes -band [IO.FileAttributes]::ReparsePoint) { return $false }
    return (Get-Content -LiteralPath $Path -Raw) -eq $markerContent
}

function Remove-UserPath([string]$Directory) {
    $full = [IO.Path]::GetFullPath($Directory).TrimEnd('\')
    $entries = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $kept = @($entries | Where-Object {
        $entry = $_
        try {
            $expanded = [Environment]::ExpandEnvironmentVariables($entry)
            -not [IO.Path]::GetFullPath($expanded).TrimEnd('\').Equals($full, [StringComparison]::OrdinalIgnoreCase)
        } catch {
            -not $entry.TrimEnd('\').Equals($full, [StringComparison]::OrdinalIgnoreCase)
        }
    })
    if ($kept.Count -ne $entries.Count) {
        [Environment]::SetEnvironmentVariable('Path', ($kept -join ';'), 'User')
        Write-Host "Removed $full from your user PATH."
    }
}

try {
    Wait-Process -Id $WaitForProcessId -ErrorAction SilentlyContinue
    if ($Action -eq 'npm') {
		if ($WaitForParentProcessId -gt 0) {
			Wait-Process -Id $WaitForParentProcessId -ErrorAction SilentlyContinue
		}
        & $NpmPath uninstall -g $NpmPackage
        if ($LASTEXITCODE -ne 0) { throw "npm uninstall failed with exit code $LASTEXITCODE" }
    } else {
        $installDir = Split-Path -Parent $Executable
        $marker = Join-Path $installDir $markerName
        $validMarker = Test-Marker $marker
        $localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
        $defaultDir = Join-Path $localAppData 'Programs\Etherscan\bin'
        $legacyDefault = [IO.Path]::GetFullPath($installDir).TrimEnd('\').Equals([IO.Path]::GetFullPath($defaultDir).TrimEnd('\'), [StringComparison]::OrdinalIgnoreCase)
		try {
			Remove-Item -LiteralPath $Executable -Force
		} catch {
			$quoted = "'" + $Executable.Replace("'", "''") + "'"
			throw "Unable to remove $Executable. Open an elevated PowerShell and run: Remove-Item -LiteralPath $quoted -Force. $($_.Exception.Message)"
		}
        Write-Host "Removed $Executable"

        $other = @(Get-ChildItem -LiteralPath $installDir -Force | Where-Object { -not ($validMarker -and $_.FullName -eq $marker) } | Select-Object -First 1)
        if (($validMarker -or $legacyDefault) -and $other.Count -eq 0) {
            Remove-UserPath $installDir
            if ($validMarker) { Remove-Item -LiteralPath $marker -Force }
            Remove-Item -LiteralPath $installDir -Force -ErrorAction SilentlyContinue
        }
    }
    if (Test-Path -LiteralPath $ConfigDir) {
        Remove-Item -LiteralPath $ConfigDir -Recurse -Force
        Write-Host "Removed $ConfigDir"
    }
    Write-Host 'Etherscan CLI uninstalled.'
} catch {
    Write-Error $_
    exit 1
} finally {
    if ($CleanupScript -and -not [string]::IsNullOrWhiteSpace($PSCommandPath)) {
        Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
    }
}
`, installMarkerName, installMarkerContent)
