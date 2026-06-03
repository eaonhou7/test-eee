#Requires -Version 5.1
[CmdletBinding()]
param(
  [string]$Root = "C:\Users\Administrator\Desktop\eaon\system",
  [string]$AppName = "test-eee-git",
  [string]$MySqlFolderName = "",
  [string]$PrebuiltRelativePath = "deploy\windows-static",
  [int]$LogRetentionDays = 7,
  [int]$KeepMySqlBackups = 3,
  [switch]$KeepInstallers
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$App = Join-Path $Root $AppName
$DeploymentRoot = Join-Path $App $PrebuiltRelativePath
$script:RemovedFiles = 0
$script:RemovedDirs = 0

function Write-MaintLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[windows-disk-maintenance] $Message"
}

function Remove-ItemSafely {
  param([Parameter(Mandatory = $true)][string]$Path)
  try {
    if (Test-Path -LiteralPath $Path -PathType Container) {
      Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction Stop
      $script:RemovedDirs++
    } elseif (Test-Path -LiteralPath $Path -PathType Leaf) {
      Remove-Item -LiteralPath $Path -Force -ErrorAction Stop
      $script:RemovedFiles++
    }
    Write-MaintLog "removed $Path"
  } catch {
    Write-Warning "could not remove ${Path}: $($_.Exception.Message)"
  }
}

function Remove-OldFiles {
  param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][int]$RetentionDays,
    [string]$Filter = "*"
  )

  if (-not (Test-Path -LiteralPath $Path)) {
    return
  }

  $retention = [Math]::Max(0, $RetentionDays)
  $cutoff = (Get-Date).AddDays(-1 * $retention)
  Get-ChildItem -LiteralPath $Path -File -Recurse -Filter $Filter -ErrorAction SilentlyContinue |
    Where-Object { $_.LastWriteTime -lt $cutoff } |
    ForEach-Object { Remove-ItemSafely -Path $_.FullName }
}

function Find-MySqlHome {
  if ($MySqlFolderName) {
    $namedHome = Join-Path $Root $MySqlFolderName
    if (Test-Path -LiteralPath (Join-Path $namedHome "bin\mysqld.exe")) {
      return $namedHome
    }
  }

  $mysqlHome = Get-ChildItem -LiteralPath $Root -Directory -Filter "mysql-8.0.*-winx64" -ErrorAction SilentlyContinue |
    Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName "bin\mysqld.exe") } |
    Sort-Object LastWriteTime -Descending |
    Select-Object -First 1
  if ($mysqlHome) {
    return $mysqlHome.FullName
  }
  return ""
}

function Remove-OldMySqlBackups {
  $mysqlHome = Find-MySqlHome
  if (-not $mysqlHome) {
    return
  }

  $keepCount = [Math]::Max(0, $KeepMySqlBackups)
  $backups = @(Get-ChildItem -LiteralPath $mysqlHome -Directory -Filter "data.bak-*" -ErrorAction SilentlyContinue |
    Sort-Object LastWriteTime -Descending)

  if ($backups.Count -le $keepCount) {
    return
  }

  $backups | Select-Object -Skip $keepCount | ForEach-Object {
    Remove-ItemSafely -Path $_.FullName
  }
}

function Remove-Installers {
  if ($KeepInstallers) {
    Write-MaintLog "keeping installers because -KeepInstallers was specified"
    return
  }
  if (-not (Test-Path -LiteralPath $Root)) {
    return
  }

  $patterns = @(
    "Git-*-64-bit.exe",
    "Git-*.exe",
    "vc_redist.x64*.exe",
    "VC_redist.x64*.exe",
    "mysql-8.0.*-winx64.zip",
    "go-installer*.msi",
    "go*.windows-amd64.msi",
    "node-installer*.msi",
    "node-v*-x64.msi"
  )

  foreach ($pattern in $patterns) {
    Get-ChildItem -LiteralPath $Root -File -Filter $pattern -ErrorAction SilentlyContinue |
      ForEach-Object { Remove-ItemSafely -Path $_.FullName }
  }
}

function Invoke-GitMaintenance {
  if (-not (Test-Path -LiteralPath (Join-Path $App ".git"))) {
    return
  }

  $git = Get-Command git -ErrorAction SilentlyContinue
  if (-not $git) {
    Write-Warning "git was not found; skipping git gc"
    return
  }

  Write-MaintLog "running git gc --prune=now"
  & $git.Source -C $App gc --prune=now
  if ($LASTEXITCODE -ne 0) {
    Write-Warning "git gc exited with code $LASTEXITCODE"
  }
}

Write-MaintLog "cleaning logs older than $LogRetentionDays day(s)"
Remove-OldFiles -Path (Join-Path $DeploymentRoot "tmp\static-runtime\logs") -RetentionDays $LogRetentionDays
Remove-OldFiles -Path (Join-Path $DeploymentRoot "tmp\windows-lowmem-deploy") -RetentionDays $LogRetentionDays
Remove-OldFiles -Path (Join-Path $DeploymentRoot "server\log") -RetentionDays $LogRetentionDays

Remove-OldMySqlBackups
Remove-Installers
Invoke-GitMaintenance

Write-MaintLog "done. removed files=$($script:RemovedFiles) directories=$($script:RemovedDirs)"
