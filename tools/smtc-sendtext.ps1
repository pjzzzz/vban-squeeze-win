<#
.SYNOPSIS
    Sends the current Windows "now playing" title to Lyrion as VBAN TEXT packets.

.DESCRIPTION
    Works with any player that reports to the Windows System Media Transport
    Controls (MediaMonkey, Spotify, browsers, ...). Every time the title or
    artist changes it sends a VBAN TEXT packet (sub-protocol 0x40) with the same
    stream name as the audio stream, so the Lyrion plugin can show the track.

    Run this on the machine that plays the audio (the VBAN sender), not on the
    Lyrion server. Use Windows PowerShell 5.1 (powershell.exe), not pwsh: the
    WinRT media API is not available in PowerShell 7.

.EXAMPLE
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tools\smtc-sendtext.ps1 -Target 192.168.178.30 -Stream Stream1

.EXAMPLE
    # Run once and exit (for testing)
    .\smtc-sendtext.ps1 -Target 192.168.178.30 -Once
#>
[CmdletBinding()]
param(
    [string]$Target = '192.168.178.30',
    [int]$Port = 6980,
    [string]$Stream = 'Stream1',
    [int]$Interval = 2,
    [string]$App = '',
    [switch]$Once,
    [switch]$Quiet
)

$ErrorActionPreference = 'Stop'

Add-Type -AssemblyName System.Runtime.WindowsRuntime

$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object {
    $_.Name -eq 'AsTask' -and
    $_.GetParameters().Count -eq 1 -and
    $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`1'
})[0]

function Await($operation, $resultType) {
    $task = $asTaskGeneric.MakeGenericMethod($resultType).Invoke($null, @($operation))
    if (-not $task.Wait(5000)) {
        throw 'WinRT call timed out'
    }
    return $task.Result
}

function Get-NowPlaying {
    $managerType = [Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager, Windows.Media.Control, ContentType = WindowsRuntime]
    $manager = Await ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager]::RequestAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionManager])

    $candidates = @()

    foreach ($session in $manager.GetSessions()) {
        if ($App -and $session.SourceAppUserModelId -notlike "*$App*") {
            continue
        }

        $info = $session.GetPlaybackInfo()
        if ("$($info.PlaybackStatus)" -ne 'Playing') {
            continue
        }

        $props = Await ($session.TryGetMediaPropertiesAsync()) ([Windows.Media.Control.GlobalSystemMediaTransportControlsSessionMediaProperties])

        if ([string]::IsNullOrWhiteSpace($props.Title)) {
            continue
        }

        $candidates += [pscustomobject]@{
            Title   = $props.Title
            Artist  = $props.Artist
            Updated = $session.LastUpdatedTime
            App     = $session.SourceAppUserModelId
        }
    }

    if (-not $candidates) {
        return $null
    }

    # Prefer sessions that report an artist (media players) over browser tabs
    # that only expose a stream title; then take the most recently updated one.
    $withArtist = @($candidates | Where-Object { -not [string]::IsNullOrWhiteSpace($_.Artist) })
    if ($withArtist) {
        $candidates = $withArtist
    }

    $pick = $candidates | Sort-Object Updated -Descending | Select-Object -First 1

    if ([string]::IsNullOrWhiteSpace($pick.Artist)) {
        return $pick.Title
    }

    return "$($pick.Artist) - $($pick.Title)"
}

function Send-VbanText {
    param([string]$Text)

    $bytes = [System.Text.Encoding]::UTF8.GetBytes($Text)
    if ($bytes.Length -gt 1400) {
        $bytes = $bytes[0..1399]
    }

    $packet = [byte[]]::new(28 + $bytes.Length)
    [System.Text.Encoding]::ASCII.GetBytes('VBAN').CopyTo($packet, 0)
    $packet[4] = 0x40   # VBAN TEXT sub-protocol, bps 0
    $packet[5] = 0      # nbs
    $packet[6] = 0      # ident
    $packet[7] = 1      # UTF-8

    $name = [byte[]]::new(16)
    [System.Text.Encoding]::ASCII.GetBytes($Stream).CopyTo($name, 0)
    $name.CopyTo($packet, 8)

    $bytes.CopyTo($packet, 28)

    $client = [System.Net.Sockets.UdpClient]::new()
    try {
        [void]$client.Send($packet, $packet.Length, (New-Object System.Net.IPEndPoint([System.Net.IPAddress]::Parse($Target), $Port)))
    }
    finally {
        $client.Dispose()
    }
}

if (-not $Quiet) {
    Write-Host ("sending now-playing text to {0}:{1} stream '{2}' (every {3}s)" -f $Target, $Port, $Stream, $Interval)
}

$last = $null

do {
    try {
        $title = Get-NowPlaying
        if ($title -and $title -ne $last) {
            Send-VbanText -Text $title
            $last = $title
            if (-not $Quiet) {
                Write-Host ("title: {0}" -f $title)
            }
        }
    }
    catch {
        if (-not $Quiet) {
            Write-Warning ("SMTC read failed: {0}" -f $_.Exception.Message)
        }
    }

    if (-not $Once) {
        Start-Sleep -Seconds $Interval
    }
} while (-not $Once)
