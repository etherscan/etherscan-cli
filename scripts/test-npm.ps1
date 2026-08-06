$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$version = "0.0.0-development"
$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) "etherscan-npm-test-$PID-$([Guid]::NewGuid().ToString('N'))"
$fixtureDirectory = Join-Path $tempDirectory "fixtures"
$bundleDirectory = Join-Path $tempDirectory "bundle"
$prefixDirectory = Join-Path $tempDirectory "global prefix"
$ignoredPrefix = Join-Path $tempDirectory "ignored prefix"
$invalidPrefix = Join-Path $tempDirectory "invalid prefix"
$processorArchitecture = if ([string]::IsNullOrWhiteSpace($env:PROCESSOR_ARCHITEW6432)) {
    $env:PROCESSOR_ARCHITECTURE
}
else {
    $env:PROCESSOR_ARCHITEW6432
}
$architecture = if ($processorArchitecture -match '^(ARM64|aarch64)$') { "arm64" } else { "amd64" }
$archiveName = "etherscan_${version}_windows_$architecture.zip"
$archivePath = Join-Path $fixtureDirectory $archiveName
$checksumPath = Join-Path $fixtureDirectory "checksums.txt"
$previousDownloadBase = $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL
$previousNpmCache = $env:npm_config_cache
$previousGoCache = $env:GOCACHE

function Invoke-Checked {
    param(
        [string]$Command,
        [string[]]$Arguments
    )
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE"
    }
}

try {
    New-Item -ItemType Directory -Path $fixtureDirectory, $bundleDirectory -Force | Out-Null
    $env:GOCACHE = Join-Path $tempDirectory "go-cache"
    $env:npm_config_cache = Join-Path $tempDirectory "npm-cache"

    $fixtureExecutable = Join-Path $bundleDirectory "etherscan.exe"
    Push-Location $repositoryRoot
    try {
        Invoke-Checked npm.cmd @("run", "test:release")
        Invoke-Checked go @("build", "-buildvcs=false", "-ldflags", "-X main.version=$version", "-o", $fixtureExecutable, "./cmd/etherscan")
        Compress-Archive -LiteralPath $fixtureExecutable -DestinationPath $archivePath
        $hash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
        Set-Content -LiteralPath $checksumPath -Value "$hash  $archiveName"

        $packResult = (& npm.cmd pack --json --pack-destination $tempDirectory | ConvertFrom-Json)
        if ($LASTEXITCODE -ne 0) {
            throw "npm pack failed with exit code $LASTEXITCODE"
        }
        $tarball = Join-Path $tempDirectory $packResult.filename
    }
    finally {
        Pop-Location
    }

    $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL = $fixtureDirectory
    $savedPSModulePath = $env:PSModulePath
    $env:PSModulePath = ""
    try {
        Invoke-Checked npm.cmd @("install", "--global", "--prefix", $prefixDirectory, $tarball)
    }
    finally {
        $env:PSModulePath = $savedPSModulePath
    }
    $globalOutput = & (Join-Path $prefixDirectory "etherscan.cmd") version
    if ($LASTEXITCODE -ne 0 -or ($globalOutput -join "`n").Trim() -ne $version) {
        throw "global npm installation returned unexpected version: $globalOutput"
    }

    $npxOutput = & npx.cmd --yes --package $tarball etherscan version
    if ($LASTEXITCODE -ne 0 -or ($npxOutput -join "`n").Trim() -ne $version) {
        throw "npx returned unexpected version: $npxOutput"
    }

    $savedErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & (Join-Path $prefixDirectory "etherscan.cmd") definitely-not-a-command *> $null
    $argumentExitCode = $LASTEXITCODE
    $ErrorActionPreference = $savedErrorAction
    if ($argumentExitCode -eq 0) {
        throw "npm launcher did not forward a failing command or its exit code"
    }

    Invoke-Checked npm.cmd @("install", "--global", "--ignore-scripts", "--prefix", $ignoredPrefix, $tarball)
    $ErrorActionPreference = "Continue"
    & (Join-Path $ignoredPrefix "etherscan.cmd") version 2>$null
    $ignoredExitCode = $LASTEXITCODE
    $ErrorActionPreference = $savedErrorAction
    if ($ignoredExitCode -eq 0) {
        throw "launcher succeeded after lifecycle scripts were disabled"
    }

    $savedProcessorArchitecture = $env:PROCESSOR_ARCHITECTURE
    $savedProcessorArchitectureW6432 = $env:PROCESSOR_ARCHITEW6432
    $env:PROCESSOR_ARCHITECTURE = "x86"
    $env:PROCESSOR_ARCHITEW6432 = $null
    $ErrorActionPreference = "Continue"
    Push-Location $repositoryRoot
    try {
        & node npm/postinstall.js *> $null
        $unsupportedExitCode = $LASTEXITCODE
    }
    finally {
        Pop-Location
        $env:PROCESSOR_ARCHITECTURE = $savedProcessorArchitecture
        $env:PROCESSOR_ARCHITEW6432 = $savedProcessorArchitectureW6432
        $ErrorActionPreference = $savedErrorAction
    }
    if ($unsupportedExitCode -eq 0) {
        throw "npm postinstall accepted an unsupported architecture"
    }

    Set-Content -LiteralPath $checksumPath -Value "$('0' * 64)  $archiveName"
    $ErrorActionPreference = "Continue"
    & npm.cmd install --global --prefix $invalidPrefix $tarball *> $null
    $invalidExitCode = $LASTEXITCODE
    $ErrorActionPreference = $savedErrorAction
    if ($invalidExitCode -eq 0) {
        throw "npm installation accepted an invalid release checksum"
    }

    $global:LASTEXITCODE = 0
    Write-Host "npm package tests passed."
}
finally {
    $env:ETHERSCAN_INSTALL_TEST_DOWNLOAD_BASE_URL = $previousDownloadBase
    $env:npm_config_cache = $previousNpmCache
    $env:GOCACHE = $previousGoCache
    Remove-Item -LiteralPath $tempDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
