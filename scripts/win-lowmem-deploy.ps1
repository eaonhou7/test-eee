#Requires -Version 5.1
#Requires -RunAsAdministrator

# Windows 低内存一键部署脚本：适配 C:\Users\Administrator\Desktop\eaon\system 这种离线目录。
[CmdletBinding()]
param(
  # 离线部署根目录，里面应包含 test-eee-git、mysql-8.0.46-winx64、PortableGit 等目录。
  [string]$Root = "C:\Users\Administrator\Desktop\eaon\system",
  # 项目目录名，默认匹配用户 Windows 桌面截图里的 test-eee-git。
  [string]$AppName = "test-eee-git",
  # MySQL zip 解压后的目录名。
  [string]$MySqlFolderName = "mysql-8.0.46-winx64",
  # 本机 MySQL root 密码，首次初始化会设置成这个值。
  [string]$MySqlRootPassword = "123456a",
  # GVA 初始化接口使用的后台 admin 密码。
  [string]$AdminPassword = "123456",
  # MySQL 监听端口，默认 3306。
  [int]$MySqlPort = 3306,
  # Microsoft 官方 VC++ x64 运行库下载地址；有网时脚本会自动下载。
  [string]$VcRedistUrl = "https://aka.ms/vc14/vc_redist.x64.exe",
  # 备份并重新初始化 MySQL data 目录；仅首次部署或确认不要旧数据时使用。
  [switch]$ResetMySqlData,
  # 跳过 git pull，适合目标机完全离线时使用。
  [switch]$SkipGitPull
)

# 严格模式让路径、命令或变量错误尽早暴露。
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# 计算项目、MySQL 和数据目录路径。
$App = Join-Path $Root $AppName
$MySqlHome = Join-Path $Root $MySqlFolderName
$MySqlData = Join-Path $MySqlHome "data"
$DeployLogDir = Join-Path $App "tmp\windows-lowmem-deploy"
$MySqlLogDir = Join-Path $DeployLogDir "mysql"

# 统一部署日志前缀。
function Write-DeployLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[win-lowmem-deploy] $Message"
}

# 统一失败出口，便于用户定位卡在哪一步。
function Fail {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Error "[win-lowmem-deploy] $Message"
  exit 1
}

# 输出日志尾部，避免 MySQL 启动失败时只看到泛化错误。
function Show-FileTail {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [int]$Tail = 80
  )
  if (Test-Path -LiteralPath $Path) {
    Write-Warning "[win-lowmem-deploy] recent log from ${Path}:"
    Get-Content -LiteralPath $Path -Tail $Tail | ForEach-Object { Write-Warning $_ }
  }
}

# 判断端口是否已经有监听进程。
function Test-PortBusy {
  param([Parameter(Mandatory = $true)][int]$Port)
  $client = New-Object System.Net.Sockets.TcpClient
  try {
    $async = $client.BeginConnect("127.0.0.1", $Port, $null, $null)
    return [bool]($async.AsyncWaitHandle.WaitOne(300, $false) -and $client.Connected)
  } catch {
    return $false
  } finally {
    $client.Close()
  }
}

