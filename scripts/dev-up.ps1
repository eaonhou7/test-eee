#Requires -Version 5.1
[CmdletBinding()]
param(
  [int]$ApiPort = $(if ($env:API_PORT) { [int]$env:API_PORT } else { 9999 }),
  [int]$WebPort = $(if ($env:WEB_PORT) { [int]$env:WEB_PORT } else { 8080 }),
  [string]$WebHost = $(if ($env:WEB_HOST) { $env:WEB_HOST } else { "127.0.0.1" }),
  [string]$MySqlHost = $(if ($env:MYSQL_HOST) { $env:MYSQL_HOST } else { "127.0.0.1" }),
  [int]$MySqlPort = $(if ($env:MYSQL_PORT) { [int]$env:MYSQL_PORT } else { 3306 }),
  [string]$MySqlUser = $(if ($env:MYSQL_USER) { $env:MYSQL_USER } else { "root" }),
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlDatabase = $(if ($env:MYSQL_DATABASE) { $env:MYSQL_DATABASE } else { "amazon_admin" }),
  [string]$RedisHost = $(if ($env:REDIS_HOST) { $env:REDIS_HOST } else { "127.0.0.1" }),
  [int]$RedisPort = $(if ($env:REDIS_PORT) { [int]$env:REDIS_PORT } else { 6379 }),
  [string]$AdminPassword = $(if ($env:ADMIN_PASSWORD) { $env:ADMIN_PASSWORD } else { "123456" }),
  [int]$ServerReadyTimeout = $(if ($env:SERVER_READY_TIMEOUT) { [int]$env:SERVER_READY_TIMEOUT } else { 120 }),
  [int]$WebReadyTimeout = $(if ($env:WEB_READY_TIMEOUT) { [int]$env:WEB_READY_TIMEOUT } else { 120 })
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RuntimeDir = Join-Path $RootDir "tmp\dev-runtime"
$LogDir = Join-Path $RuntimeDir "logs"
$PidDir = Join-Path $RuntimeDir "pids"
$BinDir = Join-Path $RuntimeDir "bin"
$GoCacheDir = Join-Path $RuntimeDir "go-cache"
$GoModCacheDir = Join-Path $RuntimeDir "go-mod"
$ServerDir = Join-Path $RootDir "server"
$WebDir = Join-Path $RootDir "web"
$ServerBin = Join-Path $BinDir "gva-server.exe"
$WebDepsStamp = Join-Path $WebDir "node_modules\.deps-stamp"
$ConfigTemplate = Join-Path $ServerDir "config.local.example.yaml"
$ConfigFile = Join-Path $ServerDir "config.local.yaml"

New-Item -ItemType Directory -Force -Path $LogDir, $PidDir, $BinDir, $GoCacheDir, $GoModCacheDir | Out-Null

function Write-DevLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[dev-up.ps1] $Message"
}

function Fail {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Error "[dev-up.ps1] $Message"
  exit 1
}

function Resolve-RequiredCommand {
  param([Parameter(Mandatory = $true)][string[]]$Names)
  foreach ($name in $Names) {
    $command = Get-Command $name -ErrorAction SilentlyContinue
    if ($command) {
      return $command.Source
    }
  }
  Fail "missing required command: $($Names -join ' or ')"
}

function ConvertTo-PSLiteral {
  param([AllowEmptyString()][Parameter(Mandatory = $true)][string]$Value)
  return "'" + ($Value -replace "'", "''") + "'"
}

function Invoke-ExternalCommand {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string[]]$Arguments,
    [switch]$Quiet
  )
  $previousErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    if ($Quiet) {
      & $Path @Arguments *> $null
    } else {
      & $Path @Arguments
    }
    $exitCode = $LASTEXITCODE
  } catch {
    $exitCode = $LASTEXITCODE
    if ($null -eq $exitCode) {
      $exitCode = 1
    }
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
  return [int]$exitCode
}

function Get-ConfigValue {
  param(
    [Parameter(Mandatory = $true)][string]$Section,
    [Parameter(Mandatory = $true)][string]$Key
  )
  if (-not (Test-Path -LiteralPath $ConfigFile)) {
    return ""
  }

  $inSection = $false
  $pattern = "^\s*$([regex]::Escape($Key))\s*:\s*(.*)$"
  foreach ($line in Get-Content -LiteralPath $ConfigFile) {
    if ($line -match "^[^\s#][^:]*:") {
      $currentSection = ($line -split ":", 2)[0].Trim()
      $inSection = ($currentSection -eq $Section)
      continue
    }
    if ($inSection -and $line -match $pattern) {
      $value = $matches[1] -replace "\s+#.*$", ""
      $value = $value.Trim().Trim('"').Trim("'")
      return $value
    }
  }
  return ""
}

