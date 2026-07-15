& {
    Set-StrictMode -Version 2.0
    $ErrorActionPreference = "Stop"
    $ProgressPreference = "SilentlyContinue"

    $repository = "etherscan/etherscan-cli"
    $tempDir = $null
    $previousSecurityProtocol = [Net.ServicePointManager]::SecurityProtocol

    function Fail {
        param([string]$Message)
        throw "etherscan installer: $Message"
    }

    function Download-File {
        param(
            [string]$Uri,
            [string]$OutFile
        )

        Invoke-WebRequest -Uri $Uri -OutFile $OutFile -UseBasicParsing
    }

    try {
        if ($env:OS -ne "Windows_NT") {
            Fail "only Windows is supported by this installer"
        }

        [Net.ServicePointManager]::SecurityProtocol =
            $previousSecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

        $architecture = $null
        try {
            $architecture = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
        }
        catch {
            $architecture = $env:PROCESSOR_ARCHITEW6432
            if ([string]::IsNullOrWhiteSpace($architecture)) {
                $architecture = $env:PROCESSOR_ARCHITECTURE
            }
        }

        switch ($architecture.ToUpperInvariant()) {
            { $_ -in @("X64", "AMD64") } { $arch = "amd64"; break }
            { $_ -in @("ARM64", "AARCH64") } { $arch = "arm64"; break }
            default { Fail "unsupported architecture: $architecture" }
        }

        if (-not [string]::IsNullOrWhiteSpace($env:VERSION)) {
            $tag = $env:VERSION.Trim()
            if (-not $tag.StartsWith("v")) {
                $tag = "v$tag"
            }
        }
        else {
            $release = Invoke-RestMethod `
                -Uri "https://api.github.com/repos/$repository/releases/latest" `
                -Headers @{ "User-Agent" = "etherscan-installer" }
            $tag = [string]$release.tag_name
        }

        if ([string]::IsNullOrWhiteSpace($tag)) {
            Fail "could not determine the latest release"
        }

        $version = $tag -replace "^v", ""
        $archive = "etherscan_${version}_windows_${arch}.zip"
        $baseUrl = "https://github.com/$repository/releases/download/$tag"
        $tempDir = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
        $archivePath = Join-Path $tempDir $archive
        $checksumsPath = Join-Path $tempDir "checksums.txt"
        $extractDir = Join-Path $tempDir "extract"

        New-Item -ItemType Directory -Path $tempDir | Out-Null
        New-Item -ItemType Directory -Path $extractDir | Out-Null

        Write-Host "Downloading etherscan $tag for windows/$arch..."
        Download-File -Uri "$baseUrl/$archive" -OutFile $archivePath
        Download-File -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath

        $archivePattern = [regex]::Escape($archive)
        $checksumLine = Get-Content -Path $checksumsPath |
            Where-Object { $_ -match "^([0-9a-fA-F]{64})\s+\*?$archivePattern$" } |
            Select-Object -First 1

        if ([string]::IsNullOrWhiteSpace($checksumLine)) {
            Fail "release checksum not found for $archive"
        }

        $expected = ($checksumLine -split "\s+")[0].ToLowerInvariant()
        $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            Fail "checksum verification failed"
        }

        Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force
        $executable = Get-ChildItem -Path $extractDir -Filter "etherscan.exe" -File -Recurse |
            Select-Object -First 1
        if ($null -eq $executable) {
            Fail "release archive does not contain etherscan.exe"
        }

        if (-not [string]::IsNullOrWhiteSpace($env:INSTALL_DIR)) {
            $installDir = [Environment]::ExpandEnvironmentVariables($env:INSTALL_DIR.Trim())
            $installDir = [IO.Path]::GetFullPath($installDir)
        }
        else {
            $localAppData = [Environment]::GetFolderPath("LocalApplicationData")
            $installDir = Join-Path $localAppData "Etherscan\bin"
        }

        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $destination = Join-Path $installDir "etherscan.exe"
        Copy-Item -Path $executable.FullName -Destination $destination -Force

        $userPath = [Environment]::GetEnvironmentVariable(
            "Path",
            [EnvironmentVariableTarget]::User
        )
        $expandedInstallDir = [Environment]::ExpandEnvironmentVariables($installDir).TrimEnd("\")
        $pathEntries = @($userPath -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
        $pathContainsInstallDir = @(
            $pathEntries | Where-Object {
                [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\") -ieq $expandedInstallDir
            }
        ).Count -gt 0

        if (-not $pathContainsInstallDir) {
            if ([string]::IsNullOrWhiteSpace($userPath)) {
                $newUserPath = $installDir
            }
            else {
                $newUserPath = "$($userPath.TrimEnd(';'));$installDir"
            }
            [Environment]::SetEnvironmentVariable(
                "Path",
                $newUserPath,
                [EnvironmentVariableTarget]::User
            )
            Write-Host "Added $installDir to your user PATH."
        }

        $processPathEntries = @($env:Path -split ";")
        $processPathContainsInstallDir = @(
            $processPathEntries | Where-Object {
                -not [string]::IsNullOrWhiteSpace($_) -and
                [Environment]::ExpandEnvironmentVariables($_).TrimEnd("\") -ieq $expandedInstallDir
            }
        ).Count -gt 0
        if (-not $processPathContainsInstallDir) {
            $env:Path = "$installDir;$env:Path"
        }

        Write-Host "Installed etherscan to $destination"
        Write-Host "Run 'etherscan version' to verify the installation. If it is not found, open a new terminal first."
    }
    catch {
        if ($_.Exception.Message.StartsWith("etherscan installer:")) {
            throw
        }
        throw "etherscan installer: $($_.Exception.Message)"
    }
    finally {
        [Net.ServicePointManager]::SecurityProtocol = $previousSecurityProtocol
        if ($null -ne $tempDir -and (Test-Path -LiteralPath $tempDir)) {
            Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}
