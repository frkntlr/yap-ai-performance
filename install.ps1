# PowerShell 5.1+ Required
# Thin Windows bootstrap for Yap AI Performance.
# Installs minimal deps if needed, locates or builds yap.exe, then runs: yap install
#
# Run with:
#   powershell -ExecutionPolicy Bypass -File install.ps1
# Optional:
#   powershell -ExecutionPolicy Bypass -File install.ps1 -DryRun
#   powershell -ExecutionPolicy Bypass -File install.ps1 -SkipBuild

param(
    [switch]$DryRun,
    [switch]$SkipBuild
)

$ErrorActionPreference = "Stop"

function Write-Blue   { param($msg) Write-Host $msg -ForegroundColor Cyan }
function Write-Green  { param($msg) Write-Host $msg -ForegroundColor Green }
function Write-Yellow { param($msg) Write-Host $msg -ForegroundColor Yellow }
function Write-Red    { param($msg) Write-Host $msg -ForegroundColor Red }

function Refresh-Path {
    $env:PATH = [System.Environment]::GetEnvironmentVariable("PATH", "Machine") + ";" +
                [System.Environment]::GetEnvironmentVariable("PATH", "User")
}

function Test-Cmd {
    param([string]$Name)
    try {
        $null = Get-Command $Name -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

Write-Blue "===================================================="
Write-Blue "     Yap AI Performance — Windows Bootstrap"
Write-Blue "===================================================="
Write-Host "This script prepares the environment and delegates to 'yap install'."
Write-Host ""

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ScriptDir

# ── winget ──────────────────────────────────────────────────
$wingetAvailable = Test-Cmd "winget"
if ($wingetAvailable) {
    Write-Green "winget is available."
} else {
    Write-Yellow "winget not found. Some auto-installs may be skipped."
}

# ── Python ──────────────────────────────────────────────────
$pythonCmd = $null
foreach ($cmd in @("python", "py", "python3")) {
    if (Test-Cmd $cmd) {
        try {
            $ver = & $cmd --version 2>&1
            if ($ver -match "Python 3") {
                $pythonCmd = $cmd
                Write-Green "Python found: $ver ($cmd)"
                break
            }
        } catch {}
    }
}
if (-not $pythonCmd) {
    if ($wingetAvailable) {
        Write-Yellow "Installing Python 3.12 via winget..."
        winget install --id Python.Python.3.12 --silent --accept-package-agreements --accept-source-agreements
        Refresh-Path
        $pythonCmd = "python"
    } else {
        Write-Red "Python 3 is required. Install from https://www.python.org/downloads/ and re-run."
        exit 1
    }
}

# ── Git ─────────────────────────────────────────────────────
if (-not (Test-Cmd "git")) {
    if ($wingetAvailable) {
        Write-Yellow "Installing Git via winget..."
        winget install --id Git.Git --silent --accept-package-agreements --accept-source-agreements
        Refresh-Path
    } else {
        Write-Yellow "git not found; continuing (yap install may install it)."
    }
}

# ── Go (optional, for local build) ──────────────────────────
$hasGo = Test-Cmd "go"

# ── Locate yap.exe ──────────────────────────────────────────
$yapCandidates = @(
    (Join-Path $ScriptDir "dist\yap-windows-amd64.exe"),
    (Join-Path $ScriptDir "dist\yap.exe"),
    (Join-Path $env:LOCALAPPDATA "Programs\yap\yap.exe")
)
$yapExe = $null
foreach ($c in $yapCandidates) {
    if (Test-Path $c) {
        $yapExe = $c
        break
    }
}
if (-not $yapExe -and (Test-Cmd "yap")) {
    $yapExe = (Get-Command yap).Source
}

if (-not $yapExe) {
    if ($SkipBuild) {
        Write-Red "yap.exe not found and -SkipBuild was set."
        Write-Red "Build with: go build -o dist\yap-windows-amd64.exe .\cmd\yap"
        Write-Red "Or download a release binary into dist\ and re-run."
        exit 1
    }
    if (-not $hasGo) {
        Write-Red "yap.exe not found and Go is not installed."
        Write-Red "Either install Go (https://go.dev/dl/) and re-run, or place yap.exe under dist\."
        exit 1
    }
    Write-Yellow "Building yap.exe via Go..."
    New-Item -ItemType Directory -Force -Path (Join-Path $ScriptDir "dist") | Out-Null
    $out = Join-Path $ScriptDir "dist\yap-windows-amd64.exe"
    & go build -ldflags="-s -w" -o $out .\cmd\yap
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path $out)) {
        Write-Red "go build failed."
        exit 1
    }
    $yapExe = $out
    Write-Green "Built: $yapExe"
} else {
    Write-Green "Using yap: $yapExe"
}

# ── Delegate to yap install ─────────────────────────────────
Write-Blue "`nRunning yap install (source of truth)..."
$installArgs = @("install")
if ($DryRun) {
    $installArgs += "--dry-run"
}

& $yapExe @installArgs
$code = $LASTEXITCODE
if ($code -ne 0) {
    Write-Red "yap install exited with code $code"
    exit $code
}

Write-Host ""
Write-Green "======================================================"
Write-Green "  Bootstrap finished. Run: yap status"
Write-Green "======================================================"
Write-Host "If 'yap' is not found in a new terminal, reopen the shell so PATH refreshes."
Write-Host "Local binary dir: $env:LOCALAPPDATA\Programs\yap"