function Test-PidFileRunning {
  param([Parameter(Mandatory = $true)][string]$PidFile)
  if (-not (Test-Path -LiteralPath $PidFile)) {
    return $false
  }
  $pidText = (Get-Content -LiteralPath $PidFile -Raw).Trim()
  if (-not ($pidText -match "^\d+$")) {
    return $false
  }
  return $null -ne (Get-Process -Id ([int]$pidText) -ErrorAction SilentlyContinue)
}

function Remove-StalePid {
  param([Parameter(Mandatory = $true)][string]$PidFile)
  if ((Test-Path -LiteralPath $PidFile) -and -not (Test-PidFileRunning -PidFile $PidFile)) {
    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
  }
}

function Show-LogExcerpt {
  param([Parameter(Mandatory = $true)][string]$LogFile)
  if (Test-Path -LiteralPath $LogFile) {
    Write-Warning "[dev-up.ps1] recent log from ${LogFile}:"
    Get-Content -LiteralPath $LogFile -Tail 40 | ForEach-Object { Write-Warning $_ }
  }
}

function Test-PortBusy {
  param([Parameter(Mandatory = $true)][int]$Port)
  $client = New-Object System.Net.Sockets.TcpClient
  try {
    $async = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
    $connected = $async.AsyncWaitHandle.WaitOne(300, $false) -and $client.Connected
    return [bool]$connected
  } catch {
    return $false
  } finally {
    $client.Close()
  }
}

function Get-PortListenerSummary {
  param([Parameter(Mandatory = $true)][int]$Port)
  if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) {
    return "port $Port is in use"
  }
  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  $summaries = @()
  foreach ($pidValue in ($connections | Select-Object -ExpandProperty OwningProcess -Unique)) {
    $process = Get-Process -Id $pidValue -ErrorAction SilentlyContinue
    if ($process) {
      $summaries += "$($process.ProcessName) pid=$pidValue"
    } else {
      $summaries += "pid=$pidValue"
    }
  }
  if ($summaries.Count -eq 0) {
    return "unknown listener on port $Port"
  }
  return ($summaries -join "; ")
}

function Wait-ForCondition {
  param(
    [Parameter(Mandatory = $true)][string]$Message,
    [Parameter(Mandatory = $true)][int]$Attempts,
    [Parameter(Mandatory = $true)][scriptblock]$Probe
  )
  for ($index = 1; $index -le $Attempts; $index++) {
    if (& $Probe) {
      return
    }
    Start-Sleep -Seconds 1
  }
  Fail $Message
}

function Wait-ForProcessReady {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][int]$Attempts,
    [Parameter(Mandatory = $true)][string]$PidFile,
    [Parameter(Mandatory = $true)][string]$LogFile,
    [Parameter(Mandatory = $true)][int]$Port,
    [Parameter(Mandatory = $true)][scriptblock]$Probe
  )
  for ($index = 1; $index -le $Attempts; $index++) {
    if (& $Probe) {
      return
    }
    if ((Test-Path -LiteralPath $PidFile) -and -not (Test-PidFileRunning -PidFile $PidFile)) {
      Show-LogExcerpt -LogFile $LogFile
      Fail "$Name exited before becoming ready"
    }
    Start-Sleep -Seconds 1
  }
  Show-LogExcerpt -LogFile $LogFile
  if (Test-PortBusy -Port $Port) {
    Fail "$Name did not become ready; current listener(s) on port ${Port}: $(Get-PortListenerSummary -Port $Port)"
  }
  Fail "$Name did not become ready"
}

function Ensure-LocalConfig {
  if (-not (Test-Path -LiteralPath $ConfigFile)) {
    Copy-Item -LiteralPath $ConfigTemplate -Destination $ConfigFile
  }
}

