# ============================================================
# uninstall.ps1 — Uninstall tt from user-local directory
#
# Removes tt.exe from %LOCALAPPDATA%\Axsh\Tokotachi\bin
# and removes it from the user PATH.
#
# Usage:
#   powershell -ExecutionPolicy Bypass -File .\scripts\dist\uninstall.ps1
# ============================================================

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ─── Configuration ──────────────────────────────────────────
$BinaryName = "tt"
$InstallDir = Join-Path $env:LOCALAPPDATA "Axsh\Tokotachi\bin"

# ─── Colored output helpers ─────────────────────────────────
function Write-Info  { param([string]$Message) Write-Host "[INFO] $Message" -ForegroundColor Blue }
function Write-Pass  { param([string]$Message) Write-Host "[PASS] $Message" -ForegroundColor Green }
function Write-Fail  { param([string]$Message) Write-Host "[FAIL] $Message" -ForegroundColor Red }
function Write-Warn  { param([string]$Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }

# ─── Confirmation prompt ────────────────────────────────────
$DestBinary = Join-Path $InstallDir "$BinaryName.exe"

if (-not (Test-Path $DestBinary)) {
    Write-Warn "$BinaryName is not installed at: $DestBinary"
    Write-Info "Nothing to uninstall."
    exit 0
}

Write-Host ""
Write-Host "This will uninstall $BinaryName from:" -ForegroundColor Yellow
Write-Host "  $InstallDir" -ForegroundColor Yellow
Write-Host ""

$confirm = Read-Host "Proceed? (y/N)"
if ($confirm -ne "y" -and $confirm -ne "Y") {
    Write-Info "Uninstall cancelled."
    exit 0
}

# ─── Step 1: Remove binary ─────────────────────────────────
Remove-Item -Path $DestBinary -Force
Write-Pass "Removed: $DestBinary"

# ─── Step 2: Remove from user PATH ─────────────────────────
$UserPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User)
$pathEntries = $UserPath -split ";" | Where-Object { $_ -ne "" }

$filteredEntries = $pathEntries | Where-Object {
    $_.TrimEnd('\') -ne $InstallDir.TrimEnd('\')
}

if ($filteredEntries.Count -lt $pathEntries.Count) {
    $newPath = $filteredEntries -join ";"
    [Environment]::SetEnvironmentVariable("Path", $newPath, [EnvironmentVariableTarget]::User)
    Write-Pass "Removed from user PATH: $InstallDir"
} else {
    Write-Info "PATH did not contain: $InstallDir"
}

# ─── Step 3: Clean up empty directories ────────────────────
# Remove bin/ -> Tokotachi/ -> Axsh/ if empty
$dirsToClean = @(
    $InstallDir,
    (Split-Path $InstallDir -Parent),    # Tokotachi
    (Split-Path (Split-Path $InstallDir -Parent) -Parent)  # Axsh
)

foreach ($dir in $dirsToClean) {
    if ((Test-Path $dir) -and ((Get-ChildItem $dir -Force).Count -eq 0)) {
        Remove-Item -Path $dir -Force
        Write-Info "Removed empty directory: $dir"
    }
}

# ─── Done ───────────────────────────────────────────────────
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Pass "Uninstall complete!"
Write-Host ""
Write-Host "  NOTE: Open a NEW terminal for PATH changes to take effect." -ForegroundColor Yellow
Write-Host "============================================" -ForegroundColor Cyan
