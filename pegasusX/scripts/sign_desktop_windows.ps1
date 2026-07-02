# Sign PegasusX desktop Windows installers with Authenticode (EV cert).
# Usage (GitHub Actions or release VM):
#   $env:WINDOWS_CODESIGN_PFX_B64 = '<base64 pfx>'
#   $env:WINDOWS_CODESIGN_PASSWORD = '<pfx password>'
#   .\scripts\sign_desktop_windows.ps1 -Path "apps\retailer-app-desktop\src-tauri\target\...\bundle\nsis\*.exe"
param(
  [Parameter(Mandatory = $true)]
  [string]$Path,
  [string]$TimestampUrl = "http://timestamp.digicert.com"
)

$ErrorActionPreference = "Stop"

if (-not $env:WINDOWS_CODESIGN_PFX_B64) {
  Write-Host "SKIP Authenticode: WINDOWS_CODESIGN_PFX_B64 not set"
  exit 0
}

$pfxPath = Join-Path $env:RUNNER_TEMP "pegasusx-codesign.pfx"
[System.IO.File]::WriteAllBytes($pfxPath, [Convert]::FromBase64String($env:WINDOWS_CODESIGN_PFX_B64))

$files = Get-ChildItem -Path $Path -ErrorAction SilentlyContinue
if (-not $files) {
  Write-Error "No files matched: $Path"
}

foreach ($file in $files) {
  Write-Host "Signing $($file.FullName)"
  & signtool.exe sign /f $pfxPath /p $env:WINDOWS_CODESIGN_PASSWORD /tr $TimestampUrl /td sha256 /fd sha256 $file.FullName
  if ($LASTEXITCODE -ne 0) {
    Write-Error "signtool failed for $($file.FullName)"
  }
}

Write-Host "windows-codesign-ok"
