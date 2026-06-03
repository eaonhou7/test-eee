#Requires -Version 5.1
# 低内存静态部署参数：优先读取环境变量，便于 2C2G 机器用 STATIC_BUILD/SERVER_BUILD 跳过构建。
[CmdletBinding()]
param(
  [int]$ApiPort = $(if ($env:API_PORT) { [int]$env:API_PORT } else { 9999 }),
  [string]$MySqlHost = $(if ($env:MYSQL_HOST) { $env:MYSQL_HOST } else { "127.0.0.1" }),
  [int]$MySqlPort = $(if ($env:MYSQL_PORT) { [int]$env:MYSQL_PORT } else { 3306 }),
  [string]$MySqlUser = $(if ($env:MYSQL_USER) { $env:MYSQL_USER } else { "root" }),
  [string]$MySqlPassword = $env:MYSQL_PASSWORD,
  [string]$MySqlDatabase = $(if ($env:MYSQL_DATABASE) { $env:MYSQL_DATABASE } else { "amazon_admin" }),
  [string]$StaticBuild = $(if ($env:STATIC_BUILD) { $env:STATIC_BUILD } else { "1" }),
  [string]$ServerBuild = $(if ($env:SERVER_BUILD) { $env:SERVER_BUILD } else { "1" }),
  [string]$NodeOptions = $(if ($env:NODE_OPTIONS) { $env:NODE_OPTIONS } else { "--max-old-space-size=1536" }),
  [int]$GoBuildParallelism = $(if ($env:GO_BUILD_P) { [int]$env:GO_BUILD_P } else { 1 }),
  [string]$AdminPassword = $(if ($env:ADMIN_PASSWORD) { $env:ADMIN_PASSWORD } else { "123456" }),
  [int]$ServerReadyTimeout = $(if ($env:SERVER_READY_TIMEOUT) { [int]$env:SERVER_READY_TIMEOUT } else { 120 })
)

# 严格模式让脚本遇到未定义变量或命令错误时立即失败。
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# 项目根目录和静态部署运行目录，所有运行产物放到 tmp\static-runtime。
$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$RuntimeDir = Join-Path $RootDir "tmp\static-runtime"
$LogDir = Join-Path $RuntimeDir "logs"
$PidDir = Join-Path $RuntimeDir "pids"
$BinDir = Join-Path $RuntimeDir "bin"
$GoCacheDir = Join-Path $RuntimeDir "go-cache"
$GoModCacheDir = Join-Path $RuntimeDir "go-mod"
$ServerDir = Join-Path $RootDir "server"
$WebDir = Join-Path $RootDir "web"
$WebDist = Join-Path $WebDir "dist"
$ServerBin = Join-Path $BinDir "gva-server.exe"
$WebDepsStamp = Join-Path $WebDir "node_modules\.static-deps-stamp"
$ConfigTemplate = Join-Path $ServerDir "config.local.example.yaml"
$ConfigFile = Join-Path $ServerDir "config.local.yaml"

# 确保日志、PID、二进制和 Go 缓存目录存在。
New-Item -ItemType Directory -Force -Path $LogDir, $PidDir, $BinDir, $GoCacheDir, $GoModCacheDir | Out-Null

# 统一输出静态部署脚本日志前缀。
function Write-StaticLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[static-up.ps1] $Message"
}

# 失败时输出错误并终止脚本。
function Fail {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Error "[static-up.ps1] $Message"
  exit 1
}

# 兼容 0/false/no/off/skip 等关闭开关写法。
function Test-FlagEnabled {
  param([AllowEmptyString()][string]$Value)
  $normalized = "$Value".Trim().ToLowerInvariant()
  return -not ($normalized -in @("0", "false", "no", "off", "skip"))
}

# 查找必需命令，兼容 .exe/.cmd 和无后缀命令名。
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

# 生成 PowerShell 字符串字面量，避免路径里有空格时 runner 脚本解析失败。
function ConvertTo-PSLiteral {
  param([AllowEmptyString()][Parameter(Mandatory = $true)][string]$Value)
  return "'" + ($Value -replace "'", "''") + "'"
}

# 执行外部命令并返回真实 exit code，避免 mysqladmin 的 stderr warning 中断脚本。
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

# 从 config.local.yaml 读取简单的 section/key 值，用于自动获取 MySQL 密码。
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

# 判断 PID 文件记录的进程是否仍在运行。
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

# 清理已经失效的 PID 文件，避免误判服务已启动。
function Remove-StalePid {
  param([Parameter(Mandatory = $true)][string]$PidFile)
  if ((Test-Path -LiteralPath $PidFile) -and -not (Test-PidFileRunning -PidFile $PidFile)) {
    Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
  }
}