# 保持 config.local.yaml 的 system.addr 和脚本健康检查端口一致。
function Sync-ApiPortToLocalConfig {
  $lines = Get-Content -LiteralPath $ConfigFile
  $inSystem = $false
  $patched = foreach ($line in $lines) {
    if ($line -match "^system:\s*$") {
      $inSystem = $true
      $line
      continue
    }
    if ($inSystem -and $line -match "^[^\s#].*:\s*$") {
      $inSystem = $false
    }

    if ($inSystem -and $line -match "^\s*addr:") {
      "    addr: $ApiPort"
      continue
    }
    $line
  }
  Set-Content -LiteralPath $ConfigFile -Value $patched -Encoding UTF8
}

function Test-ConfigUsesRedis {
  $sourceFile = if (Test-Path -LiteralPath $ConfigFile) { $ConfigFile } else { $ConfigTemplate }
  foreach ($line in Get-Content -LiteralPath $sourceFile) {
    if ($line -match "^\s*use-redis:\s*([^#\s]+)") {
      return ($matches[1].Trim().Trim('"').Trim("'").ToLowerInvariant() -eq "true")
    }
  }
  return $false
}

function Test-MySqlReady {
  $arguments = @("--protocol=TCP", "--host=$MySqlHost", "--port=$MySqlPort", "--user=$MySqlUser")
  if (-not [string]::IsNullOrEmpty($MySqlPassword)) {
    $arguments += "--password=$MySqlPassword"
  }
  $arguments += "ping"
  $exitCode = Invoke-ExternalCommand -Path $script:MySqlAdminCommand -Arguments $arguments -Quiet
  return ($exitCode -eq 0)
}

function Start-MySqlServicesIfPresent {
  $services = Get-Service -Name "MySQL*" -ErrorAction SilentlyContinue
  foreach ($service in $services) {
    if ($service.Status -ne "Running") {
      Write-DevLog "Starting Windows service: $($service.Name)"
      Start-Service -Name $service.Name -ErrorAction SilentlyContinue
    }
  }
}

function Ensure-MySqlReady {
  if (Test-MySqlReady) {
    return
  }
  Start-MySqlServicesIfPresent
  Wait-ForCondition -Message "mysql did not become ready; check MYSQL_PASSWORD or server/config.local.yaml" -Attempts 30 -Probe { Test-MySqlReady }
}

function Test-RedisReady {
  & $script:RedisCliCommand -h $RedisHost -p $RedisPort ping 2>$null | Select-String -Quiet "^PONG$"
}

function Ensure-RedisReady {
  if (-not (Test-ConfigUsesRedis)) {
    Write-DevLog "Skipping Redis startup because local config has use-redis: false"
    return
  }
  $script:RedisCliCommand = Resolve-RequiredCommand -Names @("redis-cli.exe", "redis-cli")
  Wait-ForCondition -Message "redis did not become ready" -Attempts 30 -Probe { Test-RedisReady }
}

function Test-ServerSourcesChanged {
  if (-not (Test-Path -LiteralPath $ServerBin)) {
    return $true
  }
  $binTime = (Get-Item -LiteralPath $ServerBin).LastWriteTimeUtc
  foreach ($path in @("go.mod", "go.sum")) {
    $fullPath = Join-Path $ServerDir $path
    if ((Test-Path -LiteralPath $fullPath) -and ((Get-Item -LiteralPath $fullPath).LastWriteTimeUtc -gt $binTime)) {
      return $true
    }
  }
  $newerSource = Get-ChildItem -LiteralPath $ServerDir -Recurse -Filter "*.go" -File |
    Where-Object { $_.LastWriteTimeUtc -gt $binTime } |
    Select-Object -First 1
  return ($null -ne $newerSource)
}

function Build-ServerBinary {
  if (-not (Test-ServerSourcesChanged)) {
    Write-DevLog "Reusing cached server binary"
    return
  }

  Write-DevLog "Building server binary"
  $previousGoCache = $env:GOCACHE
  $previousGoModCache = $env:GOMODCACHE
  Push-Location $ServerDir
  try {
    $env:GOCACHE = $GoCacheDir
    $env:GOMODCACHE = $GoModCacheDir
    & $script:GoCommand build -o $ServerBin .
    if ($LASTEXITCODE -ne 0) {
      Fail "go build failed"
    }
  } finally {
    $env:GOCACHE = $previousGoCache
    $env:GOMODCACHE = $previousGoModCache
    Pop-Location
  }
}