# 输出端口监听进程摘要，便于判断是否被已有 MySQL 或其他服务占用。
function Get-PortListenerSummary {
  param([Parameter(Mandatory = $true)][int]$Port)
  if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) {
    return "port $Port is in use; run netstat -ano | findstr :$Port"
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

# 把离线目录里的工具路径临时加入当前 PowerShell 的 PATH。
function Add-PathEntry {
  param([Parameter(Mandatory = $true)][string]$PathValue)
  if (Test-Path -LiteralPath $PathValue) {
    $env:Path = "$PathValue;$env:Path"
  }
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

# 执行版本命令，既验证工具可用，也把版本打印给操作者。
function Show-CommandVersion {
  param(
    [Parameter(Mandatory = $true)][string[]]$Names,
    [Parameter(Mandatory = $true)][string[]]$Arguments,
    [switch]$AllowFailure
  )
  $commandPath = Resolve-RequiredCommand -Names $Names
  $previousErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    $output = & $commandPath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
  } catch {
    $output = @($_)
    $exitCode = 1
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
  if ($output) {
    $output | ForEach-Object { Write-Host $_ }
  }
  if ($exitCode -ne 0) {
    if ($AllowFailure) {
      Write-Warning "version check failed: $($Names -join ' or '); continuing and validating later with runtime checks"
      return
    }
    Fail "version check failed: $($Names -join ' or ')"
  }
}

# 执行指定 exe 的版本命令；MySQL zip 版优先使用固定路径，避免 PATH 命中错误程序。
function Show-ExecutableVersion {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string[]]$Arguments,
    [switch]$AllowFailure
  )
  $previousErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    $output = & $Path @Arguments 2>&1
    $exitCode = $LASTEXITCODE
  } catch {
    $output = @($_)
    $exitCode = 1
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
  if ($output) {
    $output | ForEach-Object { Write-Host $_ }
  }
  if ($exitCode -ne 0) {
    if ($AllowFailure) {
      Write-Warning "version check failed: $Path; continuing and validating later with runtime checks"
      return
    }
    Fail "version check failed: $Path"
  }
}

# 执行外部命令并返回真实 exit code，避免 PowerShell 把 native stderr warning 当成脚本失败。
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

