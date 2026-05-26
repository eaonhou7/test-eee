#Requires -Version 5.1
[CmdletBinding()]
param(
  [int]$ApiPort = $(if ($env:API_PORT) { [int]$env:API_PORT } else { 9999 }),
  [int]$WebPort = $(if ($env:WEB_PORT) { [int]$env:WEB_PORT } else { 8080 })
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$PidDir = Join-Path $RootDir "tmp\dev-runtime\pids"

function Write-DevLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[dev-down.ps1] $Message"
}

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
      Write-DevLog "stopping $Name (PID $pidText)"
      Stop-Process -Id ([int]$pidText) -Force -ErrorAction SilentlyContinue
    }
  }
  Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
}

function Stop-PortListener {
  param(
    [Parameter(Mandatory = $true)][string]$Name,
    [Parameter(Mandatory = $true)][int]$Port
  )
  if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) {
    return
  }

  $connections = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
  $pids = @($connections | Select-Object -ExpandProperty OwningProcess -Unique)
  foreach ($pidValue in $pids) {
    $process = Get-Process -Id $pidValue -ErrorAction SilentlyContinue
    if ($process) {
      Write-DevLog "stopping $Name listener on port $Port (PID $pidValue)"
      Stop-Process -Id $pidValue -Force -ErrorAction SilentlyContinue
    }
  }
}

Stop-PidFileProcess -Name "server" -PidFile (Join-Path $PidDir "server.pid")
Stop-PidFileProcess -Name "web" -PidFile (Join-Path $PidDir "web.pid")
Start-Sleep -Milliseconds 300
Stop-PortListener -Name "server" -Port $ApiPort
Stop-PortListener -Name "web" -Port $WebPort