# 服务启动失败时输出最近日志，方便定位 MySQL、端口或配置问题。
function Show-LogExcerpt {
  param([Parameter(Mandatory = $true)][string]$LogFile)
  if (Test-Path -LiteralPath $LogFile) {
    Write-Warning "[static-up.ps1] recent log from ${LogFile}:"
    Get-Content -LiteralPath $LogFile -Tail 40 | ForEach-Object { Write-Warning $_ }
  }
}

# 用 TCP 连接快速判断端口是否被占用。
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

# 输出端口占用摘要，用于报错信息。
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

# 等待某个检查条件成功，超时后统一报错。
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

# 等待 Go 后端健康检查和前端静态首页都可用。
function Wait-ForStaticReady {
  param(
    [Parameter(Mandatory = $true)][string]$PidFile,
    [Parameter(Mandatory = $true)][string]$LogFile
  )
  for ($index = 1; $index -le $ServerReadyTimeout; $index++) {
    if ((Test-ServerHealth) -and (Test-WebHealth)) {
      return
    }
    if ((Test-Path -LiteralPath $PidFile) -and -not (Test-PidFileRunning -PidFile $PidFile)) {
      Show-LogExcerpt -LogFile $LogFile
      Fail "server exited before becoming ready"
    }
    Start-Sleep -Seconds 1
  }
  Show-LogExcerpt -LogFile $LogFile
  Fail "server did not become ready on port $ApiPort"
}