# 如果 Go/Node 缺失，则从离线根目录里的 MSI 安装包静默安装。
function Install-MsiWhenCommandMissing {
  param(
    [Parameter(Mandatory = $true)][string]$CommandName,
    [Parameter(Mandatory = $true)][string]$InstallerFilter,
    [Parameter(Mandatory = $true)][string]$InstallPath
  )

  if (Get-Command $CommandName -ErrorAction SilentlyContinue) {
    return
  }

  $installer = Get-ChildItem -LiteralPath $Root -Filter $InstallerFilter -File -ErrorAction SilentlyContinue |
    Select-Object -First 1
  if (-not $installer) {
    Fail "$CommandName not found and installer $InstallerFilter was not found under $Root"
  }

  Write-DeployLog "$CommandName was missing; installing $($installer.FullName)"
  Start-Process -FilePath "msiexec.exe" -Wait -ArgumentList "/i `"$($installer.FullName)`" /qn /norestart"
  $env:Path = "$InstallPath;$env:Path"

  if (-not (Get-Command $CommandName -ErrorAction SilentlyContinue)) {
    Fail "$CommandName is still unavailable after installing $($installer.Name)"
  }
}

# 检查 MySQL 8 zip 版依赖的 VC++ x64 运行库是否存在。
function Test-VcRuntimePresent {
  $system32 = Join-Path $env:WINDIR "System32"
  foreach ($dllName in @("VCRUNTIME140.dll", "VCRUNTIME140_1.dll", "MSVCP140.dll")) {
    if (-not (Test-Path -LiteralPath (Join-Path $system32 $dllName))) {
      return $false
    }
  }
  return $true
}

# 查找离线根目录里的 VC++ x64 运行库安装包。
function Find-VcRuntimeInstaller {
  $installer = Get-ChildItem -LiteralPath $Root -Filter "vc_redist.x64*.exe" -File -ErrorAction SilentlyContinue |
    Select-Object -First 1
  if ($installer) {
    return $installer
  }
  return Get-ChildItem -LiteralPath $Root -Filter "VC_redist.x64*.exe" -File -ErrorAction SilentlyContinue |
    Select-Object -First 1
}

# 有网时自动下载 Microsoft 官方 VC++ x64 运行库安装包到离线根目录。
function Download-VcRuntimeInstaller {
  $downloadPath = Join-Path $Root "vc_redist.x64.exe"
  Write-DeployLog "downloading Microsoft Visual C++ Redistributable x64 to $downloadPath"
  try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -Uri $VcRedistUrl -OutFile $downloadPath -UseBasicParsing
  } catch {
    Write-Warning "download failed from ${VcRedistUrl}: $($_.Exception.Message)"
    return $null
  }

  if ((Test-Path -LiteralPath $downloadPath) -and ((Get-Item -LiteralPath $downloadPath).Length -gt 1048576)) {
    return Get-Item -LiteralPath $downloadPath
  }

  Write-Warning "downloaded vc_redist.x64.exe is missing or too small"
  return $null
}

# 缺少 VC++ 运行库时，优先本地安装；没有本地安装包则尝试官方下载。
function Ensure-VcRuntime {
  if (Test-VcRuntimePresent) {
    return
  }

  $installer = Find-VcRuntimeInstaller
  if (-not $installer) {
    $installer = Download-VcRuntimeInstaller
  }
  if (-not $installer) {
    Fail "missing Microsoft Visual C++ Redistributable x64. Download vc_redist.x64.exe from $VcRedistUrl, put it under $Root, and rerun this script."
  }

  Write-DeployLog "installing Microsoft Visual C++ Redistributable x64: $($installer.FullName)"
  $process = Start-Process -FilePath $installer.FullName -Wait -PassThru -ArgumentList "/install /quiet /norestart"
  if (($process.ExitCode -ne 0) -and ($process.ExitCode -ne 3010)) {
    Write-Warning "VC++ runtime installer exited with code $($process.ExitCode); checking DLLs anyway"
  }

  if (-not (Test-VcRuntimePresent)) {
    Fail "VC++ runtime is still missing after installing $($installer.Name); reboot Windows or install Microsoft Visual C++ Redistributable x64 manually."
  }
}

# 用单引号生成 SQL 字符串字面量，避免密码里出现单引号时 SQL 语法错误。
function ConvertTo-SqlLiteral {
  param([AllowEmptyString()][Parameter(Mandatory = $true)][string]$Value)
  return "'" + ($Value -replace "'", "''") + "'"
}

# 测试 MySQL 是否能用指定认证方式 ping 通。
function Test-MySqlPing {
  param([switch]$UsePassword)
  foreach ($hostName in @("127.0.0.1", "localhost")) {
    $arguments = @("--protocol=TCP", "--host=$hostName", "--port=$MySqlPort", "--user=root")
    if ($UsePassword) {
      $arguments += "--password=$MySqlRootPassword"
    }
    $arguments += "ping"
    $exitCode = Invoke-ExternalCommand -Path $script:MySqlAdminExe -Arguments $arguments -Quiet
    if ($exitCode -eq 0) {
      return $true
    }
  }
  return $false
}

# 执行 root SQL，兼容新 data 目录只允许 root@localhost 的场景。
function Invoke-MySqlRootSql {
  param(
    [Parameter(Mandatory = $true)][string]$Sql,
    [switch]$UsePassword
  )
  foreach ($hostName in @("127.0.0.1", "localhost")) {
    $arguments = @("--protocol=TCP", "--host=$hostName", "--port=$MySqlPort", "--user=root")
    if ($UsePassword) {
      $arguments += "--password=$MySqlRootPassword"
    }
    $arguments += @("--execute=$Sql")
    $exitCode = Invoke-ExternalCommand -Path $script:MySqlExe -Arguments $arguments
    if ($exitCode -eq 0) {
      return $true
    }
    Write-Warning "mysql command failed on $hostName with exit code $exitCode; trying next host if available"
  }
  return $false
}

# 等待 MySQL 进入可连接状态，并区分是否已经设置 root 密码。
function Wait-ForMySqlAuthMode {
  param([int]$Attempts = 30)
  for ($index = 1; $index -le $Attempts; $index++) {
    if (Test-MySqlPing -UsePassword) {
      return "password"
    }
    if (Test-MySqlPing) {
      return "nopassword"
    }
    Start-Sleep -Seconds 2
  }
  return ""
}

# 首次使用 zip 版 MySQL 时初始化 data 目录。
function Test-MySqlDataInitialized {
  return (
    (Test-Path -LiteralPath $MySqlData) -and
    (Test-Path -LiteralPath (Join-Path $MySqlData "mysql")) -and
    (Test-Path -LiteralPath (Join-Path $MySqlData "auto.cnf"))
  )
}

# 备份已有 data 目录，避免直接删除用户数据。
function Backup-MySqlDataDir {
  if (-not (Test-Path -LiteralPath $MySqlData)) {
    return
  }
  $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
  $backupPath = Join-Path $MySqlHome "data.bak-$timestamp"
  Write-DeployLog "backing up MySQL data directory to $backupPath"
  Move-Item -LiteralPath $MySqlData -Destination $backupPath
}

function Initialize-MySqlDataIfNeeded {
  if ($ResetMySqlData -and (Test-Path -LiteralPath $MySqlData)) {
    Backup-MySqlDataDir
  }

  if (Test-MySqlDataInitialized) {
    return
  }

  if (Test-Path -LiteralPath $MySqlData) {
    Write-Warning "MySQL data directory exists but does not look initialized; backing it up before reinitializing"
    Backup-MySqlDataDir
  }

  Write-DeployLog "initializing MySQL data directory"
  $initOutLog = Join-Path $MySqlLogDir "initialize.out.log"
  $initErrLog = Join-Path $MySqlLogDir "initialize.err.log"
  Set-Content -LiteralPath $initOutLog -Value "" -Encoding UTF8
  Set-Content -LiteralPath $initErrLog -Value "" -Encoding UTF8

  $process = Start-Process -FilePath $script:MySqlDaemonExe `
    -ArgumentList @(
      "--no-defaults",
      "--initialize-insecure",
      "--basedir=$MySqlHome",
      "--datadir=$MySqlData",
      "--console"
    ) `
    -RedirectStandardOutput $initOutLog `
    -RedirectStandardError $initErrLog `
    -Wait `
    -PassThru

  if ($process.ExitCode -ne 0) {
    Show-FileTail -Path $initErrLog
    Show-FileTail -Path $initOutLog
    Fail "mysqld --initialize-insecure failed"
  }
}

