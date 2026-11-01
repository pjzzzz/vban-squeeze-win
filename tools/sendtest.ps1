<#
.SYNOPSIS
    Sends a synthetic VBAN audio stream (440 Hz sine, 16-bit stereo) for testing.

.DESCRIPTION
    Test helper for vban_url_receptor.exe. It lets you verify the receiver
    without VoiceMeeter or Lyrion. The stream name, sample rate, port and
    target address must match the URL you pass to the receiver.

.EXAMPLE
    .\tools\sendtest.ps1 -Target 127.0.0.1 -Port 6980 -Stream Stream1 -Seconds 5
#>
[CmdletBinding()]
param(
    [string]$Target = '127.0.0.1',
    [int]$Port = 6980,
    [string]$Stream = 'Stream1',
    [int]$Seconds = 5,
    [int]$SampleRate = 44100,
    [int]$Frequency = 440,
    [int]$Amplitude = 16000,
    [int]$SamplesPerFrame = 256
)

$ErrorActionPreference = 'Stop'

$sampleRates = @(6000, 12000, 24000, 48000, 96000, 192000, 384000, 8000, 16000,
    32000, 64000, 128000, 256000, 512000, 11025, 22050, 44100, 88200, 176400, 352800, 705600)
$srIndex = [array]::IndexOf($sampleRates, $SampleRate)
if ($srIndex -lt 0) {
    throw "Unsupported sample rate $SampleRate (VBAN only supports the standard list)"
}

$channels = 2
$frameBytes = $SamplesPerFrame * $channels * 2
if ($frameBytes -gt 1436) {
    throw "SamplesPerFrame too large: VBAN payload must stay below 1436 bytes"
}
if ($Stream.Length -gt 16) {
    throw "Stream name must be at most 16 characters"
}

# One second of 16-bit stereo PCM, looped while sending.
$second = [byte[]]::new($SampleRate * $channels * 2)
$twoPi = 2.0 * [math]::PI
for ($i = 0; $i -lt $SampleRate; $i++) {
    $value = [int16][math]::Round($Amplitude * [math]::Sin($twoPi * $Frequency * $i / $SampleRate))
    $bytes = [System.BitConverter]::GetBytes($value)
    [Array]::Copy($bytes, 0, $second, $i * 4, 2)
    [Array]::Copy($bytes, 0, $second, $i * 4 + 2, 2)
}

$header = [byte[]]::new(28)
[System.Text.Encoding]::ASCII.GetBytes('VBAN').CopyTo($header, 0)
$header[4] = [byte]($srIndex -bor 0x00)   # audio sub-protocol
$header[5] = [byte]($SamplesPerFrame - 1)
$header[6] = [byte]($channels - 1)
$header[7] = [byte]1                      # 16-bit PCM
$name = [byte[]]::new(16)
[System.Text.Encoding]::ASCII.GetBytes($Stream).CopyTo($name, 0)
$name.CopyTo($header, 8)

$packet = [byte[]]::new(28 + $frameBytes)
$header.CopyTo($packet, 0)

$client = [System.Net.Sockets.UdpClient]::new()
$endpoint = [System.Net.IPEndPoint]::new([System.Net.IPAddress]::Parse($Target), $Port)

# Multicast targets need loopback enabled for local tests (receiver on the
# same machine); harmless for unicast.
try {
    $ip = [System.Net.IPAddress]::Parse($Target)
    if (($ip.GetAddressBytes()[0] -band 0xF0) -eq 0xE0) {
        $client.Client.SetSocketOption(
            [System.Net.Sockets.SocketOptionLevel]::IP,
            [System.Net.Sockets.SocketOptionName]::MulticastLoopback, $true)
    }
} catch { }

Write-Host ("sending {0} s of VBAN to {1}:{2} stream '{3}' ({4} Hz, 16-bit, 2 ch)" -f `
        $Seconds, $Target, $Port, $Stream, $SampleRate)

$stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
[uint32]$frameCounter = 0
$offset = 0
$sentFrames = 0
$totalFrames = $Seconds * $SampleRate
$packets = 0

while ($sentFrames -lt $totalFrames) {
    $remaining = $second.Length - $offset
    if ($remaining -ge $frameBytes) {
        [Array]::Copy($second, $offset, $packet, 28, $frameBytes)
    }
    else {
        [Array]::Copy($second, $offset, $packet, 28, $remaining)
        [Array]::Copy($second, 0, $packet, 28 + $remaining, $frameBytes - $remaining)
    }
    [System.BitConverter]::GetBytes($frameCounter).CopyTo($packet, 24)

    $client.Send($packet, $packet.Length, $endpoint) | Out-Null

    $frameCounter++
    $packets++
    $sentFrames += $SamplesPerFrame
    $offset = ($offset + $frameBytes) % $second.Length

    $targetMs = $sentFrames / $SampleRate * 1000.0
    $wait = $targetMs - $stopwatch.Elapsed.TotalMilliseconds
    if ($wait -gt 1) {
        Start-Sleep -Milliseconds ([int]$wait)
    }
}

$client.Dispose()
Write-Host ("done: {0} packets, {1} frames, {2:N2} s elapsed" -f `
        $packets, $sentFrames, $stopwatch.Elapsed.TotalSeconds)
