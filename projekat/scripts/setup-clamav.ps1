# Jednokratno: preuzmi ClamAV virus definicije u storage/clamav/database (bez admina).
$ErrorActionPreference = "Stop"
$Root = Split-Path $PSScriptRoot -Parent
$DbDir = Join-Path $Root "storage\clamav\database"
$Conf = Join-Path $Root "storage\clamav\freshclam.conf"
$Freshclam = Join-Path ${env:ProgramFiles} "ClamAV\freshclam.exe"
$Clamscan = Join-Path ${env:ProgramFiles} "ClamAV\clamscan.exe"

if (-not (Test-Path $Freshclam)) {
    Write-Error 'freshclam not found - install ClamAV x64 first.'
}

New-Item -ItemType Directory -Path $DbDir -Force | Out-Null

$confText = "DatabaseDirectory $DbDir`nDatabaseMirror database.clamav.net`n"
Set-Content -Path $Conf -Value $confText -Encoding ASCII

Write-Host "Downloading virus definitions to: $DbDir"
& $Freshclam --config-file=$Conf
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host 'OK. clamscan version:'
& $Clamscan -d $DbDir --version
