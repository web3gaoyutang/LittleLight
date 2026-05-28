# Windows PowerShell helper for local Docker logic verification.
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$ComposeFile = Join-Path $RepoRoot "deploy\docker-compose.yml"
$Api = $env:LITTLELIGHT_API_URL
if (-not $Api) { $Api = "http://localhost:8080" }
$HttpAddr = ":" + ([Uri]$Api).Port
$UserID = "00000000-0000-0000-0000-000000000001"
$Day = "2026-05-27"
$RunID = [Guid]::NewGuid().ToString("N").Substring(0, 8)
$ApiJob = $null

function Import-LocalEnv {
    $envFile = Join-Path $RepoRoot ".env"
    if (-not (Test-Path -LiteralPath $envFile)) {
        return
    }
    Get-Content -LiteralPath $envFile | ForEach-Object {
        $line = $_.Trim()
        if ($line -eq "" -or $line.StartsWith("#") -or -not $line.Contains("=")) {
            return
        }
        $parts = $line.Split("=", 2)
        [Environment]::SetEnvironmentVariable($parts[0].Trim(), $parts[1].Trim(), "Process")
    }
}

function Invoke-ApiJson {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Path,
        [object]$Body = $null
    )
    $headers = @{ "X-User-ID" = $UserID }
    $params = @{
        Method = $Method
        Uri = "$Api$Path"
        Headers = $headers
        TimeoutSec = 20
    }
    if ($null -ne $Body) {
        $params.ContentType = "application/json; charset=utf-8"
        $params.Body = ($Body | ConvertTo-Json -Depth 12)
    }
    Invoke-RestMethod @params
}

function Assert-True {
    param(
        [Parameter(Mandatory = $true)][bool]$Condition,
        [Parameter(Mandatory = $true)][string]$Message
    )
    if (-not $Condition) {
        throw $Message
    }
}

function Wait-Health {
    for ($i = 0; $i -lt 60; $i++) {
        try {
            $health = Invoke-RestMethod -Uri "$Api/healthz" -TimeoutSec 3
            if ($health.status -eq "ok") {
                return
            }
        } catch {
            Start-Sleep -Seconds 2
        }
    }
    throw "API health check did not pass: $Api/healthz"
}

