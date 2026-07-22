[CmdletBinding()]
param(
    [string]$Version = $env:ETHERSCAN_VERSION,
    [string]$InstallDir = $env:ETHERSCAN_INSTALL_DIR,
    [switch]$NoPathUpdate,
    [int]$WaitForProcessId = 0,
    [switch]$CleanupScript
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

$Repository = "etherscan/etherscan-cli"
$DownloadBaseUrl = $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL

function Get-EtherscanArchitecture {
    $architecture = $env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrWhiteSpace($architecture)) {
        $architecture = $env:PROCESSOR_ARCHITECTURE
    }

    switch -Regex ($architecture) {
        "^(AMD64|x86_64)$" { return "amd64" }
        "^(ARM64|aarch64)$" { return "arm64" }
        default { throw "Unsupported Windows architecture: $architecture. Etherscan CLI supports amd64 and arm64." }
    }
}

function Get-GitHubApiHeaders {
    $headers = @{
        Accept = "application/vnd.github+json"
        "User-Agent" = "etherscan-cli-installer"
        "X-GitHub-Api-Version" = "2022-11-28"
    }
    if (-not [string]::IsNullOrWhiteSpace($env:ETHERSCAN_GITHUB_TOKEN)) {
        $headers.Authorization = "Bearer $($env:ETHERSCAN_GITHUB_TOKEN)"
    }
    return $headers
}

function Resolve-EtherscanVersion {
    param([string]$RequestedVersion)

    if (-not [string]::IsNullOrWhiteSpace($RequestedVersion) -and $RequestedVersion -ne "latest") {
        $tag = if ($RequestedVersion.StartsWith("v")) { $RequestedVersion } else { "v$RequestedVersion" }
    }
    else {
        if (-not [string]::IsNullOrWhiteSpace($DownloadBaseUrl)) {
            throw "A version is required when the installer test download source is used."
        }
        $release = Invoke-RestMethod `
            -Uri "https://api.github.com/repos/$Repository/releases/latest" `
            -Headers (Get-GitHubApiHeaders)
        $tag = [string]$release.tag_name
    }

    if ($tag -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$') {
        throw "Invalid release version: $tag"
    }

    return @{
        Tag = $tag
        Version = $tag.Substring(1)
    }
}

function Copy-InstallerFile {
    param(
        [string]$Base,
        [string]$Name,
        [string]$Destination
    )

    if (Test-Path -LiteralPath $Base -PathType Container) {
        Copy-Item -LiteralPath (Join-Path $Base $Name) -Destination $Destination
        return
    }

    $uri = "$($Base.TrimEnd('/'))/$Name"
    $parsedUri = [Uri]$uri
    if ($parsedUri.Scheme -ne "https") {
        throw "Remote downloads must use HTTPS: $uri"
    }

    $headers = @{
        "User-Agent" = "etherscan-cli-installer"
    }
    if ($parsedUri.Host -in @("github.com", "api.github.com") -and
        -not [string]::IsNullOrWhiteSpace($env:ETHERSCAN_GITHUB_TOKEN)) {
        $headers.Authorization = "Bearer $($env:ETHERSCAN_GITHUB_TOKEN)"
    }

    Invoke-WebRequest -Uri $uri -OutFile $Destination -Headers $headers -UseBasicParsing
}

