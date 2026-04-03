param(
    [int]$DockerStartupTimeoutSec = 180,
    [int]$AppStartupTimeoutSec = 600
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Set-Location $PSScriptRoot

$testRoot = Join-Path $PSScriptRoot '.local-test'
$binaryPath = Join-Path $testRoot 'new-api'
$dockerfilePath = Join-Path $testRoot 'Dockerfile'
$imageRef = 'calciumion/new-api:latest'

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

function Ensure-LocalPrerequisites {
    if (-not (Test-CommandExists 'bun')) {
        throw "bun is not installed or not in PATH."
    }
    if (-not (Test-CommandExists 'go')) {
        throw "go is not installed or not in PATH."
    }

    $requiredImages = @(
        'calciumion/new-api:latest',
        'postgres:15',
        'redis:latest'
    )

    foreach ($requiredImage in $requiredImages) {
        if (-not (Test-LocalImage $requiredImage)) {
            throw "Required local image '$requiredImage' was not found. Run .\start_dev_env.ps1 once to prime local images before using local-only test mode."
        }
    }

    $nodeModulesPath = Join-Path $PSScriptRoot 'web\node_modules'
    if (-not (Test-Path $nodeModulesPath)) {
        throw "Local frontend dependencies were not found at '$nodeModulesPath'. Run 'bun install' once before using local-only test mode."
    }
}

function Get-ServerGoArch {
    $serverArchRaw = & docker version --format '{{.Server.Arch}}'
    if (-not $serverArchRaw) {
        $serverArchRaw = & docker info --format '{{.Architecture}}'
    }
    $serverArch = "$serverArchRaw".Trim().ToLowerInvariant()
    if ([string]::IsNullOrWhiteSpace($serverArch)) {
        Write-WarnLine "Unable to detect Docker server arch, defaulting GOARCH=amd64."
        return 'amd64'
    }
    switch ($serverArch) {
        'amd64' { return 'amd64' }
        'x86_64' { return 'amd64' }
        'arm64' { return 'arm64' }
        'aarch64' { return 'arm64' }
        default {
            Write-WarnLine "Unknown Docker server arch '$serverArch', defaulting GOARCH=amd64."
            return 'amd64'
        }
    }
}

function Build-FrontendLocally {
    Write-Step "Building frontend from local source with bun..."
    Push-Location (Join-Path $PSScriptRoot 'web')
    try {
        & bun run build
        if ($LASTEXITCODE -ne 0) {
            throw "bun run build failed with exit code $LASTEXITCODE."
        }
    } finally {
        Pop-Location
    }
    Write-Ok "Frontend build completed."
}

function Build-BackendLocally {
    $goArch = Get-ServerGoArch
    Write-Step "Building backend binary from local source for linux/$goArch ..."

    New-Item -ItemType Directory -Force -Path $testRoot | Out-Null
    if (Test-Path $binaryPath) {
        Remove-Item -Force $binaryPath
    }

    $versionRaw = Get-Content (Join-Path $PSScriptRoot 'VERSION') -Raw -ErrorAction SilentlyContinue
    $version = "$versionRaw".Trim()
    if ([string]::IsNullOrWhiteSpace($version)) {
        $version = 'dev-local-test'
    }

    Push-Location $PSScriptRoot
    try {
        $env:CGO_ENABLED = '0'
        $env:GOOS = 'linux'
        $env:GOARCH = $goArch
        $env:GOPROXY = 'off'
        $env:GOSUMDB = 'off'
        $env:GOFLAGS = '-mod=readonly'
        & go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$version'" -o $binaryPath
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed with exit code $LASTEXITCODE."
        }
    } finally {
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Remove-Item Env:GOOS -ErrorAction SilentlyContinue
        Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
        Remove-Item Env:GOPROXY -ErrorAction SilentlyContinue
        Remove-Item Env:GOSUMDB -ErrorAction SilentlyContinue
        Remove-Item Env:GOFLAGS -ErrorAction SilentlyContinue
        Pop-Location
    }

    @"
FROM calciumion/new-api:latest
COPY new-api /new-api
"@ | Set-Content -Path $dockerfilePath -Encoding ASCII

    Write-Ok "Backend binary prepared."
}

function Build-LocalRuntimeImage {
    Write-Step "Repacking local runtime image without pulling remote layers..."
    Push-Location $testRoot
    try {
        & docker build --load --network none -t $imageRef -f Dockerfile .
        if ($LASTEXITCODE -ne 0) {
            throw "docker build failed with exit code $LASTEXITCODE."
        }
    } finally {
        Pop-Location
    }
    Write-Ok "Local runtime image refreshed."
}

function Start-ComposeApp {
    Write-Step "Starting containers from local images only..."
    & docker compose up -d --no-build
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose up -d --no-build failed with exit code $LASTEXITCODE."
    }
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
    Write-Step "Preparing local-only test build: $PSScriptRoot"
    Ensure-DockerDaemon
    Ensure-LocalPrerequisites
    Build-FrontendLocally
    Build-BackendLocally
    Build-LocalRuntimeImage
    Start-ComposeApp
    Wait-AppReady

    Write-Host ''
    Write-Ok "Local-only test environment is running."
    Write-Host "URL: http://localhost:3000" -ForegroundColor Green
    Write-Host "View logs: docker compose logs -f new-api"
    Write-Host "Stop env:  docker compose down"
    Write-Host "Note: this script rebuilds only local source, reuses local Docker images, disables Go module network fetches, and avoids Docker build network access. If local caches or images are missing, it will fail instead of pulling from the internet."
} catch {
    Write-Host ''
    Write-Host "[ERROR] $($_.Exception.Message)" -ForegroundColor Red
    Write-WarnLine "If startup failed, inspect logs with: docker compose logs --tail 200"
    exit 1
}