# 如果本地配置不存在，复制模板生成 server\config.local.yaml。
function Ensure-LocalConfig {
  if (-not (Test-Path -LiteralPath $ConfigFile)) {
    Copy-Item -LiteralPath $ConfigTemplate -Destination $ConfigFile
    Write-StaticLog "Created server\config.local.yaml from template; edit MySQL settings if needed"
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

# 检查配置里是否启用了 Redis；低内存静态模式默认不迁移/启动 Redis。
function Test-ConfigUsesRedis {
  $sourceFile = if (Test-Path -LiteralPath $ConfigFile) { $ConfigFile } else { $ConfigTemplate }
  foreach ($line in Get-Content -LiteralPath $sourceFile) {
    if ($line -match "^\s*use-redis:\s*([^#\s]+)") {
      return ($matches[1].Trim().Trim('"').Trim("'").ToLowerInvariant() -eq "true")
    }
  }
  return $false
}

# 使用 mysqladmin ping 验证 MySQL 是否可连接。
function Test-MySqlReady {
  $arguments = @("--protocol=TCP", "--host=$MySqlHost", "--port=$MySqlPort", "--user=$MySqlUser")
  if (-not [string]::IsNullOrEmpty($MySqlPassword)) {
    $arguments += "--password=$MySqlPassword"
  }
  $arguments += "ping"
  $exitCode = Invoke-ExternalCommand -Path $script:MySqlAdminCommand -Arguments $arguments -Quiet
  return ($exitCode -eq 0)
}

# 如果本机存在 MySQL Windows 服务，尝试启动它。
function Start-MySqlServicesIfPresent {
  $services = Get-Service -Name "MySQL*" -ErrorAction SilentlyContinue
  foreach ($service in $services) {
    if ($service.Status -ne "Running") {
      Write-StaticLog "Starting Windows service: $($service.Name)"
      Start-Service -Name $service.Name -ErrorAction SilentlyContinue
    }
  }
}

# 等待 MySQL 可用，失败时提示检查密码和配置。
function Ensure-MySqlReady {
  if (Test-MySqlReady) {
    return
  }
  Start-MySqlServicesIfPresent
  Wait-ForCondition -Message "mysql did not become ready; check MYSQL_PASSWORD or server\config.local.yaml" -Attempts 30 -Probe { Test-MySqlReady }
}

# 静态部署目标是低内存运行；如果配置打开 Redis，要求用户明确处理。
function Ensure-RedisNotRequired {
  if (-not (Test-ConfigUsesRedis)) {
    Write-StaticLog "Skipping Redis because local config has use-redis: false"
    return
  }
  Fail "server\config.local.yaml has use-redis: true. Disable Redis for low-memory deployment or start Redis manually before this script."
}

# 判断源文件是否比目标文件更新，用于决定是否重装依赖或重建二进制。
function Test-PathNewerThan {
  param(
    [Parameter(Mandatory = $true)][string]$Source,
    [Parameter(Mandatory = $true)][string]$Target
  )
  if (-not (Test-Path -LiteralPath $Source)) {
    return $false
  }
  if (-not (Test-Path -LiteralPath $Target)) {
    return $true
  }
  return ((Get-Item -LiteralPath $Source).LastWriteTimeUtc -gt (Get-Item -LiteralPath $Target).LastWriteTimeUtc)
}

# 按 package.json/package-lock.json 变更判断是否需要重新安装前端依赖。
function Ensure-WebDependencies {
  $nodeModules = Join-Path $WebDir "node_modules"
  $packageJson = Join-Path $WebDir "package.json"
  $packageLock = Join-Path $WebDir "package-lock.json"
  if (
    (Test-Path -LiteralPath $nodeModules) -and
    (Test-Path -LiteralPath $WebDepsStamp) -and
    -not (Test-PathNewerThan -Source $packageJson -Target $WebDepsStamp) -and
    -not (Test-PathNewerThan -Source $packageLock -Target $WebDepsStamp)
  ) {
    return
  }

  $script:NpmCommand = Resolve-RequiredCommand -Names @("npm.cmd", "npm")
  Write-StaticLog "Installing web dependencies"
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

# 构建或复用 web\dist；2C2G 机器建议设置 STATIC_BUILD=0 跳过前端构建。
function Build-StaticWeb {
  $indexHtml = Join-Path $WebDist "index.html"
  if (-not (Test-FlagEnabled -Value $StaticBuild)) {
    if (-not (Test-Path -LiteralPath $indexHtml)) {
      Fail "web\dist\index.html not found; run npm run build:static first"
    }
    Write-StaticLog "Reusing existing web\dist"
    return
  }

  Ensure-WebDependencies
  $script:NpmCommand = Resolve-RequiredCommand -Names @("npm.cmd", "npm")
  Write-StaticLog "Building static web assets"
  $previousNodeOptions = $env:NODE_OPTIONS
  Push-Location $WebDir
  try {
    # 默认限制 Node heap，减少前端构建把 2G 机器内存打满的概率。
    if ([string]::IsNullOrWhiteSpace($previousNodeOptions) -and -not [string]::IsNullOrWhiteSpace($NodeOptions)) {
      $env:NODE_OPTIONS = $NodeOptions
    }
    & $script:NpmCommand run build:static
    if ($LASTEXITCODE -ne 0) {
      Fail "npm run build:static failed"
    }
  } finally {
    $env:NODE_OPTIONS = $previousNodeOptions
    Pop-Location
  }
}

# 判断 Go 源码、go.mod 或 go.sum 是否比已有二进制更新。
function Test-ServerSourcesChanged {
  if (-not (Test-Path -LiteralPath $ServerBin)) {
    return $true
  }
  if ((Test-PathNewerThan -Source (Join-Path $ServerDir "go.mod") -Target $ServerBin) -or
      (Test-PathNewerThan -Source (Join-Path $ServerDir "go.sum") -Target $ServerBin)) {
    return $true
  }
  $binTime = (Get-Item -LiteralPath $ServerBin).LastWriteTimeUtc
  $newerSource = Get-ChildItem -LiteralPath $ServerDir -Recurse -Filter "*.go" -File |
    Where-Object { $_.LastWriteTimeUtc -gt $binTime } |
    Select-Object -First 1
  return ($null -ne $newerSource)
}

# 构建或复用 Go 后端二进制；默认 go build -p 1 降低并发和峰值内存。
function Build-ServerBinary {
  if ((-not (Test-FlagEnabled -Value $ServerBuild)) -and (Test-Path -LiteralPath $ServerBin)) {
    Write-StaticLog "Reusing existing server binary"
    return
  }
  if ((Test-FlagEnabled -Value $ServerBuild) -and -not (Test-ServerSourcesChanged)) {
    Write-StaticLog "Reusing cached server binary"
    return
  }

  $script:GoCommand = Resolve-RequiredCommand -Names @("go.exe", "go")
  Write-StaticLog "Building server binary with go build -p $GoBuildParallelism"
  $previousGoCache = $env:GOCACHE
  $previousGoModCache = $env:GOMODCACHE
  Push-Location $ServerDir
  try {
    $env:GOCACHE = $GoCacheDir
    $env:GOMODCACHE = $GoModCacheDir
    & $script:GoCommand build -p $GoBuildParallelism -o $ServerBin .
    if ($LASTEXITCODE -ne 0) {
      Fail "go build failed"
    }
  } finally {
    $env:GOCACHE = $previousGoCache
    $env:GOMODCACHE = $previousGoModCache
    Pop-Location
  }
}

# 后端健康检查，确认 API 服务已经可用。
function Test-ServerHealth {
  try {
    $response = Invoke-RestMethod -Uri "http://127.0.0.1:$ApiPort/health" -TimeoutSec 5
    return ($response -eq "ok")
  } catch {
    return $false
  }
}

# 前端首页检查，确认 Go 已经托管 web\dist。
function Test-WebHealth {
  try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:$ApiPort/" -UseBasicParsing -TimeoutSec 5
    return ($response.StatusCode -ge 200 -and $response.Content -match "<html|<!doctype html")
  } catch {
    return $false
  }
}

# 创建隐藏 PowerShell 后台 runner，并把 runner 进程 PID 写入文件。
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

# 启动单进程静态部署：Go 同时提供后端 API 和前端静态资源。
function Start-StaticServer {
  $pidFile = Join-Path $PidDir "server.pid"
  $logFile = Join-Path $LogDir "server.log"
  $indexHtml = Join-Path $WebDist "index.html"
  Remove-StalePid -PidFile $pidFile

  if (Test-PidFileRunning -PidFile $pidFile) {
    Write-StaticLog "Server already running (PID $((Get-Content -LiteralPath $pidFile -Raw).Trim()))"
    return
  }
  if (Test-PortBusy -Port $ApiPort) {
    if ((Test-ServerHealth) -and (Test-WebHealth)) {
      Write-StaticLog "Static deployment already reachable on port $ApiPort ($(Get-PortListenerSummary -Port $ApiPort))"
      return
    }
    Fail "port $ApiPort is already in use by another process: $(Get-PortListenerSummary -Port $ApiPort)"
  }
  if (-not (Test-Path -LiteralPath $ServerBin)) {
    Fail "server binary not found at $ServerBin"
  }
  if (-not (Test-Path -LiteralPath $indexHtml)) {
    Fail "web\dist\index.html not found"
  }

  Write-StaticLog "Starting server with static web root"
  Set-Content -LiteralPath $logFile -Value "" -Encoding UTF8
  # 注入 GVA_STATIC_ROOT，让后端注册 web\dist 静态路由。
  Start-Runner -Name "static-server" -PidFile $pidFile -Lines @(
    '$ErrorActionPreference = "Stop"',
    '$env:GVA_CONFIG = "config.local.yaml"',
    ('$env:GVA_STATIC_ROOT = ' + (ConvertTo-PSLiteral $WebDist)),
    "Set-Location -LiteralPath $(ConvertTo-PSLiteral $ServerDir)",
    "& $(ConvertTo-PSLiteral $ServerBin) *>> $(ConvertTo-PSLiteral $logFile)"
  )

  Wait-ForStaticReady -PidFile $pidFile -LogFile $logFile
}

# 调用初始化检查接口，判断数据库是否已经初始化。
function Get-InitCheckResponse {
  Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:$ApiPort/init/checkdb" -TimeoutSec 10
}

# 首次启动时自动初始化 GVA 数据库和管理员账号。
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

  Write-StaticLog "Initializing GVA database"
  Invoke-RestMethod `
    -Method Post `
    -Uri "http://127.0.0.1:$ApiPort/init/initdb" `
    -ContentType "application/json" `
    -Body $payload `
    -TimeoutSec 120 | Out-Null
}

# 如数据库未初始化，则执行初始化并等待后端恢复。
function Ensure-ServerInitialized {
  $checkResponse = Get-InitCheckResponse
  if ($checkResponse.data.needInit) {
    Initialize-Database
    Wait-ForCondition -Message "server did not recover after initdb" -Attempts $ServerReadyTimeout -Probe { Test-ServerHealth }
  }
}

# 输出访问地址、日志位置和停止命令。
function Write-RuntimeSummary {
  Write-Host ""
  Write-Host "Static deployment started:"
  Write-Host "  Login: http://127.0.0.1:$ApiPort/#/login"
  Write-Host "  Backend: http://127.0.0.1:$ApiPort"
  Write-Host "  Health: http://127.0.0.1:$ApiPort/health"
  Write-Host "  Logs: $LogDir"
  Write-Host ""
  Write-Host "Runtime processes:"
  Write-Host "  Go backend + MySQL only. Vite/Node dev server is not started."
  Write-Host ""
  Write-Host "Stop:"
  Write-Host "  .\scripts\static-down.ps1"
}

# 主流程：准备配置、确认 MySQL、构建/复用资源、启动服务、自动初始化数据库。
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

$script:MySqlAdminCommand = Resolve-RequiredCommand -Names @("mysqladmin.exe", "mysqladmin")
$script:NpmCommand = $null
$script:GoCommand = $null

Ensure-MySqlReady
Ensure-RedisNotRequired
Build-StaticWeb
Build-ServerBinary
Start-StaticServer
Ensure-ServerInitialized
Write-RuntimeSummary