# 启动 zip 版 MySQL，并等待 127.0.0.1:3306 可连接。
function Start-PortableMySqlIfNeeded {
  $authMode = Wait-ForMySqlAuthMode -Attempts 3
  if ($authMode) {
    return $authMode
  }

  if (Test-PortBusy -Port $MySqlPort) {
    Fail "port $MySqlPort is already in use by $(Get-PortListenerSummary -Port $MySqlPort). Stop that process, or rerun with -MySqlPort another port."
  }

  Write-DeployLog "starting portable MySQL"
  $mysqlOutLog = Join-Path $MySqlLogDir "mysqld.out.log"
  $mysqlErrLog = Join-Path $MySqlLogDir "mysqld.err.log"
  Set-Content -LiteralPath $mysqlOutLog -Value "" -Encoding UTF8
  Set-Content -LiteralPath $mysqlErrLog -Value "" -Encoding UTF8

  $mysqlProcess = Start-Process -FilePath $script:MySqlDaemonExe -ArgumentList @(
    "--no-defaults",
    "--basedir=$MySqlHome",
    "--datadir=$MySqlData",
    "--port=$MySqlPort",
    "--bind-address=127.0.0.1",
    "--character-set-server=utf8mb4",
    "--collation-server=utf8mb4_unicode_ci",
    "--console"
  ) `
    -RedirectStandardOutput $mysqlOutLog `
    -RedirectStandardError $mysqlErrLog `
    -WindowStyle Hidden `
    -PassThru

  for ($index = 1; $index -le 45; $index++) {
    if (Test-MySqlPing -UsePassword) {
      return "password"
    }
    if (Test-MySqlPing) {
      return "nopassword"
    }
    if ($mysqlProcess.HasExited) {
      Show-FileTail -Path $mysqlErrLog
      Show-FileTail -Path $mysqlOutLog
      Fail "mysqld.exe exited before becoming ready; see logs under $MySqlLogDir"
    }
    Start-Sleep -Seconds 2
  }

  Show-FileTail -Path $mysqlErrLog
  Show-FileTail -Path $mysqlOutLog
  Fail "MySQL did not become ready on 127.0.0.1:$MySqlPort; see logs under $MySqlLogDir"
}

