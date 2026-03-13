# ============================================================
# install.ps1 — Install tt to user-local directory
#
# Installs tt.exe to %LOCALAPPDATA%\Axsh\Tokotachi\bin
# and adds it to the user PATH. No admin privileges required.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File .\scripts\dist\install.ps1
# ============================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ─── Configuration ──────────────────────────────────────────
$BinaryName    = "tt"
$FeatureDir    = "features\tt"
$InstallDir    = Join-Path $env:LOCALAPPDATA "Axsh\Tokotachi\bin"

# ─── Colored output helpers ─────────────────────────────────
function Write-Info  { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Blue }
function Write-Pass  { param([string]$Message) Write-Host "[PASS] $Message" -ForegroundColor Green }
function Write-Fail  { param([string]$Message) Write-Host "[FAIL] $Message" -ForegroundColor Red }
function Write-Warn  { param([string]$Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }

# ─── Resolve project root ───────────────────────────────────
# This script lives at scripts/dist/install.ps1
$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = (Resolve-Path (Join-Path $ScriptDir "..\..")).Path

# ─── Step 1: Ensure binary exists ───────────────────────────
$BinDir       = Join-Path $ProjectRoot "bin"
$SourceBinary = Join-Path $BinDir $BinaryName

if (-not (Test-Path $SourceBinary)) {
    Write-Warn "Binary not found at: $SourceBinary"
    Write-Info "Building tt..."

    # Check that Go is available
    $goExe = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goExe) {
        Write-Fail "Go is not installed. Please install Go and try again."
        exit 1
    }

    # Check that the feature source exists
    $FeaturePath = Join-Path $ProjectRoot $FeatureDir
    if (-not (Test-Path (Join-Path $FeaturePath "go.mod"))) {
        Write-Fail "Feature source not found: $FeaturePath"
        exit 1
    }

    # Create bin directory
    if (-not (Test-Path $BinDir)) {
        New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    }

    # Build tt using go build
    Push-Location $FeaturePath
    try {
        & go build -o $SourceBinary .
        if ($LASTEXITCODE -ne 0) {
            Write-Fail "Build failed. Please fix errors and try again."
            exit 1
        }
    } finally {
        Pop-Location
    }

    if (-not (Test-Path $SourceBinary)) {
        Write-Fail "Binary still not found after build: $SourceBinary"
        exit 1
    }
}

Write-Pass "Binary found: $SourceBinary"

# ─── Step 2: Create install directory ───────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Write-Info "Created directory: $InstallDir"
} else {
    Write-Info "Directory already exists: $InstallDir"
}

# ─── Step 3: Copy binary ───────────────────────────────────
$DestBinary = Join-Path $InstallDir "$BinaryName.exe"
Copy-Item -Path $SourceBinary -Destination $DestBinary -Force
Write-Pass "Installed: $DestBinary"

# ─── Step 4: Add to user PATH ──────────────────────────────
$UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)

# Normalize paths for comparison
$pathEntries = $UserPath -split ";" | Where-Object { $_ -ne "" }
$alreadyInPath = $pathEntries | Where-Object {
    $_.TrimEnd('\') -eq $InstallDir.TrimEnd('\')
}

if ($alreadyInPath) {
    Write-Info "PATH already contains: $InstallDir"
} else {
    $newPath = ($pathEntries + $InstallDir) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
    Write-Pass "Added to user PATH: $InstallDir"
}

# ─── Done ───────────────────────────────────────────────────
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Pass "Installation complete!"
Write-Host ""
Write-Host "  Install location : $DestBinary"
Write-Host "  PATH configured  : $InstallDir"
Write-Host ""
Write-Host "  NOTE: Open a NEW terminal for PATH changes to take effect." -ForegroundColor Yellow
Write-Host "  Then run: tt --help" -ForegroundColor Yellow
Write-Host "============================================" -ForegroundColor Cyan
