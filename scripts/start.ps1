[CmdletBinding()]
param(
    [switch]$NoBuild
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $projectRoot

function Get-DotEnvValue {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][string]$DefaultValue
    )

    $environmentValue = [Environment]::GetEnvironmentVariable($Name)
    if (-not [string]::IsNullOrWhiteSpace($environmentValue)) {
        return $environmentValue
    }

    $entry = Get-Content -LiteralPath ".env" |
        Where-Object { $_ -match "^$([Regex]::Escape($Name))=(.*)$" } |
        Select-Object -Last 1

    if ($null -eq $entry) {
        return $DefaultValue
    }

    return ($entry -split "=", 2)[1].Trim()
}

function New-JwtSecret {
    $bytes = New-Object byte[] 32
    $generator = [Security.Cryptography.RandomNumberGenerator]::Create()
    try {
        $generator.GetBytes($bytes)
    }
    finally {
        $generator.Dispose()
    }

    return -join ($bytes | ForEach-Object { $_.ToString("x2") })
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "未找到 Docker。请先安装并启动 Docker Desktop。"
}

docker compose version | Out-Null
if ($LASTEXITCODE -ne 0) {
    throw "Docker Compose 不可用，请确认 Docker Desktop 已正常启动。"
}

if (-not (Test-Path -LiteralPath ".env")) {
    if (-not (Test-Path -LiteralPath ".env.example")) {
        throw "缺少 .env.example，无法生成运行配置。"
    }

    $secret = New-JwtSecret
    $content = Get-Content -LiteralPath ".env.example" -Raw
    $content = $content -replace "(?m)^JWT_SECRET=.*$", "JWT_SECRET=$secret"
    [IO.File]::WriteAllText((Join-Path $projectRoot ".env"), $content, (New-Object Text.UTF8Encoding($false)))
    Write-Host "已根据 .env.example 创建 .env，并生成随机 JWT 密钥。" -ForegroundColor Green
}

$apiPort = Get-DotEnvValue -Name "API_PORT" -DefaultValue "8080"
$webPort = Get-DotEnvValue -Name "WEB_PORT" -DefaultValue "3000"
$apiHealthUrl = "http://localhost:$apiPort/healthz"
$webUrl = "http://localhost:$webPort"

$composeArgs = @("compose", "up", "-d")
if (-not $NoBuild) {
    $composeArgs += "--build"
}

Write-Host "正在启动 StudyTrack..." -ForegroundColor Cyan
& docker @composeArgs
if ($LASTEXITCODE -ne 0) {
    docker compose ps
    throw "Docker Compose 启动失败。如果端口被占用，请修改 .env 中的 POSTGRES_PORT、API_PORT 或 WEB_PORT 后重试。"
}

Write-Host "正在等待 API 和 Web 就绪..." -ForegroundColor Cyan
$ready = $false
for ($attempt = 1; $attempt -le 60; $attempt++) {
    try {
        $apiResponse = Invoke-WebRequest -Uri $apiHealthUrl -UseBasicParsing -TimeoutSec 3
        $webResponse = Invoke-WebRequest -Uri $webUrl -UseBasicParsing -TimeoutSec 3
        if ($apiResponse.StatusCode -eq 200 -and $webResponse.StatusCode -eq 200) {
            $ready = $true
            break
        }
    }
    catch {
        # 服务仍在初始化，继续等待。
    }

    Start-Sleep -Seconds 2
}

if (-not $ready) {
    docker compose ps
    throw "服务未在 120 秒内就绪。请运行 'docker compose logs' 查看日志。"
}

Write-Host ""
Write-Host "StudyTrack 已启动" -ForegroundColor Green
Write-Host "Web：$webUrl"
Write-Host "API：http://localhost:$apiPort/api/v1"
Write-Host "健康检查：$apiHealthUrl"
Write-Host "停止服务：docker compose down"
