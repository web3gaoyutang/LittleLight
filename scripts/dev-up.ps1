# Windows PowerShell helper
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

docker compose -f deploy/docker-compose.yml up --build
