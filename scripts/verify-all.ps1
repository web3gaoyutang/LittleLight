# Windows PowerShell helper for local engineering verification.
param(
    [switch]$IncludeDockerLogic,
    [switch]$SkipH5Build
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$ServerDir = Join-Path $RepoRoot "server"
$AppDir = Join-Path $RepoRoot "app"
$ComposeFile = Join-Path $RepoRoot "deploy\docker-compose.yml"

$machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$env:Path = "$machinePath;$userPath;$env:Path"

function Invoke-Step {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Script
    )
    Write-Host ""
    Write-Host "==> $Name"
    & $Script
    Write-Host "ok: $Name"
}

function Assert-FileContains {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Pattern,
        [Parameter(Mandatory = $true)][string]$Message
    )
    $content = Get-Content -LiteralPath $Path -Raw -Encoding UTF8
    if ($content -notmatch $Pattern) {
        throw $Message
    }
}

function Resolve-Npm {
    $known = "D:\dev\node\npm.cmd"
    if (Test-Path -LiteralPath $known) {
        return $known
    }
    $command = Get-Command npm.cmd -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        return $command.Source
    }
    $command = Get-Command npm -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        return $command.Source
    }
    throw "npm was not found on PATH."
}

function Read-LocalEnv {
    $result = @{}
    $envFile = Join-Path $RepoRoot ".env"
    if (-not (Test-Path -LiteralPath $envFile)) {
        return $result
    }
    Get-Content -LiteralPath $envFile -Encoding UTF8 | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#") -or -not $line.Contains("=")) {
            return
        }
        $parts = $line.Split("=", 2)
        $result[$parts[0].Trim()] = $parts[1].Trim()
    }
    return $result
}

function Assert-LocalSecretsAreNotTracked {
    $localEnv = Read-LocalEnv
    $values = @()
    if ($localEnv.ContainsKey("LLM_API_KEY")) {
        $apiKey = [string]$localEnv["LLM_API_KEY"]
        if ($apiKey -match "^sk-.{8,}" -and $apiKey -ne "sk-xxx") {
            $values += @{ Name = "LLM_API_KEY"; Value = $apiKey }
        }
    }
    if ($localEnv.ContainsKey("LLM_BASE_URL")) {
        $baseUrl = [string]$localEnv["LLM_BASE_URL"]
        if ($baseUrl -ne "" -and $baseUrl -notmatch "example\.com") {
            $values += @{ Name = "LLM_BASE_URL"; Value = $baseUrl }
        }
    }
    foreach ($item in $values) {
        $null = & git -C $RepoRoot grep -n --fixed-strings -e $item.Value -- . 2>$null
        if ($LASTEXITCODE -eq 0) {
            throw "$($item.Name) from local .env appears in tracked files."
        }
        if ($LASTEXITCODE -gt 1) {
            throw "git grep failed while checking $($item.Name)."
        }
    }
}

Invoke-Step "OpenAPI contract shape" {
    python (Join-Path $RepoRoot "scripts\verify-docs.py")
}

Invoke-Step "H5 API environment shape" {
    $appEnv = Join-Path $RepoRoot "app\.env.example"
    Assert-FileContains -Path $appEnv -Pattern "(?m)^VITE_API_BASE_URL=/api/v1$" -Message "H5 default API base should stay same-origin for Docker nginx proxy."
    Assert-FileContains -Path $appEnv -Pattern "(?m)^VITE_DEV_API_TARGET=http://localhost:8080$" -Message "H5 dev API proxy target is missing."
}

Invoke-Step "Backend env example shape" {
    $envExample = Join-Path $RepoRoot ".env.example"
    foreach ($key in @("APP_ENV", "HTTP_ADDR", "DATABASE_URL", "REDIS_ADDR", "REDIS_DB", "MIGRATIONS_DIR", "AI_PROVIDER", "LLM_API_KEY", "LLM_BASE_URL", "LLM_MODEL")) {
        Assert-FileContains -Path $envExample -Pattern "(?m)^$key=" -Message "$key is missing from .env.example."
    }
}

Invoke-Step "Docker Compose config" {
    docker compose -f $ComposeFile config --quiet
}

Invoke-Step "Web gateway readiness config" {
    $nginxConfig = Join-Path $RepoRoot "deploy\nginx\default.conf"
    $composeConfig = Join-Path $RepoRoot "deploy\docker-compose.yml"
    Assert-FileContains -Path $nginxConfig -Pattern "location /readyz" -Message "nginx should proxy /readyz to the API service."
    Assert-FileContains -Path $composeConfig -Pattern "condition: service_healthy" -Message "web service should wait for API health before starting."
    Assert-FileContains -Path $composeConfig -Pattern "http://localhost/healthz" -Message "web service healthcheck should call the nginx health endpoint."
}

Invoke-Step "Deployment runbook coverage" {
    $runbook = Join-Path $RepoRoot "docs\deployment-runbook.md"
    foreach ($keyword in @("pg_dump", "FLUSHDB", "down migration", "docker-build", "readyz")) {
        Assert-FileContains -Path $runbook -Pattern $keyword -Message "deployment runbook should cover $keyword."
    }
}

Invoke-Step "Database schema document coverage" {
    $schemaDoc = Join-Path $RepoRoot "docs\database-schema.md"
    foreach ($keyword in @("users", "courses", "parent_profiles", "reminders", "communication_records", "healing_entries", "ai_generations", "favorites", "idx_courses_user_weekday", "002_seed.sql")) {
        Assert-FileContains -Path $schemaDoc -Pattern $keyword -Message "database schema document should cover $keyword."
    }
}

Invoke-Step "Go tests" {
    Push-Location $ServerDir
    try {
        go test ./...
    } finally {
        Pop-Location
    }
}

Invoke-Step "API client tests" {
    $npm = Resolve-Npm
    Push-Location $AppDir
    try {
        & $npm run test:api
    } finally {
        Pop-Location
    }
}

if (-not $SkipH5Build) {
    Invoke-Step "uni-app H5 build" {
        $npm = Resolve-Npm
        Push-Location $AppDir
        try {
            & $npm run build:h5
        } finally {
            Pop-Location
        }
    }
}

Invoke-Step "Local secret guard" {
    Assert-LocalSecretsAreNotTracked
}

if ($IncludeDockerLogic) {
    Invoke-Step "Docker logic verification" {
        & powershell -ExecutionPolicy Bypass -File (Join-Path $RepoRoot "scripts\verify-docker.ps1")
    }
}

Write-Host ""
Write-Host "All requested verification steps passed."