function Ensure-WebDependencies {
  $nodeModules = Join-Path $WebDir "node_modules"
  $packageJson = Join-Path $WebDir "package.json"
  $packageLock = Join-Path $WebDir "package-lock.json"
  if (
    (Test-Path -LiteralPath $nodeModules) -and
    (Test-Path -LiteralPath $WebDepsStamp) -and
    ((Get-Item -LiteralPath $packageJson).LastWriteTimeUtc -le (Get-Item -LiteralPath $WebDepsStamp).LastWriteTimeUtc) -and
    ((Get-Item -LiteralPath $packageLock).LastWriteTimeUtc -le (Get-Item -LiteralPath $WebDepsStamp).LastWriteTimeUtc)
  ) {
    return
  }

  Write-DevLog "Installing web dependencies"
  Push-Location $WebDir
  try {
    & $script:NpmCommand install --prefer-offline --no-audit --no-fund
    if ($LASTEXITCODE -ne 0) {
      Fail "npm install failed"
    }
    New-Item -ItemType File -Force -Path $WebDepsStamp | Out-Null
  } finally {
    Pop-Location
  }
}

function Prepare-DevPrerequisites {
  Write-DevLog "Preparing web dependencies and server binary"
  Ensure-WebDependencies
  Build-ServerBinary
}

function Test-ServerHealth {
  try {
    $response = Invoke-RestMethod -Uri "http://127.0.0.1:$ApiPort/health" -TimeoutSec 5
    return ($response -eq "ok")
  } catch {
    return $false
  }
}

function Test-WebHealth {
  try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:$WebPort" -UseBasicParsing -TimeoutSec 5
    return ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500)
  } catch {
    return $false
  }
}

function Start-Runner {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$PidFile,
    [Parameter(Mandatory = $true)][string[]]$Lines
  )
  $runnerPath = Join-Path $RuntimeDir "$Name-runner.ps1"
  Set-Content -LiteralPath $runnerPath -Value $Lines -Encoding UTF8

  $shell = Get-Command pwsh.exe -ErrorAction SilentlyContinue
  if (-not $shell) {
    $shell = Get-Command powershell.exe -ErrorAction SilentlyContinue
  }
  if (-not $shell) {
    Fail "missing PowerShell executable for background process"
  }

  $process = Start-Process -FilePath $shell.Source `
    -ArgumentList @("-NoProfile", "-ExecutionPolicy", "Bypass", "-File", $runnerPath) `
    -WindowStyle Hidden `
    -PassThru
  Set-Content -LiteralPath $PidFile -Value "$($process.Id)"
}

function Start-Server {
  $pidFile = Join-Path $PidDir "server.pid"
  $logFile = Join-Path $LogDir "server.log"
  Remove-StalePid -PidFile $pidFile

  if (Test-PidFileRunning -PidFile $pidFile) {
    Write-DevLog "Server already running (PID $((Get-Content -LiteralPath $pidFile -Raw).Trim()))"
    return
  }
  if (Test-PortBusy -Port $ApiPort) {
    if (Test-ServerHealth) {
      Write-DevLog "Server already reachable on port $ApiPort ($(Get-PortListenerSummary -Port $ApiPort))"
      return
    }
    Fail "server port $ApiPort is already in use by another process: $(Get-PortListenerSummary -Port $ApiPort)"
  }
  if (-not (Test-Path -LiteralPath $ServerBin)) {
    Fail "server binary not found at $ServerBin; build step did not complete"
  }

  Write-DevLog "Starting gin-vue-admin server"
  Set-Content -LiteralPath $logFile -Value "" -Encoding UTF8
  Start-Runner -Name "server" -PidFile $pidFile -Lines @(
    '$ErrorActionPreference = "Stop"',
    '$env:GVA_CONFIG = "config.local.yaml"',
    "Set-Location -LiteralPath $(ConvertTo-PSLiteral $ServerDir)",
    "& $(ConvertTo-PSLiteral $ServerBin) *>> $(ConvertTo-PSLiteral $logFile)"
  )
}

function Get-InitCheckResponse {
  Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$ApiPort/init/checkdb" -TimeoutSec 10
}

function Initialize-Database {
  $payload = @{
    adminPassword = $AdminPassword
    dbType = "mysql"
    host = $MySqlHost
    port = "$MySqlPort"
    userName = $MySqlUser
    password = $MySqlPassword
    dbName = $MySqlDatabase
  } | ConvertTo-Json -Compress

  Write-DevLog "Initializing GVA database"
  Invoke-RestMethod `
    -Method Post `
    -Uri "http://127.0.0.1:$ApiPort/init/initdb" `
    -ContentType "application/json" `
    -Body $payload `
    -TimeoutSec 120 | Out-Null
}

