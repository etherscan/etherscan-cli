$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$version = "0.0.0-development"
$tempDirectory = Join-Path ([IO.Path]::GetTempPath()) "etherscan-npm-test-$PID-$([Guid]::NewGuid().ToString('N'))"
$bundleDirectory = Join-Path $tempDirectory "bundle"
$prefixDirectory = Join-Path $tempDirectory "global prefix"
$missingPrefix = Join-Path $tempDirectory "missing prefix"
$processorArchitecture = if ([string]::IsNullOrWhiteSpace($env:PROCESSOR_ARCHITEW6432)) {
    $env:PROCESSOR_ARCHITECTURE
}
else {
    $env:PROCESSOR_ARCHITEW6432
}
$npmArchitecture = if ($processorArchitecture -match '^(ARM64|aarch64)$') { "arm64" } else { "x64" }
$platformName = "cli-win32-$npmArchitecture"
$platformStage = Join-Path $tempDirectory $platformName
$previousNpmCache = $env:npm_config_cache
$previousGoCache = $env:GOCACHE

function Invoke-Checked {
    param([string]$Command, [string[]]$Arguments)
    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Command failed with exit code $LASTEXITCODE"
    }
}

try {
    New-Item -ItemType Directory -Path $bundleDirectory, $platformStage -Force | Out-Null
    $env:GOCACHE = Join-Path $tempDirectory "go-cache"
    $env:npm_config_cache = Join-Path $tempDirectory "npm-cache"

    Push-Location $repositoryRoot
    try {
        Invoke-Checked npm.cmd @("run", "test:release")
        $fixtureExecutable = Join-Path $bundleDirectory "etherscan.exe"
        Invoke-Checked go @("build", "-buildvcs=false", "-ldflags", "-X main.version=$version", "-o", $fixtureExecutable, "./cmd/etherscan")
        Copy-Item -LiteralPath (Join-Path $repositoryRoot "npm\$platformName\package.json") -Destination $platformStage
        Copy-Item -LiteralPath (Join-Path $repositoryRoot "LICENSE") -Destination $platformStage
        Copy-Item -LiteralPath $fixtureExecutable -Destination (Join-Path $platformStage "etherscan.exe")

        $platformPack = (& npm.cmd pack $platformStage --json --pack-destination $tempDirectory | ConvertFrom-Json)
        if ($LASTEXITCODE -ne 0) { throw "platform npm pack failed" }
        $umbrellaPack = (& npm.cmd pack --json --pack-destination $tempDirectory | ConvertFrom-Json)
        if ($LASTEXITCODE -ne 0) { throw "umbrella npm pack failed" }
    }
    finally {
        Pop-Location
    }

    $platformTarball = Join-Path $tempDirectory $platformPack.filename
    $umbrellaTarball = Join-Path $tempDirectory $umbrellaPack.filename
    Invoke-Checked npm.cmd @("install", "--global", "--ignore-scripts", "--prefix", $prefixDirectory, $platformTarball, $umbrellaTarball)

    $globalOutput = & (Join-Path $prefixDirectory "etherscan.cmd") version
    if ($LASTEXITCODE -ne 0 -or ($globalOutput -join "`n").Trim() -ne $version) {
        throw "global npm installation returned an unexpected version: $globalOutput"
    }

    $npxOutput = & npx.cmd --yes --package $platformTarball --package $umbrellaTarball etherscan version
    if ($LASTEXITCODE -ne 0 -or ($npxOutput -join "`n").Trim() -ne $version) {
        throw "npx returned an unexpected version: $npxOutput"
    }

    $savedErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    & (Join-Path $prefixDirectory "etherscan.cmd") definitely-not-a-command *> $null
    $argumentExitCode = $LASTEXITCODE
    $ErrorActionPreference = $savedErrorAction
    if ($argumentExitCode -eq 0) { throw "npm launcher did not forward a failing command" }

    Invoke-Checked npm.cmd @("install", "--global", "--ignore-scripts", "--omit=optional", "--prefix", $missingPrefix, $umbrellaTarball)
    $ErrorActionPreference = "Continue"
    $missingOutput = (& (Join-Path $missingPrefix "etherscan.cmd") version 2>&1) -join "`n"
    $ErrorActionPreference = $savedErrorAction
    if ($missingOutput -notlike "*without --omit=optional*") {
        throw "missing platform package did not produce an actionable error: $missingOutput"
    }

    $global:LASTEXITCODE = 0
    Write-Host "npm package tests passed."
}
finally {
    $env:npm_config_cache = $previousNpmCache
    $env:GOCACHE = $previousGoCache
    Remove-Item -LiteralPath $tempDirectory -Recurse -Force -ErrorAction SilentlyContinue
}
