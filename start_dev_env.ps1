param(
    [switch]$SkipBuild,
    [int]$DockerStartupTimeoutSec = 180,
    [int]$AppStartupTimeoutSec = 600
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Set-Location $PSScriptRoot

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Ok {
    param([string]$Message)
    Write-Host "[OK] $Message" -ForegroundColor Green
}

function Write-WarnLine {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Test-CommandExists {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Test-DockerDaemon {
    try {
        docker info *> $null
        return $true
    } catch {
        return $false
    }
}

function Test-LocalImage {
    param([string]$ImageRef)

    & docker image inspect $ImageRef *> $null
    return $LASTEXITCODE -eq 0
}

function Wait-Until {
    param(
        [scriptblock]$Condition,
        [int]$TimeoutSec,
        [int]$SleepSec,
        [string]$Description
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        if (& $Condition) {
            return $true
        }
        Start-Sleep -Seconds $SleepSec
    }

    throw "Timed out waiting for $Description after $TimeoutSec seconds."
}

function Ensure-DockerDaemon {
    if (-not (Test-CommandExists 'docker')) {
        throw "Docker CLI is not installed or not in PATH."
    }

    if (Test-DockerDaemon) {
        Write-Ok "Docker daemon is available."
        return
    }

    $dockerDesktopPath = 'C:\Program Files\Docker\Docker\Docker Desktop.exe'
    if (-not (Test-Path $dockerDesktopPath)) {
        throw "Docker daemon is not running, and Docker Desktop was not found at '$dockerDesktopPath'."
    }

    Write-Step "Docker daemon is not running. Starting Docker Desktop..."
    Start-Process -FilePath $dockerDesktopPath | Out-Null

    $null = Wait-Until `
        -Condition { Test-DockerDaemon } `
        -TimeoutSec $DockerStartupTimeoutSec `
        -SleepSec 3 `
        -Description 'Docker daemon'

    Write-Ok "Docker daemon is ready."
}

function Start-ComposeApp {
    $composeArgs = @('compose', 'up', '-d')
    if (-not $SkipBuild) {
        $composeArgs += '--build'
    }

    Write-Step "Starting development test environment with Docker Compose..."
    & docker @composeArgs
    if ($LASTEXITCODE -eq 0) {
        return
    }

    if (-not $SkipBuild -and (Test-LocalImage 'calciumion/new-api:latest')) {
        Write-WarnLine "Build failed, but a local image exists. Falling back to 'docker compose up -d' without rebuild."
        & docker compose up -d
        if ($LASTEXITCODE -eq 0) {
            return
        }
    }

    throw "docker $($composeArgs -join ' ') failed with exit code $LASTEXITCODE."
}

function Wait-AppReady {
    Write-Step "Waiting for http://localhost:3000/api/status ..."

    $null = Wait-Until `
        -Condition {
            try {
                $response = Invoke-WebRequest -UseBasicParsing 'http://localhost:3000/api/status' -TimeoutSec 10
                return $response.StatusCode -eq 200 -and $response.Content -match '"success"\s*:\s*true'
            } catch {
                return $false
            }
        } `
        -TimeoutSec $AppStartupTimeoutSec `
        -SleepSec 5 `
        -Description 'new-api application'

    Write-Ok "Application is healthy."
}

try {
    Write-Step "Preparing repository: $PSScriptRoot"
    Ensure-DockerDaemon
    Start-ComposeApp
    Wait-AppReady

    Write-Host ''
    Write-Ok "Development test environment is running."
    Write-Host "URL: http://localhost:3000" -ForegroundColor Green
    Write-Host "View logs: docker compose logs -f new-api"
    Write-Host "Stop env:  docker compose down"
} catch {
    Write-Host ''
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    Write-WarnLine "If startup failed, inspect logs with: docker compose logs --tail 200"
    exit 1
}
