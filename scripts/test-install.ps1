$ErrorActionPreference = "Stop"

$installer = Join-Path $PSScriptRoot "install.ps1"
$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) "etherscan-installer-test-$PID-$([Guid]::NewGuid().ToString('N'))"
$fixtureDirectory = Join-Path $tempDirectory "fixtures"
$bundleDirectory = Join-Path $tempDirectory "bundle"
$installDirectory = Join-Path $tempDirectory "install dir"
$version = "9.9.9-test.1"

$architecture = $env:PROCESSOR_ARCHITEW6432
if ([string]::IsNullOrWhiteSpace($architecture)) {
    $architecture = $env:PROCESSOR_ARCHITECTURE
}
$goArchitecture = switch -Regex ($architecture) {
    "^(AMD64|x86_64)$" { "amd64" }
    "^(ARM64|aarch64)$" { "arm64" }
    default { throw "Unsupported test architecture: $architecture" }
}
$archiveName = "etherscan_${version}_windows_$goArchitecture.zip"
$archivePath = Join-Path $fixtureDirectory $archiveName
$checksumPath = Join-Path $fixtureDirectory "checksums.txt"
$previousDownloadBaseUrl = $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL

function Write-Fixture {
    param([string]$Content)

    Remove-Item -LiteralPath $bundleDirectory -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $archivePath -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path $bundleDirectory -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $bundleDirectory "etherscan.exe") -Value $Content -NoNewline
    Compress-Archive -Path (Join-Path $bundleDirectory "*") -DestinationPath $archivePath
    $hash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath $checksumPath -Value "$hash  $archiveName"
}

try {
    New-Item -ItemType Directory -Path $fixtureDirectory -Force | Out-Null
    $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL = $fixtureDirectory

    Write-Fixture -Content "first"
    & $installer -Version $version -InstallDir $installDirectory -NoPathUpdate
    $installed = Join-Path $installDirectory "etherscan.exe"
    if ((Get-Content -LiteralPath $installed -Raw) -ne "first") {
        throw "fresh installation did not install the expected executable"
    }

    Write-Fixture -Content "second"
    & $installer -Version $version -InstallDir $installDirectory -NoPathUpdate
    if ((Get-Content -LiteralPath $installed -Raw) -ne "second") {
        throw "reinstallation did not replace the executable"
    }

    Set-Content -LiteralPath $checksumPath -Value "$('0' * 64)  $archiveName"
    $failed = $false
    try {
        & $installer -Version $version -InstallDir $installDirectory -NoPathUpdate
    }
    catch {
        $failed = $_.Exception.Message -like "Checksum verification failed*"
    }
    if (-not $failed) {
        throw "installer accepted an invalid checksum"
    }
    if ((Get-Content -LiteralPath $installed -Raw) -ne "second") {
        throw "failed verification modified the installed executable"
    }

    $failed = $false
    try {
        $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL = "http://example.invalid"
        & $installer -Version $version -InstallDir $installDirectory -NoPathUpdate
    }
    catch {
        $failed = $_.Exception.Message -like "Remote downloads must use HTTPS*"
    }
    if (-not $failed) {
        throw "installer accepted an insecure download URL"
    }

    Write-Host "PowerShell installer tests passed."
}
finally {
    $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL = $previousDownloadBaseUrl
    Remove-Item -LiteralPath $tempDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
