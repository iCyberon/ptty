#Requires -Version 5.1
$ErrorActionPreference = "Stop"

$Repo = "iCyberon/ptty"
$Binary = "ptty"

function Main {
    Write-Host ""
    Write-Host "  ptty installer" -ForegroundColor Blue
    Write-Host ""

    $script:Arch = Get-Arch
    $script:Version = Get-LatestVersion

    Write-Host "  Platform:  " -NoNewline
    Write-Host "windows/$Arch" -ForegroundColor White
    Write-Host "  Version:   " -NoNewline
    Write-Host "$Version" -ForegroundColor White
    Write-Host ""

    $script:InstallDir = Choose-InstallDir
    Write-Host ""

    Download-And-Install

    Write-Host ""
    Write-Host "  Installed $Binary $Version to $InstallDir\$Binary.exe" -ForegroundColor Green
    Write-Host ""

    $envPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($envPath -notlike "*$InstallDir*") {
        Write-Host "  Note: " -NoNewline -ForegroundColor Yellow
        Write-Host "$InstallDir is not in your PATH."
        Write-Host ""
        $answer = Read-Host "  Add to PATH? [Y/n]"
        if ($answer -eq "" -or $answer -eq "Y" -or $answer -eq "y") {
            [Environment]::SetEnvironmentVariable("PATH", "$envPath;$InstallDir", "User")
            $env:PATH = "$env:PATH;$InstallDir"
            Write-Host "  Added to PATH. Restart your terminal to use '$Binary'." -ForegroundColor Green
        }
        Write-Host ""
    }
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "x86"   { return "amd64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
    if (-not $release.tag_name) {
        throw "Failed to fetch latest version from GitHub"
    }
    return $release.tag_name
}

function Choose-InstallDir {
    $default = "$env:LOCALAPPDATA\Programs\$Binary"
    $alt = "$env:ProgramFiles\$Binary"

    Write-Host "  Where should $Binary be installed?"
    Write-Host ""
    Write-Host "    1) $default (default)"
    Write-Host "    2) $alt (may require admin)"
    Write-Host "    3) Custom path"
    Write-Host ""
    $choice = Read-Host "  Choice [1]"

    switch ($choice) {
        ""  { $dir = $default }
        "1" { $dir = $default }
        "2" { $dir = $alt }
        "3" {
            $dir = Read-Host "  Path"
        }
        default { $dir = $default }
    }

    if (-not (Test-Path $dir)) {
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
    }

    return $dir
}

function Download-And-Install {
    $archiveName = "${Binary}_windows_${Arch}.zip"
    $checksumName = "checksums.txt"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version"

    $tmp = New-TemporaryFile | ForEach-Object {
        Remove-Item $_
        New-Item -ItemType Directory -Path "$($_.FullName)_dir"
    }

    try {
        Write-Host "  Downloading $archiveName..." -NoNewline
        $archivePath = Join-Path $tmp.FullName $archiveName
        Invoke-WebRequest -Uri "$downloadUrl/$archiveName" -OutFile $archivePath -UseBasicParsing
        Write-Host " done" -ForegroundColor Green

        Write-Host "  Verifying checksum..." -NoNewline
        $checksumPath = Join-Path $tmp.FullName $checksumName
        Invoke-WebRequest -Uri "$downloadUrl/$checksumName" -OutFile $checksumPath -UseBasicParsing

        $checksumLine = Get-Content $checksumPath | Where-Object { $_ -match $archiveName }
        $expected = ($checksumLine -split "\s+")[0]
        $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()

        if ($actual -ne $expected) {
            throw "Checksum mismatch!`n  Expected: $expected`n  Got:      $actual"
        }
        Write-Host " done" -ForegroundColor Green

        Write-Host "  Extracting..." -NoNewline
        $extractDir = Join-Path $tmp.FullName "extract"
        Expand-Archive -Path $archivePath -DestinationPath $extractDir
        Write-Host " done" -ForegroundColor Green

        Write-Host "  Installing to $InstallDir..." -NoNewline
        Copy-Item (Join-Path $extractDir "$Binary.exe") -Destination (Join-Path $InstallDir "$Binary.exe") -Force
        Write-Host " done" -ForegroundColor Green
    }
    finally {
        Remove-Item -Recurse -Force $tmp.FullName -ErrorAction SilentlyContinue
    }
}

Main