# 确保 root 密码正确，并补齐后端配置需要的 root@127.0.0.1 权限。
function Ensure-MySqlRootPassword {
  param([Parameter(Mandatory = $true)][string]$AuthMode)
  $passwordSql = ConvertTo-SqlLiteral -Value $MySqlRootPassword
  $sql = @"
CREATE USER IF NOT EXISTS 'root'@'localhost' IDENTIFIED BY $passwordSql;
ALTER USER 'root'@'localhost' IDENTIFIED BY $passwordSql;
CREATE USER IF NOT EXISTS 'root'@'127.0.0.1' IDENTIFIED BY $passwordSql;
ALTER USER 'root'@'127.0.0.1' IDENTIFIED BY $passwordSql;
GRANT ALL PRIVILEGES ON *.* TO 'root'@'127.0.0.1' WITH GRANT OPTION;
FLUSH PRIVILEGES;
"@

  if ($AuthMode -eq "password") {
    Write-DeployLog "ensuring MySQL root password and 127.0.0.1 privileges"
    if (-not (Invoke-MySqlRootSql -Sql $sql -UsePassword)) {
      Fail "failed to verify MySQL root password and privileges"
    }
    return
  }

  Write-DeployLog "setting MySQL root password and 127.0.0.1 privileges"
  if (-not (Invoke-MySqlRootSql -Sql $sql)) {
    Fail "failed to set MySQL root password"
  }

  if (-not (Test-MySqlPing -UsePassword)) {
    Fail "MySQL root password was set, but password login still failed"
  }
}

# 创建业务数据库 amazon_admin。
function Ensure-ApplicationDatabase {
  Write-DeployLog "creating database amazon_admin if needed"
  $sql = "CREATE DATABASE IF NOT EXISTS amazon_admin DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_unicode_ci;"
  if (-not (Invoke-MySqlRootSql -Sql $sql -UsePassword)) {
    Fail "failed to create database amazon_admin"
  }
}

