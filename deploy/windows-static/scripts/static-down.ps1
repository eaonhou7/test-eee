#Requires -Version 5.1
# 低内存静态部署关闭脚本，只处理 static-runtime 下的 Go 后端进程。
[CmdletBinding()]
param(
  [int]$ApiPort = $(if ($env:API_PORT) { [int]$env:API_PORT } else { 9999 })
)

# 严格模式让脚本遇到错误时立即失败。
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# 定位项目根目录和静态部署 PID 目录。
$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$PidDir = Join-Path $RootDir "tmp\static-runtime\pids"

# 统一输出静态部署关闭日志前缀。
function Write-StaticLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[static-down.ps1] $Message"
}

# 根据 PID 文件停止脚本启动的 Go 后端进程。
function Stop-PidFileProcess {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][string]$PidFile
  )
  if (-not (Test-Path -LiteralPath $PidFile)) {
    return
  }

  $pidText = (Get-Content -LiteralPath $PidFile -Raw).Trim()
  if ($pidText -match "^\d+$") {
    $process = Get-Process -Id ([int]$pidText) -ErrorAction SilentlyContinue
    if ($process) {
      Write-StaticLog "stopping $Name (PID $pidText)"
      Stop-Process -Id ([int]$pidText) -Force -ErrorAction SilentlyContinue
    }
  }
  Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

# PID 文件缺失时兜底清理监听 API_PORT 的进程。
function Stop-PortListener {
  param([Parameter(Mandatory = $true)][int]$Port)
  if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) {
    return
  }

  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  $pids = @($connections | Select-Object -ExpandProperty OwningProcess -Unique)
  foreach ($pidValue in $pids) {
    $process = Get-Process -Id $pidValue -ErrorAction SilentlyContinue
    if ($process) {
      Write-StaticLog "stopping listener on port $Port (PID $pidValue)"
      Stop-Process -Id $pidValue -Force -ErrorAction SilentlyContinue
    }
  }
}

# 先按 PID 文件关闭，再按端口做兜底清理。
Stop-PidFileProcess -Name "static server" -PidFile (Join-Path $PidDir "server.pid")
Start-Sleep -Milliseconds 300
Stop-PortListener -Port $ApiPort
