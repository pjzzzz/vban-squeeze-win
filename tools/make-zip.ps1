<#
.SYNOPSIS
    Builds the LMS-installable plugin zip (vban-<version>.zip).

.DESCRIPTION
    Copies plugin\VBAN to dist\VBAN, verifies the receiver binary exists and
    compresses the folder so the zip contains a top-level VBAN directory, which
    is what Lyrion expects.

.EXAMPLE
    .\tools\make-zip.ps1 -Version 1.3.2
#>
[CmdletBinding()]
param(
    [string]$Version = '1.3.2',
    [string]$Exe = '',
    [string]$OutputDir = ''
)

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
if (-not $OutputDir) {
    $OutputDir = Join-Path $root 'dist'
}
if (-not $Exe) {
    $Exe = Join-Path $root 'plugin\VBAN\Bin\MSWin32-x64-multi-thread\vban_url_receptor.exe'
}

if (-not (Test-Path $Exe)) {
    throw "receiver binary not found: $Exe (run: go build -o `"$Exe`" ./receiver)"
}

$stage = Join-Path $OutputDir 'stage'
$target = Join-Path $stage 'VBAN'

if (Test-Path $stage) {
    Remove-Item -LiteralPath $stage -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $target | Out-Null

Copy-Item -Path (Join-Path $root 'plugin\VBAN\*') -Destination $target -Recurse -Force

# .gitkeep only exists to keep the Bin directory in git; never ship it.
Get-ChildItem -Path $target -Recurse -Filter '.gitkeep' -File | Remove-Item -Force

$binDir = Join-Path $target 'Bin\MSWin32-x64-multi-thread'
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
Copy-Item -LiteralPath $Exe -Destination (Join-Path $binDir 'vban_url_receptor.exe') -Force

$zip = Join-Path $OutputDir ("vban-{0}.zip" -f $Version)
if (Test-Path $zip) {
    Remove-Item -LiteralPath $zip -Force
}
Compress-Archive -Path $target -DestinationPath $zip -CompressionLevel Optimal

Remove-Item -LiteralPath $stage -Recurse -Force

$item = Get-Item $zip
Write-Host ("created {0} ({1:N0} bytes)" -f $item.FullName, $item.Length)
