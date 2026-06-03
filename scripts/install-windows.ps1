#Requires -Version 5.1
[CmdletBinding()]
param(
  [string]$Root = "C:\Users\Administrator\Desktop\eaon\system",
  [string]$AppName = "test-eee-git",
  [string]$GitRepoUrl = "https://github.com/eaonhou7/test-eee.git",
  [string]$GitBranch = "",
  [string]$MySqlRootPassword = "123456a",
  [string]$AdminPassword = "123456",
  [int]$ApiPort = 9999,
  [int]$MySqlPort = 3306,
  [switch]$BuildOnTarget,
  [string]$PrebuiltRelativePath = "deploy\windows-static",
  [switch]$Offline,
  [switch]$SkipGitPull,
  [switch]$ResetMySqlData
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$MinimumGitVersion = [version]"2.40.0"
$MinimumGoVersion = [version]"1.24.2"
$MinimumNodeVersion = [version]"20.0.0"
$MySqlVersion = "8.0.46"
$MySqlZipName = "mysql-$MySqlVersion-winx64.zip"
$MySqlZipUrl = "https://cdn.mysql.com/Downloads/MySQL-8.0/$MySqlZipName"
$VcRedistUrl = "https://aka.ms/vc14/vc_redist.x64.exe"
$GitForWindowsLatestReleaseApi = "https://api.github.com/repos/git-for-windows/git/releases/latest"
$ScriptProjectRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$script:UsePrebuiltStatic = $false

function Write-InstallLog {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Host "[install-windows] $Message"
}

function Fail {
  param([Parameter(Mandatory = $true)][string]$Message)
  Write-Error "[install-windows] $Message"
  exit 1
}

function Test-IsWindows {
  return ($env:OS -eq "Windows_NT")
}

function Test-Administrator {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($identity)
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function ConvertTo-ProcessArgument {
  param([AllowEmptyString()][Parameter(Mandatory = $true)][string]$Value)
  return '"' + ($Value -replace '"', '\"') + '"'
}

function Get-PowerShellExecutable {
  $currentProcess = Get-Process -Id $PID -ErrorAction SilentlyContinue
  if ($currentProcess -and $currentProcess.Path) {
    return $currentProcess.Path
  }

  $windowsPowerShell = Join-Path $PSHOME "powershell.exe"
  if (Test-Path -LiteralPath $windowsPowerShell) {
    return $windowsPowerShell
  }

  $pwsh = Get-Command "pwsh.exe" -ErrorAction SilentlyContinue
  if ($pwsh) {
    return $pwsh.Source
  }

  $powershell = Get-Command "powershell.exe" -ErrorAction SilentlyContinue
  if ($powershell) {
    return $powershell.Source
  }

  Fail "PowerShell executable was not found."
}

function Restart-AsAdministratorIfNeeded {
  if (Test-Administrator) {
    return
  }

  if (-not $PSCommandPath) {
    Fail "This script must be run from a .ps1 file so it can relaunch as administrator."
  }

  $powerShellExe = Get-PowerShellExecutable
  $arguments = @(
    "-NoProfile",
    "-ExecutionPolicy",
    "Bypass",
    "-File",
    (ConvertTo-ProcessArgument -Value $PSCommandPath)
  )

  foreach ($entry in $PSBoundParameters.GetEnumerator()) {
    if ($entry.Value -is [System.Management.Automation.SwitchParameter]) {
      if ($entry.Value.IsPresent) {
        $arguments += "-$($entry.Key)"
      }
      continue
    }

    $arguments += "-$($entry.Key)"
    $arguments += (ConvertTo-ProcessArgument -Value ([string]$entry.Value))
  }

  Write-InstallLog "Administrator permission is required; relaunching with UAC prompt."
  Start-Process -FilePath $powerShellExe -ArgumentList $arguments -Verb RunAs | Out-Null
  exit 0
}

function Add-PathEntry {
  param([Parameter(Mandatory = $true)][string]$PathValue)
  $currentPathParts = @()
  if ($env:Path) {
    $currentPathParts = $env:Path.Split(";")
  }
  if ((Test-Path -LiteralPath $PathValue) -and -not ($currentPathParts -contains $PathValue)) {
    $env:Path = "$PathValue;$env:Path"
  }
}

function Refresh-ProcessPath {
  param([string[]]$ExtraPaths = @())

  $pathParts = @()
  $pathParts += $ExtraPaths
  $machinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($machinePath) {
    $pathParts += ($machinePath -split ";")
  }
  if ($userPath) {
    $pathParts += ($userPath -split ";")
  }
  if ($env:Path) {
    $pathParts += ($env:Path -split ";")
  }

  $seen = @{}
  $deduped = @()
  foreach ($pathPart in $pathParts) {
    $trimmed = "$pathPart".Trim()
    if (-not $trimmed) {
      continue
    }
    $key = $trimmed.ToLowerInvariant()
    if (-not $seen.ContainsKey($key)) {
      $seen[$key] = $true
      $deduped += $trimmed
    }
  }

  $env:Path = ($deduped -join ";")
}

function Resolve-CommandPath {
  param([Parameter(Mandatory = $true)][string[]]$Names)
  foreach ($name in $Names) {
    $command = Get-Command $name -ErrorAction SilentlyContinue
    if ($command) {
      return $command.Source
    }
  }
  return ""
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

function Find-LocalInstaller {
  param([Parameter(Mandatory = $true)][string[]]$Filters)

  if (-not (Test-Path -LiteralPath $Root)) {
    return $null
  }

  foreach ($filter in $Filters) {
    $installer = Get-ChildItem -LiteralPath $Root -Filter $filter -File -ErrorAction SilentlyContinue |
      Sort-Object LastWriteTime -Descending |
      Select-Object -First 1
    if ($installer) {
      return $installer
    }
  }

  return $null
}

function Test-ProjectDirectory {
  param([Parameter(Mandatory = $true)][string]$Path)
  return (Test-Path -LiteralPath (Join-Path $Path "scripts\win-lowmem-deploy.ps1"))
}

function Test-PrebuiltStaticPackage {
  param([Parameter(Mandatory = $true)][string]$Path)
  return (
    (Test-Path -LiteralPath (Join-Path $Path "scripts\static-up.ps1")) -and
    (Test-Path -LiteralPath (Join-Path $Path "scripts\static-down.ps1")) -and
    (Test-Path -LiteralPath (Join-Path $Path "server\config.local.example.yaml")) -and
    (Test-Path -LiteralPath (Join-Path $Path "web\dist\index.html")) -and
    (Test-Path -LiteralPath (Join-Path $Path "tmp\static-runtime\bin\gva-server.exe"))
  )
}

function Invoke-WingetInstall {
  param(
    [Parameter(Mandatory = $true)][string]$PackageId,
    [Parameter(Mandatory = $true)][string]$DisplayName
  )

  if ($Offline) {
    return $false
  }

  $winget = Resolve-CommandPath -Names @("winget.exe", "winget")
  if (-not $winget) {
    Write-Warning "winget was not found; cannot install $DisplayName online with winget."
    return $false
  }

  Write-InstallLog "Installing $DisplayName with winget package $PackageId"
  $installArgs = @(
    "install",
    "--id",
    $PackageId,
    "--exact",
    "--silent",
    "--accept-package-agreements",
    "--accept-source-agreements"
  )
  $exitCode = Invoke-ExternalCommand -Path $winget -Arguments $installArgs
  if ($exitCode -eq 0) {
    return $true
  }

  Write-Warning "winget install for $DisplayName exited with code $exitCode; trying winget upgrade."
  $upgradeArgs = @(
    "upgrade",
    "--id",
    $PackageId,
    "--exact",
    "--silent",
    "--accept-package-agreements",
    "--accept-source-agreements"
  )
  $exitCode = Invoke-ExternalCommand -Path $winget -Arguments $upgradeArgs
  return ($exitCode -eq 0)
}

function Install-Msi {
  param(
    [Parameter(Mandatory = $true)][System.IO.FileInfo]$Installer,
    [Parameter(Mandatory = $true)][string]$DisplayName
  )

  Write-InstallLog "Installing $DisplayName from $($Installer.FullName)"
  $process = Start-Process -FilePath "msiexec.exe" -Wait -PassThru -ArgumentList @(
    "/i",
    "`"$($Installer.FullName)`"",
    "/qn",
    "/norestart"
  )
  if (($process.ExitCode -ne 0) -and ($process.ExitCode -ne 3010)) {
    Fail "$DisplayName MSI installer exited with code $($process.ExitCode)."
  }
}

function Install-Exe {
  param(
    [Parameter(Mandatory = $true)][System.IO.FileInfo]$Installer,
    [Parameter(Mandatory = $true)][string]$DisplayName,
    [Parameter(Mandatory = $true)][string[]]$Arguments
  )

  Write-InstallLog "Installing $DisplayName from $($Installer.FullName)"
  $process = Start-Process -FilePath $Installer.FullName -Wait -PassThru -ArgumentList $Arguments
  if (($process.ExitCode -ne 0) -and ($process.ExitCode -ne 3010)) {
    Fail "$DisplayName installer exited with code $($process.ExitCode)."
  }
}

function Get-GitForWindowsInstaller {
  if ($Offline) {
    Fail "Git is missing and -Offline forbids winget or downloading Git for Windows. Put PortableGit, Git-*-64-bit.exe, or Git-*.exe under $Root and rerun."
  }

  Write-InstallLog "Looking up latest Git for Windows installer"
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  try {
    $response = Invoke-WebRequest `
      -Uri $GitForWindowsLatestReleaseApi `
      -UseBasicParsing `
      -Headers @{ "User-Agent" = "install-windows.ps1" }
    $release = $response.Content | ConvertFrom-Json
    $asset = $release.assets |
      Where-Object { $_.name -match "^Git-\d.*-64-bit\.exe$" } |
      Select-Object -First 1
  } catch {
    Fail "Could not query Git for Windows latest release: $($_.Exception.Message). Install Git manually, provide PortableGit under $Root, or put Git-*-64-bit.exe under $Root."
  }

  if (-not $asset -or -not $asset.browser_download_url) {
    Fail "Could not find a Git for Windows 64-bit installer in the latest release. Put Git-*-64-bit.exe under $Root and rerun."
  }

  $downloadPath = Join-Path $Root $asset.name
  Download-File -Url $asset.browser_download_url -Destination $downloadPath -DisplayName "Git for Windows" -MinimumBytes 10485760
  return Get-Item -LiteralPath $downloadPath
}

function Ensure-Git {
  Refresh-ProcessPath -ExtraPaths @(
    (Join-Path $Root "PortableGit\cmd"),
    (Join-Path $Root "PortableGit\bin"),
    "C:\Program Files\Git\cmd",
    "C:\Program Files\Git\bin"
  )

  if (Resolve-CommandPath -Names @("git.exe", "git")) {
    Write-InstallLog "Git is already available."
    return
  }

  if (Test-Path -LiteralPath (Join-Path $Root "PortableGit\cmd\git.exe")) {
    Add-PathEntry -PathValue (Join-Path $Root "PortableGit\cmd")
    Add-PathEntry -PathValue (Join-Path $Root "PortableGit\bin")
    Write-InstallLog "Using PortableGit under $Root."
    return
  }

  $localGitInstaller = Find-LocalInstaller -Filters @("Git-*-64-bit.exe", "Git-*.exe")
  if ($localGitInstaller) {
    Install-Exe -Installer $localGitInstaller -DisplayName "Git" -Arguments @("/VERYSILENT", "/NORESTART", "/NOCANCEL", "/SP-")
    Refresh-ProcessPath -ExtraPaths @("C:\Program Files\Git\cmd", "C:\Program Files\Git\bin")
  } elseif (-not (Invoke-WingetInstall -PackageId "Git.Git" -DisplayName "Git")) {
    $downloadedGitInstaller = Get-GitForWindowsInstaller
    Install-Exe -Installer $downloadedGitInstaller -DisplayName "Git" -Arguments @("/VERYSILENT", "/NORESTART", "/NOCANCEL", "/SP-")
    Refresh-ProcessPath -ExtraPaths @("C:\Program Files\Git\cmd", "C:\Program Files\Git\bin")
  }

  Refresh-ProcessPath -ExtraPaths @("C:\Program Files\Git\cmd", "C:\Program Files\Git\bin")
  if (-not (Resolve-CommandPath -Names @("git.exe", "git"))) {
    Fail "Git is still unavailable after installation. Reopen PowerShell or check PATH."
  }
}

function Ensure-MsiBackedCommand {
  param(
    [Parameter(Mandatory = $true)][string]$DisplayName,
    [Parameter(Mandatory = $true)][string[]]$CommandNames,
    [Parameter(Mandatory = $true)][string[]]$InstallerFilters,
    [Parameter(Mandatory = $true)][string]$InstallPath,
    [Parameter(Mandatory = $true)][string]$WingetPackageId
  )

  Refresh-ProcessPath -ExtraPaths @($InstallPath)
  if (Resolve-CommandPath -Names $CommandNames) {
    Write-InstallLog "$DisplayName is already available."
    return
  }

  $installer = Find-LocalInstaller -Filters $InstallerFilters
  if ($installer) {
    Install-Msi -Installer $installer -DisplayName $DisplayName
  } elseif (-not (Invoke-WingetInstall -PackageId $WingetPackageId -DisplayName $DisplayName)) {
    Fail "$DisplayName is missing. Provide installer $($InstallerFilters -join ' or ') under $Root, or rerun without -Offline so winget can install $WingetPackageId."
  }

  Refresh-ProcessPath -ExtraPaths @($InstallPath)
  if (-not (Resolve-CommandPath -Names $CommandNames)) {
    Fail "$DisplayName is still unavailable after installation. Reopen PowerShell or check PATH."
  }
}

function Test-VcRuntimePresent {
  $system32 = Join-Path $env:WINDIR "System32"
  foreach ($dllName in @("VCRUNTIME140.dll", "VCRUNTIME140_1.dll", "MSVCP140.dll")) {
    if (-not (Test-Path -LiteralPath (Join-Path $system32 $dllName))) {
      return $false
    }
  }
  return $true
}

function Download-File {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [Parameter(Mandatory = $true)][string]$Destination,
    [Parameter(Mandatory = $true)][string]$DisplayName,
    [long]$MinimumBytes = 1
  )

  if ($Offline) {
    Fail "$DisplayName is missing and -Offline forbids downloading $Url."
  }

  Write-InstallLog "Downloading $DisplayName to $Destination"
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  Invoke-WebRequest -Uri $Url -OutFile $Destination -UseBasicParsing

  if (-not (Test-Path -LiteralPath $Destination)) {
    Fail "Download failed for ${DisplayName}: $Destination was not created."
  }

  $file = Get-Item -LiteralPath $Destination
  if ($file.Length -lt $MinimumBytes) {
    Fail "Download failed for ${DisplayName}: file is too small ($($file.Length) bytes)."
  }
}

function Ensure-VcRuntime {
  if (Test-VcRuntimePresent) {
    Write-InstallLog "Microsoft Visual C++ Redistributable x64 is already available."
    return
  }

  $localInstaller = Find-LocalInstaller -Filters @("vc_redist.x64*.exe", "VC_redist.x64*.exe")
  if ($localInstaller) {
    Install-Exe -Installer $localInstaller -DisplayName "Microsoft Visual C++ Redistributable x64" -Arguments @("/install", "/quiet", "/norestart")
  } else {
    $installedWithWinget = Invoke-WingetInstall -PackageId "Microsoft.VCRedist.2015+.x64" -DisplayName "Microsoft Visual C++ Redistributable x64"
    Refresh-ProcessPath
    if (-not $installedWithWinget -or -not (Test-VcRuntimePresent)) {
      $downloadPath = Join-Path $Root "vc_redist.x64.exe"
      Download-File -Url $VcRedistUrl -Destination $downloadPath -DisplayName "Microsoft Visual C++ Redistributable x64" -MinimumBytes 1048576
      $downloadedInstaller = Get-Item -LiteralPath $downloadPath
      Install-Exe -Installer $downloadedInstaller -DisplayName "Microsoft Visual C++ Redistributable x64" -Arguments @("/install", "/quiet", "/norestart")
    }
  }

  if (-not (Test-VcRuntimePresent)) {
    Fail "Microsoft Visual C++ Redistributable x64 is still missing. Reboot Windows or install vc_redist.x64.exe manually."
  }
}

function Find-MySqlHome {
  if (-not (Test-Path -LiteralPath $Root)) {
    return $null
  }

  return Get-ChildItem -LiteralPath $Root -Directory -Filter "mysql-8.0.*-winx64" -ErrorAction SilentlyContinue |
    Where-Object { Test-Path -LiteralPath (Join-Path $_.FullName "bin\mysqld.exe") } |
    Sort-Object LastWriteTime -Descending |
    Select-Object -First 1
}

function Ensure-MySqlZip {
  $mysqlHome = Find-MySqlHome
  if ($mysqlHome) {
    Write-InstallLog "Using MySQL ZIP directory $($mysqlHome.FullName)."
    Add-PathEntry -PathValue (Join-Path $mysqlHome.FullName "bin")
    return $mysqlHome.Name
  }

  $localZip = Find-LocalInstaller -Filters @("mysql-8.0.*-winx64.zip")
  if (-not $localZip) {
    $zipPath = Join-Path $Root $MySqlZipName
    Download-File -Url $MySqlZipUrl -Destination $zipPath -DisplayName "MySQL $MySqlVersion ZIP" -MinimumBytes 104857600
    $localZip = Get-Item -LiteralPath $zipPath
  }

  Write-InstallLog "Expanding MySQL ZIP $($localZip.FullName) to $Root"
  Expand-Archive -LiteralPath $localZip.FullName -DestinationPath $Root -Force

  $mysqlHome = Find-MySqlHome
  if (-not $mysqlHome) {
    Fail "MySQL ZIP was expanded, but no mysql-8.0.*-winx64 folder with bin\mysqld.exe was found under $Root."
  }

  Add-PathEntry -PathValue (Join-Path $mysqlHome.FullName "bin")
  return $mysqlHome.Name
}

function Get-CommandOutput {
  param(
    [Parameter(Mandatory = $true)][string[]]$CommandNames,
    [Parameter(Mandatory = $true)][string[]]$Arguments
  )

  $commandPath = Resolve-CommandPath -Names $CommandNames
  if (-not $commandPath) {
    Fail "Required command is missing: $($CommandNames -join ' or ')."
  }

  $previousErrorActionPreference = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  try {
    $output = & $commandPath @Arguments 2>&1
    $exitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }

  if ($output) {
    $output | ForEach-Object { Write-Host $_ }
  }
  if ($exitCode -ne 0) {
    Fail "$($CommandNames[0]) $($Arguments -join ' ') exited with code $exitCode."
  }

  return ($output -join "`n")
}

function ConvertTo-VersionFromText {
  param(
    [Parameter(Mandatory = $true)][string]$Text,
    [Parameter(Mandatory = $true)][string]$Pattern
  )

  if ($Text -notmatch $Pattern) {
    return $null
  }

  $major = [int]$matches["major"]
  $minor = [int]$matches["minor"]
  $patch = 0
  if ($matches.ContainsKey("patch") -and $matches["patch"]) {
    $patch = [int]$matches["patch"]
  }

  return [version]"$major.$minor.$patch"
}

function Assert-MinimumVersion {
  param(
    [Parameter(Mandatory = $true)][string]$DisplayName,
    [Parameter(Mandatory = $true)][string[]]$CommandNames,
    [Parameter(Mandatory = $true)][string[]]$Arguments,
    [Parameter(Mandatory = $true)][string]$Pattern,
    [Parameter(Mandatory = $true)][version]$MinimumVersion,
    [Parameter(Mandatory = $true)][string]$FixHint,
    [string]$WingetPackageId = "",
    [string[]]$RefreshPaths = @()
  )

  $output = Get-CommandOutput -CommandNames $CommandNames -Arguments $Arguments
  $version = ConvertTo-VersionFromText -Text $output -Pattern $Pattern
  if (-not $version) {
    Write-Warning "Could not parse $DisplayName version from output; continuing after command succeeded."
    return
  }

  if ($version -lt $MinimumVersion) {
    if ($WingetPackageId -and -not $Offline) {
      Write-Warning "$DisplayName version $version is too old; trying winget package $WingetPackageId."
      if (Invoke-WingetInstall -PackageId $WingetPackageId -DisplayName $DisplayName) {
        Refresh-ProcessPath -ExtraPaths $RefreshPaths
        $output = Get-CommandOutput -CommandNames $CommandNames -Arguments $Arguments
        $version = ConvertTo-VersionFromText -Text $output -Pattern $Pattern
        if ($version -and $version -ge $MinimumVersion) {
          return
        }
      }
    }
    Fail "$DisplayName version $version is too old; need >= $MinimumVersion. $FixHint"
  }
}

function Resolve-Layout {
  if ((-not $PSBoundParameters.ContainsKey("Root")) -and
      (-not $PSBoundParameters.ContainsKey("AppName")) -and
      (Test-ProjectDirectory -Path $ScriptProjectRoot)) {
    $script:Root = Split-Path -Parent $ScriptProjectRoot
    $script:AppName = Split-Path -Leaf $ScriptProjectRoot
    $script:App = $ScriptProjectRoot
    Write-InstallLog "Using project checkout beside this script: $script:App"
    return
  }

  if ([string]::IsNullOrWhiteSpace($script:AppName)) {
    $script:AppName = "test-eee-git"
  }

  New-Item -ItemType Directory -Force -Path $script:Root | Out-Null
  $candidateApp = Join-Path $script:Root $script:AppName

  if (-not (Test-ProjectDirectory -Path $candidateApp)) {
    $fallbackApp = Get-ChildItem -LiteralPath $script:Root -Directory -ErrorAction SilentlyContinue |
      Where-Object { Test-ProjectDirectory -Path $_.FullName } |
      Sort-Object LastWriteTime -Descending |
      Select-Object -First 1
    if ($fallbackApp) {
      $candidateApp = $fallbackApp.FullName
      $script:AppName = $fallbackApp.Name
    }
  }

  if (-not (Test-ProjectDirectory -Path $candidateApp)) {
    if ($Offline) {
      Fail "Project checkout was not found under $script:Root and -Offline forbids git clone. Put the project there first or rerun without -Offline."
    }
    if ([string]::IsNullOrWhiteSpace($GitRepoUrl)) {
      Fail "Project checkout was not found and GitRepoUrl is empty. Pass -GitRepoUrl to clone the project."
    }

    $git = Resolve-CommandPath -Names @("git.exe", "git")
    if (-not $git) {
      Fail "Git is required before cloning the project."
    }

    if (Test-Path -LiteralPath $candidateApp) {
      Fail "Target directory exists but is not this project: $candidateApp. Remove it, pass a different -AppName, or pass -Root to another install directory."
    }

    $cloneArgs = @("clone")
    if (-not [string]::IsNullOrWhiteSpace($GitBranch)) {
      $cloneArgs += @("--branch", $GitBranch)
    }
    $cloneArgs += @($GitRepoUrl, $candidateApp)

    Write-InstallLog "Cloning project from $GitRepoUrl to $candidateApp"
    $exitCode = Invoke-ExternalCommand -Path $git -Arguments $cloneArgs
    if ($exitCode -ne 0) {
      Fail "git clone failed with exit code $exitCode. Check network access, repository permissions, or pass a reachable -GitRepoUrl."
    }
  }

  if (-not (Test-ProjectDirectory -Path $candidateApp)) {
    Fail "Project directory is missing scripts\win-lowmem-deploy.ps1 after checkout: $candidateApp."
  }

  if ((-not $SkipGitPull) -and (Test-Path -LiteralPath (Join-Path $candidateApp ".git"))) {
    $git = Resolve-CommandPath -Names @("git.exe", "git")
    if ($git) {
      Write-InstallLog "Updating project checkout with git pull --ff-only"
      $exitCode = Invoke-ExternalCommand -Path $git -Arguments @("-C", $candidateApp, "pull", "--ff-only")
      if ($exitCode -ne 0) {
        Write-Warning "git pull failed with exit code $exitCode; continuing with local checkout."
      }
    }
  }

  $script:Root = (Resolve-Path -LiteralPath $script:Root).Path
  $script:App = (Resolve-Path -LiteralPath $candidateApp).Path
  $script:AppName = Split-Path -Leaf $script:App
}

function Validate-Toolchain {
  param([bool]$RequireBuildToolchain = $true)

  Assert-MinimumVersion `
    -DisplayName "Git" `
    -CommandNames @("git.exe", "git") `
    -Arguments @("--version") `
    -Pattern "git version (?<major>\d+)\.(?<minor>\d+)(\.(?<patch>\d+))?" `
    -MinimumVersion $MinimumGitVersion `
    -FixHint "Install Git.Git with winget, or provide PortableGit under $Root." `
    -WingetPackageId "Git.Git" `
    -RefreshPaths @("C:\Program Files\Git\cmd", "C:\Program Files\Git\bin", (Join-Path $Root "PortableGit\cmd"), (Join-Path $Root "PortableGit\bin"))

  if ($RequireBuildToolchain) {
    Assert-MinimumVersion `
      -DisplayName "Go" `
      -CommandNames @("go.exe", "go") `
      -Arguments @("version") `
      -Pattern "go(?<major>\d+)\.(?<minor>\d+)(\.(?<patch>\d+))?" `
      -MinimumVersion $MinimumGoVersion `
      -FixHint "Install GoLang.Go with winget, or provide go-installer*.msi under $Root." `
      -WingetPackageId "GoLang.Go" `
      -RefreshPaths @("C:\Program Files\Go\bin")

    Assert-MinimumVersion `
      -DisplayName "Node.js" `
      -CommandNames @("node.exe", "node") `
      -Arguments @("-v") `
      -Pattern "v(?<major>\d+)\.(?<minor>\d+)(\.(?<patch>\d+))?" `
      -MinimumVersion $MinimumNodeVersion `
      -FixHint "Install OpenJS.NodeJS.LTS with winget, or provide node-installer*.msi under $Root." `
      -WingetPackageId "OpenJS.NodeJS.LTS" `
      -RefreshPaths @("C:\Program Files\nodejs")

    Get-CommandOutput -CommandNames @("npm.cmd", "npm") -Arguments @("-v") | Out-Null
  } else {
    Write-InstallLog "prebuilt static mode: skipping Go, Node.js, and npm version checks"
  }

  $mysqlExe = Join-Path $script:MySqlHomePath "bin\mysql.exe"
  if (-not (Test-Path -LiteralPath $mysqlExe)) {
    Fail "mysql.exe not found at $mysqlExe."
  }
  $exitCode = Invoke-ExternalCommand -Path $mysqlExe -Arguments @("--version")
  if ($exitCode -ne 0) {
    Fail "mysql.exe --version failed with exit code $exitCode. Check Microsoft Visual C++ Redistributable x64 and the MySQL ZIP under $Root."
  }
}

function Start-LowMemoryDeployment {
  $deployScript = Join-Path $script:App "scripts\win-lowmem-deploy.ps1"
  $powerShellExe = Get-PowerShellExecutable
  $arguments = @(
    "-NoProfile",
    "-ExecutionPolicy",
    "Bypass",
    "-File",
    $deployScript,
    "-Root",
    $script:Root,
    "-AppName",
    $script:AppName,
    "-MySqlFolderName",
    $script:MySqlFolderName,
    "-MySqlRootPassword",
    $MySqlRootPassword,
    "-AdminPassword",
    $AdminPassword,
    "-ApiPort",
    "$ApiPort",
    "-MySqlPort",
    "$MySqlPort"
  )
  if ($SkipGitPull) {
    $arguments += "-SkipGitPull"
  }
  if ($ResetMySqlData) {
    $arguments += "-ResetMySqlData"
  }
  if ($script:UsePrebuiltStatic) {
    $arguments += "-UsePrebuiltStatic"
    $arguments += "-PrebuiltRelativePath"
    $arguments += $PrebuiltRelativePath
  }

  Write-InstallLog "Starting project deployment via $deployScript"
  & $powerShellExe @arguments
  if ($LASTEXITCODE -ne 0) {
    Fail "win-lowmem-deploy.ps1 failed with exit code $LASTEXITCODE."
  }
}

if (-not (Test-IsWindows)) {
  Fail "This installer is intended for Windows 10/11 PowerShell."
}

Restart-AsAdministratorIfNeeded
New-Item -ItemType Directory -Force -Path $Root | Out-Null

Write-InstallLog "Root: $Root"
if ($Offline) {
  Write-InstallLog "Offline mode is enabled; network downloads and winget installs are disabled."
}

Ensure-Git
Resolve-Layout
Write-InstallLog "Project: $App"
$prebuiltPath = Join-Path $App $PrebuiltRelativePath
$script:UsePrebuiltStatic = ((-not $BuildOnTarget) -and (Test-PrebuiltStaticPackage -Path $prebuiltPath))
if ($script:UsePrebuiltStatic) {
  Write-InstallLog "Using prebuilt Windows static package: $prebuiltPath"
} elseif ($BuildOnTarget) {
  Write-InstallLog "BuildOnTarget was specified; installing Go and Node.js for local Windows build"
} else {
  Write-Warning "Prebuilt package not found at $prebuiltPath; falling back to Windows target build"
}

if (-not $script:UsePrebuiltStatic) {
  Ensure-MsiBackedCommand `
    -DisplayName "Go" `
    -CommandNames @("go.exe", "go") `
    -InstallerFilters @("go-installer*.msi", "go*.windows-amd64.msi") `
    -InstallPath "C:\Program Files\Go\bin" `
    -WingetPackageId "GoLang.Go"
  Ensure-MsiBackedCommand `
    -DisplayName "Node.js LTS" `
    -CommandNames @("node.exe", "node") `
    -InstallerFilters @("node-installer*.msi", "node-v*-x64.msi") `
    -InstallPath "C:\Program Files\nodejs" `
    -WingetPackageId "OpenJS.NodeJS.LTS"
} else {
  Write-InstallLog "prebuilt static mode: skipping Go and Node.js installation"
}
Ensure-VcRuntime
$script:MySqlFolderName = Ensure-MySqlZip
$script:MySqlHomePath = Join-Path $Root $script:MySqlFolderName
Refresh-ProcessPath -ExtraPaths @(
  (Join-Path $Root "PortableGit\cmd"),
  (Join-Path $Root "PortableGit\bin"),
  "C:\Program Files\Git\cmd",
  "C:\Program Files\Git\bin",
  "C:\Program Files\Go\bin",
  "C:\Program Files\nodejs",
  (Join-Path $script:MySqlHomePath "bin")
)

Validate-Toolchain -RequireBuildToolchain:(-not $script:UsePrebuiltStatic)
Start-LowMemoryDeployment

Write-Host ""
Write-Host "Windows one-click installation is complete:"
Write-Host "  Login: http://127.0.0.1:$ApiPort/#/login"
Write-Host "  Username: admin"
Write-Host "  Password: $AdminPassword"
Write-Host ""
Write-Host "Stop:"
if ($script:UsePrebuiltStatic) {
  Write-Host "  cd $prebuiltPath"
} else {
  Write-Host "  cd $App"
}
Write-Host "  .\scripts\static-down.ps1"