# 生成并修补 server\config.local.yaml，只改 MySQL 块里部署必需的字段。
function Update-LocalConfig {
  $configTemplate = Join-Path $App "server\config.local.example.yaml"
  $configFile = Join-Path $App "server\config.local.yaml"
  if (-not (Test-Path -LiteralPath $configTemplate)) {
    Fail "missing config template: $configTemplate"
  }

  New-Item -ItemType Directory -Force -Path $DeployLogDir | Out-Null
  if (Test-Path -LiteralPath $configFile) {
    $timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
    Copy-Item -LiteralPath $configFile -Destination (Join-Path $DeployLogDir "config.local.yaml.bak-$timestamp") -Force
  }

  Copy-Item -LiteralPath $configTemplate -Destination $configFile -Force

  $lines = Get-Content -LiteralPath $configFile
  $inMySql = $false
  $patched = foreach ($line in $lines) {
    if ($line -match "^mysql:\s*$") {
      $inMySql = $true
      $line
      continue
    }
    if ($inMySql -and $line -match "^[^\s#].*:\s*$") {
      $inMySql = $false
    }

    if ($inMySql) {
      if ($line -match "^\s*path:") { "    path: 127.0.0.1"; continue }
      if ($line -match "^\s*port:") { "    port: `"$MySqlPort`""; continue }
      if ($line -match "^\s*config:") { "    config: charset=utf8mb4&parseTime=True&loc=Local"; continue }
      if ($line -match "^\s*db-name:") { "    db-name: amazon_admin"; continue }
      if ($line -match "^\s*username:") { "    username: root"; continue }
      if ($line -match "^\s*password:") { "    password: $MySqlRootPassword"; continue }
      if ($line -match "^\s*log-mode:") { "    log-mode: error"; continue }
    }

    $line
  }
  Set-Content -LiteralPath $configFile -Value $patched -Encoding UTF8
}

# 可联网时更新代码；离线或 pull 失败时继续使用本地文件部署。
function Pull-ProjectIfPossible {
  if ($SkipGitPull) {
    Write-DeployLog "skipping git pull because -SkipGitPull was specified"
    return
  }
  if (-not (Test-Path -LiteralPath (Join-Path $App ".git"))) {
    return
  }

  $gitCommand = Get-Command git -ErrorAction SilentlyContinue
  if (-not $gitCommand) {
    Write-Warning "git command not found; continue with local files"
    return
  }

  Write-DeployLog "running git pull --ff-only"
  & $gitCommand.Source -C $App pull --ff-only
  if ($LASTEXITCODE -ne 0) {
    Write-Warning "git pull failed; continue with local files in $App"
  }
}

# 调用已有 static-up.ps1，真正以 Go 后端 + web\dist 静态文件方式运行。
function Start-StaticDeployment {
  $staticUp = Join-Path $App "scripts\static-up.ps1"
  if (-not (Test-Path -LiteralPath $staticUp)) {
    Fail "missing static startup script: $staticUp"
  }

  Set-Location -LiteralPath $App
  Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force

  $env:MYSQL_PASSWORD = $MySqlRootPassword
  $env:MYSQL_HOST = "127.0.0.1"
  $env:MYSQL_PORT = "$MySqlPort"
  $env:MYSQL_USER = "root"
  $env:MYSQL_DATABASE = "amazon_admin"
  $env:ADMIN_PASSWORD = $AdminPassword
  $env:STATIC_BUILD = "1"
  $env:SERVER_BUILD = "1"
  $env:NODE_OPTIONS = "--max-old-space-size=1536"
  $env:GO_BUILD_P = "1"

  Write-DeployLog "starting low-memory static deployment"
  & $staticUp
  if ($LASTEXITCODE -ne 0) {
    Fail "scripts\static-up.ps1 failed"
  }
}

# 主流程：校验目录、准备 PATH、初始化 MySQL、写配置并启动静态部署。
if (-not (Test-Path -LiteralPath $App)) {
  Fail "project directory not found: $App"
}
if (-not (Test-Path -LiteralPath $MySqlHome)) {
  Fail "MySQL directory not found: $MySqlHome"
}
New-Item -ItemType Directory -Force -Path $DeployLogDir, $MySqlLogDir | Out-Null

Add-PathEntry -PathValue (Join-Path $Root "PortableGit\cmd")
Add-PathEntry -PathValue (Join-Path $Root "PortableGit\bin")
Add-PathEntry -PathValue (Join-Path $MySqlHome "bin")
Add-PathEntry -PathValue "C:\Program Files\Go\bin"
Add-PathEntry -PathValue "C:\Program Files\nodejs"

Install-MsiWhenCommandMissing -CommandName "go" -InstallerFilter "go-installer*.msi" -InstallPath "C:\Program Files\Go\bin"
Install-MsiWhenCommandMissing -CommandName "node" -InstallerFilter "node-installer*.msi" -InstallPath "C:\Program Files\nodejs"
Ensure-VcRuntime

$script:MySqlExe = Join-Path $MySqlHome "bin\mysql.exe"
$script:MySqlAdminExe = Join-Path $MySqlHome "bin\mysqladmin.exe"
$script:MySqlDaemonExe = Join-Path $MySqlHome "bin\mysqld.exe"
foreach ($mysqlTool in @($script:MySqlExe, $script:MySqlAdminExe, $script:MySqlDaemonExe)) {
  if (-not (Test-Path -LiteralPath $mysqlTool)) {
    Fail "missing MySQL tool: $mysqlTool"
  }
}

Show-CommandVersion -Names @("git.exe", "git") -Arguments @("--version")
Show-CommandVersion -Names @("go.exe", "go") -Arguments @("version")
Show-CommandVersion -Names @("node.exe", "node") -Arguments @("-v")
Show-CommandVersion -Names @("npm.cmd", "npm") -Arguments @("-v")
Show-ExecutableVersion -Path $script:MySqlExe -Arguments @("--version") -AllowFailure

Pull-ProjectIfPossible
Initialize-MySqlDataIfNeeded
$authMode = Start-PortableMySqlIfNeeded
Ensure-MySqlRootPassword -AuthMode $authMode
Ensure-ApplicationDatabase
Update-LocalConfig
Start-StaticDeployment

Write-Host ""
Write-Host "Windows low-memory deployment is ready:"
Write-Host "  Login: http://127.0.0.1:8888/#/login"
Write-Host "  Username: admin"
Write-Host "  Password: $AdminPassword"
Write-Host ""
Write-Host "Stop:"
Write-Host "  cd $App"
Write-Host "  .\scripts\static-down.ps1"