function Ensure-ServerInitialized {
  $checkResponse = Get-InitCheckResponse
  if ($checkResponse.data.needInit) {
    Initialize-Database
    Wait-ForCondition -Message "server did not recover after initdb" -Attempts $ServerReadyTimeout -Probe { Test-ServerHealth }
  }
}

function Start-Web {
  $pidFile = Join-Path $PidDir "web.pid"
  $logFile = Join-Path $LogDir "web.log"
  $viteBin = Join-Path $WebDir "node_modules\.bin\vite.cmd"
  Remove-StalePid -PidFile $pidFile

  if (Test-PidFileRunning -PidFile $pidFile) {
    Write-DevLog "Web already running (PID $((Get-Content -LiteralPath $pidFile -Raw).Trim()))"
    return
  }
  if (Test-PortBusy -Port $WebPort) {
    if (Test-WebHealth) {
      Write-DevLog "Web already reachable on port $WebPort ($(Get-PortListenerSummary -Port $WebPort))"
      return
    }
    Fail "web port $WebPort is already in use by another process: $(Get-PortListenerSummary -Port $WebPort)"
  }
  if (-not (Test-Path -LiteralPath $viteBin)) {
    Fail "vite executable not found at $viteBin; run npm install in web/"
  }

  Write-DevLog "Starting web dev server"
  Set-Content -LiteralPath $logFile -Value "" -Encoding UTF8
  Start-Runner -Name "web" -PidFile $pidFile -Lines @(
    '$ErrorActionPreference = "Stop"',
    '$env:BROWSER = "none"',
    '$env:VITE_AUTO_OPEN = "false"',
    "Set-Location -LiteralPath $(ConvertTo-PSLiteral $WebDir)",
    "& $(ConvertTo-PSLiteral $viteBin) --host $(ConvertTo-PSLiteral $WebHost) --port $WebPort --mode development *>> $(ConvertTo-PSLiteral $logFile)"
  )
}

function Write-RuntimeSummary {
  Write-Host ""
  Write-Host "Project started:"
  Write-Host "  Frontend: http://127.0.0.1:$WebPort"
  Write-Host "  Login: http://127.0.0.1:$WebPort/#/login"
  Write-Host "  Logistics: http://127.0.0.1:$WebPort/#/amazon/logisticsQuote"
  Write-Host "  Backend: http://127.0.0.1:$ApiPort"
  Write-Host "  Health: http://127.0.0.1:$ApiPort/health"
  Write-Host ""
  Write-Host "Default account:"
  Write-Host "  Username: admin"
  Write-Host "  Password: $AdminPassword"
  Write-Host ""
  Write-Host "Logs:"
  Write-Host "  $LogDir"
}

$script:GoCommand = Resolve-RequiredCommand -Names @("go.exe", "go")
$script:NpmCommand = Resolve-RequiredCommand -Names @("npm.cmd", "npm")
$script:MySqlCommand = Resolve-RequiredCommand -Names @("mysql.exe", "mysql")
$script:MySqlAdminCommand = Resolve-RequiredCommand -Names @("mysqladmin.exe", "mysqladmin")
$script:RedisCliCommand = $null

Ensure-LocalConfig
Sync-ApiPortToLocalConfig
if ([string]::IsNullOrWhiteSpace($MySqlPassword)) {
  $configuredPassword = Get-ConfigValue -Section "mysql" -Key "password"
  if ([string]::IsNullOrWhiteSpace($configuredPassword)) {
    $MySqlPassword = "123456a"
  } else {
    $MySqlPassword = $configuredPassword
  }
}

Ensure-MySqlReady
Ensure-RedisReady
Prepare-DevPrerequisites
Start-Server
Start-Web
Wait-ForProcessReady -Name "server" -Attempts $ServerReadyTimeout -PidFile (Join-Path $PidDir "server.pid") -LogFile (Join-Path $LogDir "server.log") -Port $ApiPort -Probe { Test-ServerHealth }
Ensure-ServerInitialized
Wait-ForProcessReady -Name "web" -Attempts $WebReadyTimeout -PidFile (Join-Path $PidDir "web.pid") -LogFile (Join-Path $LogDir "web.log") -Port $WebPort -Probe { Test-WebHealth }
Write-RuntimeSummary