Push-Location $RepoRoot
try {
    Import-LocalEnv
    docker compose -f $ComposeFile up -d postgres redis
    $toolPath = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
    $apiJob = Start-Job -Name "littlelight-api-verify" -ScriptBlock {
        param($Root, $PathValue, $Addr, $LLMKey, $LLMBaseURL, $LLMModel)
        Set-StrictMode -Version Latest
        $ErrorActionPreference = "Stop"
        $env:Path = $PathValue
        $env:APP_ENV = "docker"
        $env:HTTP_ADDR = $Addr
        $env:DATABASE_URL = "postgres://littlelight:littlelight@localhost:5432/littlelight?sslmode=disable"
        $env:REDIS_ADDR = "localhost:6379"
        $env:REDIS_PASSWORD = ""
        $env:REDIS_DB = "0"
        $env:MIGRATIONS_DIR = "migrations"
        $env:AI_PROVIDER = ""
        $env:LLM_API_KEY = $LLMKey
        $env:LLM_BASE_URL = $LLMBaseURL
        $env:LLM_MODEL = $LLMModel
        Set-Location (Join-Path $Root "server")
        go run ./cmd/api
    } -ArgumentList $RepoRoot, $toolPath, $HttpAddr, $env:LLM_API_KEY, $env:LLM_BASE_URL, $env:LLM_MODEL

    Wait-Health

    $login = Invoke-ApiJson -Method "POST" -Path "/api/v1/auth/wechat/mock" -Body @{
        code = "verify-$RunID"
        nickName = "Docker Verify Teacher"
    }
    Assert-True ([string]$login.userId -ne "") "Wechat mock login user id is empty."
    Assert-True ([string]$login.sessionToken -ne "") "Wechat mock login token is empty."

    $dashboard = Invoke-ApiJson -Method "GET" -Path "/api/v1/dashboard?day=$Day"
    Assert-True ($dashboard.todayLabel -eq $Day) "Dashboard day mismatch."

    $course = Invoke-ApiJson -Method "POST" -Path "/api/v1/courses?day=$Day" -Body @{
        title = "Docker verify course $RunID"
        className = "Class 3"
        location = "Building B402"
        weekday = 3
        startTime = "09:30"
        endTime = "10:15"
        note = "Local Docker logic verification"
    }
    Assert-True ([string]$course.id -ne "") "Course id is empty."
    $courseLookup = Invoke-ApiJson -Method "GET" -Path "/api/v1/courses/$($course.id)"
    Assert-True ($courseLookup.title -eq $course.title) "Course lookup failed."

    $reminder = Invoke-ApiJson -Method "POST" -Path "/api/v1/reminders?day=$Day" -Body @{
        title = "Docker verify reminder $RunID"
        category = "todo"
        remindAt = "2026-05-27T17:20:00+08:00"
        note = "Verify reminder write and query"
    }
    Assert-True ([string]$reminder.id -ne "") "Reminder id is empty."
    $reminderDone = Invoke-ApiJson -Method "POST" -Path "/api/v1/reminders/$($reminder.id)/complete"
    Assert-True ($reminderDone.ok -eq $true) "Reminder complete failed."

    $parent = Invoke-ApiJson -Method "POST" -Path "/api/v1/parents?day=$Day" -Body @{
        studentName = "Student $RunID"
        className = "Class 5"
        parentName = "Parent $RunID"
        relationship = "mother"
        contact = "13800000000"
        communicationStyle = "empathetic first"
        riskLevel = "medium"
        importantNotes = "Local Docker verification focus"
        nextAction = "Share class feedback tomorrow afternoon"
    }
    Assert-True ([string]$parent.id -ne "") "Parent id is empty."
    $parentLookup = Invoke-ApiJson -Method "GET" -Path "/api/v1/parents/$($parent.id)"
    Assert-True ($parentLookup.studentName -eq $parent.studentName) "Parent lookup failed."

    $record = Invoke-ApiJson -Method "POST" -Path "/api/v1/communication-records" -Body @{
        parentId = $parent.id
        student = $parent.studentName
        channel = "wechat"
        reason = "Docker verification"
        summary = "Share today's observation and next plan"
        result = "Follow up tomorrow"
        riskLevel = "medium"
        followUpAt = "2026-05-28T17:20:00+08:00"
    }
    Assert-True ([string]$record.id -ne "") "Communication record id is empty."

    $drafts = Invoke-ApiJson -Method "POST" -Path "/api/v1/ai/parent-drafts" -Body @{
        issue = "Student has been less focused in class recently"
        parentStyle = "anxious"
        tone = "warm"
        studentName = $parent.studentName
    }
    Assert-True ($drafts.Count -gt 0) "AI parent drafts are empty."

    $healing = Invoke-ApiJson -Method "POST" -Path "/api/v1/healing/entries" -Body @{
        type = "praise"
        mood = "warm"
        content = "Finished local Docker verification today"
        aiReply = "You checked the infrastructure carefully."
    }
    Assert-True ([string]$healing.id -ne "") "Healing entry id is empty."

    $favorite = Invoke-ApiJson -Method "POST" -Path "/api/v1/me/favorites" -Body @{
        type = "ai_praise_phrase"
        title = "Docker verify favorite $RunID"
        content = "You completed a reliable verification loop."
    }
    Assert-True ([string]$favorite.id -ne "") "Favorite id is empty."

    $dashboardAfter = Invoke-ApiJson -Method "GET" -Path "/api/v1/dashboard?day=$Day"
    Assert-True ($dashboardAfter.coursesCount -ge 1) "Dashboard did not include courses."
    Assert-True ($dashboardAfter.followUpsCount -ge 1) "Dashboard did not include parents."

    $courseCount = docker compose -f $ComposeFile exec -T postgres psql -U littlelight -d littlelight -tAc "SELECT count(*) FROM courses WHERE title = '$($course.title)';"
    Assert-True (($courseCount.Trim()) -eq "1") "PostgreSQL course row was not found."

    $redisKey = "dashboard:${UserID}:$Day"
    $redisExists = docker compose -f $ComposeFile exec -T redis redis-cli EXISTS $redisKey
    Assert-True (($redisExists.Trim()) -eq "1") "Redis dashboard cache key was not found."

    Write-Output "Docker logic verification passed. RunId=$RunID"
} finally {
    if ($null -ne $ApiJob) {
        Stop-Job -Job $ApiJob -ErrorAction SilentlyContinue
        Receive-Job -Job $ApiJob -ErrorAction SilentlyContinue | Select-Object -Last 80
        Remove-Job -Job $ApiJob -Force -ErrorAction SilentlyContinue
    }
    Pop-Location
}
