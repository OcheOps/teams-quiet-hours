$ErrorActionPreference = "Stop"

$Repo = if ($env:TEAMS_QUIET_HOURS_REPO) { $env:TEAMS_QUIET_HOURS_REPO } else { "OcheOps/teams-quiet-hours" }
$InstallDir = if ($env:TEAMS_QUIET_HOURS_INSTALL_DIR) { $env:TEAMS_QUIET_HOURS_INSTALL_DIR } else { Join-Path $env:ProgramFiles "teams-quiet-hours" }
$ExePath = Join-Path $InstallDir "teams-quiet-hours.exe"

function Test-Admin {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($identity)
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Admin)) {
  Write-Host "teams-quiet-hours needs Administrator permission to install browser policies and scheduled tasks."
  Write-Host "Please re-run this script from an Administrator PowerShell window."
  exit 1
}

$Arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$Asset = "teams-quiet-hours-windows-$Arch.exe"
$BaseUrl = "https://github.com/$Repo/releases/latest/download"
$TempDir = New-Item -ItemType Directory -Path (Join-Path ([IO.Path]::GetTempPath()) ("teams-quiet-hours-" + [guid]::NewGuid()))

try {
  $DownloadExe = Join-Path $TempDir $Asset
  $Checksums = Join-Path $TempDir "checksums.txt"

  Write-Host "Downloading $Asset from GitHub Releases..."
  Invoke-WebRequest -Uri "$BaseUrl/$Asset" -OutFile $DownloadExe
  Invoke-WebRequest -Uri "$BaseUrl/checksums.txt" -OutFile $Checksums

  $ExpectedLine = Select-String -Path $Checksums -Pattern "  $([regex]::Escape($Asset))$"
  if (-not $ExpectedLine) { throw "Checksum for $Asset was not found." }
  $ExpectedHash = ($ExpectedLine.Line -split "\s+")[0].ToUpperInvariant()
  $ActualHash = (Get-FileHash -Algorithm SHA256 $DownloadExe).Hash.ToUpperInvariant()
  if ($ExpectedHash -ne $ActualHash) { throw "Checksum verification failed." }

  New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
  Copy-Item -Force $DownloadExe $ExePath

  $MachinePath = [Environment]::GetEnvironmentVariable("Path", "Machine")
  if ($MachinePath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$MachinePath;$InstallDir", "Machine")
    Write-Host "Added $InstallDir to the machine PATH. Open a new terminal to use teams-quiet-hours directly."
  }

  Write-Host ""
  Write-Host "Starting guided setup..."
  & $ExePath setup
  Write-Host ""
  Write-Host "Installed. Run 'teams-quiet-hours doctor' in a new terminal to check the setup."
}
finally {
  Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}