function Add-EtherscanToUserPath {
    param([string]$Directory)

    $fullDirectory = [IO.Path]::GetFullPath($Directory).TrimEnd('\')
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $entries = @($userPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $alreadyPresent = $entries | Where-Object {
        try {
            $expandedEntry = [Environment]::ExpandEnvironmentVariables($_)
            [IO.Path]::GetFullPath($expandedEntry).TrimEnd('\').Equals($fullDirectory, [StringComparison]::OrdinalIgnoreCase)
        }
        catch {
            $_.TrimEnd('\').Equals($fullDirectory, [StringComparison]::OrdinalIgnoreCase)
        }
    }

    if (-not $alreadyPresent) {
        $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
            $fullDirectory
        }
        else {
            "$($userPath.TrimEnd(';'));$fullDirectory"
        }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "Added $fullDirectory to your user PATH."
    }

    $processEntries = @($env:Path -split ';')
    if (-not ($processEntries | Where-Object { $_.TrimEnd('\').Equals($fullDirectory, [StringComparison]::OrdinalIgnoreCase) })) {
        $env:Path = "$env:Path;$fullDirectory"
    }
}

if ($env:OS -ne "Windows_NT") {
    throw "This installer supports Windows only. Use install.sh on macOS or Linux."
}

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    $localAppData = [Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)
    $InstallDir = Join-Path $localAppData "Programs\Etherscan\bin"
}
if ($InstallDir.Contains(';')) {
    throw "The installation directory cannot contain a semicolon."
}
if ($InstallDir.IndexOfAny([char[]]"`r`n") -ge 0) {
    throw "The installation directory cannot contain a line break."
}
if ($WaitForProcessId -gt 0) {
    Wait-Process -Id $WaitForProcessId -ErrorAction SilentlyContinue
}

$resolved = Resolve-EtherscanVersion -RequestedVersion $Version
$architecture = Get-EtherscanArchitecture
$archiveName = "etherscan_$($resolved.Version)_windows_$architecture.zip"
$baseUrl = if ([string]::IsNullOrWhiteSpace($DownloadBaseUrl)) {
    "https://github.com/$Repository/releases/download/$($resolved.Tag)"
}
else {
    $DownloadBaseUrl
}

$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) "etherscan-install-$PID-$([Guid]::NewGuid().ToString('N'))"
$archivePath = Join-Path $tempDirectory $archiveName
$checksumPath = Join-Path $tempDirectory "checksums.txt"
$sourceExecutable = Join-Path $tempDirectory "etherscan.exe"

try {
    New-Item -ItemType Directory -Path $tempDirectory -Force | Out-Null

    Write-Host "Downloading Etherscan CLI $($resolved.Version) for windows/$architecture..."
    Copy-InstallerFile -Base $baseUrl -Name $archiveName -Destination $archivePath
    Copy-InstallerFile -Base $baseUrl -Name "checksums.txt" -Destination $checksumPath

    $pattern = '^([0-9A-Fa-f]{64})\s+\*?' + [Regex]::Escape($archiveName) + '$'
    $checksumLine = Get-Content -LiteralPath $checksumPath | Where-Object { $_ -match $pattern } | Select-Object -First 1
    if (-not $checksumLine -or $checksumLine -notmatch $pattern) {
        throw "No checksum was published for $archiveName."
    }

    $expectedHash = $Matches[1].ToLowerInvariant()
    $actualHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $expectedHash) {
        throw "Checksum verification failed for $archiveName. Expected $expectedHash, received $actualHash."
    }

    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [IO.Compression.ZipFile]::OpenRead($archivePath)
    try {
        $executableEntries = @($zip.Entries | Where-Object { $_.FullName.Replace('\', '/') -eq "etherscan.exe" })
        if ($executableEntries.Count -ne 1) {
            throw "$archiveName must contain exactly one root-level etherscan.exe."
        }

        $inputStream = $executableEntries[0].Open()
        $outputStream = [IO.File]::Create($sourceExecutable)
        try {
            $inputStream.CopyTo($outputStream)
        }
        finally {
            $outputStream.Dispose()
            $inputStream.Dispose()
        }
    }
    finally {
        $zip.Dispose()
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $targetExecutable = Join-Path $InstallDir "etherscan.exe"
    $stagedExecutable = Join-Path $InstallDir ".etherscan.exe.new-$PID"
    $backupExecutable = Join-Path $InstallDir ".etherscan.exe.old-$PID"
    Copy-Item -LiteralPath $sourceExecutable -Destination $stagedExecutable -Force

    try {
        if (Test-Path -LiteralPath $targetExecutable) {
            Move-Item -LiteralPath $targetExecutable -Destination $backupExecutable -Force
        }
        Move-Item -LiteralPath $stagedExecutable -Destination $targetExecutable -Force
        Remove-Item -LiteralPath $backupExecutable -Force -ErrorAction SilentlyContinue
    }
    catch {
        Remove-Item -LiteralPath $stagedExecutable -Force -ErrorAction SilentlyContinue
        if ((Test-Path -LiteralPath $backupExecutable) -and -not (Test-Path -LiteralPath $targetExecutable)) {
            Move-Item -LiteralPath $backupExecutable -Destination $targetExecutable -Force
        }
        throw
    }

    if (-not $NoPathUpdate) {
        Add-EtherscanToUserPath -Directory $InstallDir
    }

    Write-Host ""
    Write-Host "Etherscan CLI $($resolved.Version) installed successfully."
    Write-Host "Installed to: $targetExecutable"
    if ($NoPathUpdate) {
        Write-Host "Add $InstallDir to PATH to run etherscan from any directory."
    }
    else {
        Write-Host "Run 'etherscan version' to verify the installation."
        Write-Host "Open a new terminal if the command is not yet available."
    }
}
finally {
    Remove-Item -LiteralPath $tempDirectory -Recurse -Force -ErrorAction SilentlyContinue
    if ($CleanupScript -and -not [string]::IsNullOrWhiteSpace($PSCommandPath)) {
        Remove-Item -LiteralPath $PSCommandPath -Force -ErrorAction SilentlyContinue
    }
}
