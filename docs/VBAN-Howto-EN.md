# VBAN plugin for Lyrion Music Server (Windows) — complete how-to

Version 1.7.4 · Windows only · GPL-3.0

This manual covers installation, VoiceMeeter configuration, sample rates,
track info, multicast, diagnostics and troubleshooting. It is the extended
version of the README; everything here has been tested with Lyrion 9.1.1 on
Windows and a Squeezebox Classic.

---

## 1. What it does

The plugin lets Lyrion Music Server play a **VBAN** audio stream from
VoiceMeeter on a Squeezebox or Squeezelite player.

```
VoiceMeeter (Windows)                 Lyrion Music Server (Windows)
┌────────────────────┐   VBAN/UDP    ┌──────────────────────────────────────┐
│ Outgoing Stream 1  │ ────────────► │ vban_url_receptor.exe                │
│ PCM 16 bit, 2 ch   │    port 6980  │   │ raw PCM (stdout)                 │
└────────────────────┘               │   ▼                                  │
                                     │ Lyrion transcoder pipeline           │
                                     │   │                                  │
                                     │   ▼                                  │
                                     │ Squeezebox / Squeezelite player      │
                                     └──────────────────────────────────────┘
```

The plugin ships a native Windows receiver (`vban_url_receptor.exe`, written in
Go, no runtime required). It listens for VBAN on UDP, converts the samples to
16-bit stereo PCM and writes that to stdout, which Lyrion reads like any other
transcoder output. Lyrion then treats the stream as a normal radio stream.

Why a separate receiver: the original plugin is Linux-only because it depends on
the Linux `vban_receptor` binary. This port replaces that with a Windows binary
that speaks the same protocol.

---

## 2. Requirements

| Component | Requirement |
|-----------|-------------|
| Lyrion Music Server | Windows, 7.9 or newer (tested with 9.1.1) |
| VBAN sender | VoiceMeeter Banana / Potato / VoiceMeeter 8 |
| Stream format | PCM 16 bit, 2 channels; rate = engine rate |
| Player | Any Squeezebox or Squeezelite player that supports raw PCM |
| Network | UDP 6980 from the sender to the Lyrion host |

Tested setup: Lyrion 9.1.1 on Windows, VoiceMeeter Potato, Squeezebox Classic.
Squeezelite and the other Squeezebox models use the same PCM path and work the
same way.

---

## 3. How it works

- The playback URL is `vban://<sender-ip>/<streamname>` (see section 6).
- When the player starts that URL, Lyrion starts one `vban_url_receptor.exe` per
  stream and reads its PCM output.
- The receiver binds the UDP port (default 6980), accepts packets whose stream
  name matches, and ignores everything else.
- The plugin tells the receiver which output rate to use (`--target-rate`), and
  that value is exactly the rate Lyrion announces to the player. Player and
  stream therefore always agree.
- When the VBAN source stops (VoiceMeeter stream Off), the receiver writes
  silence instead of ending the stream. Playback resumes by itself when packets
  return, and the player does not mark the stream as failed.

---

## 4. Installation

### 4.1 Through the plugin repository (recommended)

1. Open **Settings → Plugins**.
2. Under **Additional Repositories** add:

   ```
   https://raw.githubusercontent.com/pjzzzz/vban-squeeze-win/main/vban.xml
   ```

3. Apply, then install **VBAN** from the plugin list.
4. Restart Lyrion.
5. Confirm that **VBAN** is enabled under **Settings → Plugins**.

The repository XML deliberately has no `sha` field: the zip is fetched from
GitHub Releases over HTTPS, and a hash pinned in the repository would go stale
as soon as a new binary is built.

### 4.2 Manual installation (fallback)

Use this only when you cannot add a repository (for example to test a private
build).

1. Download `vban-<version>.zip` from
   <https://github.com/pjzzzz/vban-squeeze-win/releases>.
2. Stop Lyrion Music Server.
3. Extract the zip so that you end up with:

   ```
   C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\install.xml
   ```

   If Lyrion is installed elsewhere, use that installation's
   `Cache\InstalledPlugins\Plugins\VBAN\` folder.
4. Unblock the receiver binary (Windows marks files downloaded from the
   internet):

   ```powershell
   Unblock-File 'C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\Bin\MSWin32-x64-multi-thread\vban_url_receptor.exe'
   ```

5. Start Lyrion Music Server.
6. Confirm that **VBAN** is enabled under **Settings → Plugins**.

### 4.3 Firewall

Allow inbound VBAN traffic on the Lyrion host. In an elevated PowerShell:

```powershell
New-NetFirewallRule -DisplayName 'VBAN receptor (UDP 6980)' `
    -Direction Inbound -Protocol UDP -LocalPort 6980 `
    -Action Allow -Profile Any
```

`-Profile Any` also covers the Public profile. If you prefer to restrict it, use
`-Profile Private,Domain` and make sure the server's network is not classified
as Public. For extra hardening, limit the rule to the sending machine with
`-RemoteAddress <sender-ip>`.

Without this rule the receiver starts normally, but no packets ever arrive.

### 4.4 Verify the installation

- **Settings → Plugins** shows **VBAN** and its version.
- **Apps** contains **VBAN** (and the Radio menu has a VBAN entry).
- The binary exists:
  `...\Plugins\VBAN\Bin\MSWin32-x64-multi-thread\vban_url_receptor.exe`.

---

## 5. VoiceMeeter configuration

### 5.1 Engine sample rate

VoiceMeeter's engine rate is set under **Menu → System Settings → Sample Rate**
(44100 or 48000). The ASIO driver VoiceMeeter exposes to other programs always
runs at that rate.

Choose the rate you need for your hardware (an interface that only runs at
48000 forces 48000), and keep it consistent: the next two sections depend on it.

### 5.2 Outgoing VBAN stream

1. Click **VBAN** (top right).
2. Under **Outgoing Streams** pick the bus you want to send (A1…A5, B1…B3).
3. Set:

   | Field | Value |
   |-------|-------|
   | IP address | the Lyrion server, for example `192.168.178.30` |
   | Sample rate | **the same as the engine rate** (48000 with 48000, 44100 with 44100) |
   | Channels | `2` |
   | Format | `PCM 16 bits` |
   | Stream name | `Stream1` (must match the URL) |

4. Switch the stream **On**, and switch **VBAN On** (top left).

The sender IP is shown at the top of the VBAN window; you need it for the
playback URL.

If you change the sample rate while the stream is running, switch the stream
**Off and On again** (or restart VoiceMeeter). Otherwise the change is not
applied to the running stream and the receiver keeps seeing the old rate.

### 5.3 The source must match the engine rate (the 8.8% gotcha)

This is the one issue that looks like a plugin bug but is not.

VoiceMeeter's ASIO driver runs at the engine rate. The program that feeds
VoiceMeeter must output that same rate. If it does not, VoiceMeeter converts it
internally and the result can be about **8.8% too fast and too high** (48000 vs
44100), sometimes with wobble.

Symptoms: music is clearly higher in pitch and runs ahead; a track that normally
lasts 4:00 finishes noticeably earlier.

Fix:

- **MediaMonkey 5**: Tools → Options → Output Plug-ins → *ASIO output* →
  Configure → **Sample rate 48000** (or "Resample to 48000") when the
  VoiceMeeter engine runs at 48000.
- Any other player: set its output rate to the engine rate as well.

Nothing needs to be configured in Lyrion: the plugin passes the rate Lyrion
announces to the player to the receiver, so the receiver and the player always
agree.

### 5.4 Routing tips

- Stream the bus that actually carries the audio you want (often A1).
- Keep the VoiceMeeter master output and the VBAN stream independent: muting the
  local speakers does not mute VBAN if the bus is still fed.
- If the stream is silent, check the bus level meter in VoiceMeeter before
  touching Lyrion.

---

## 6. Playing the stream

### 6.1 URL formats

```
vban://<sender-ip>/<streamname>
vban://192.168.178.79/Stream1
vban://192.168.178.79:6980/Stream1
vban://239.255.42.42/Stream1        (multicast group, see section 8)
```

`host` is the sender's unicast address for normal streams. When it is a
multicast group (`224.0.0.0/4`), the receiver joins that group instead. `port` is
the UDP port to listen on; the stream name must match the VBAN stream name.
Defaults are `127.0.0.1`, port `6980` and stream `Stream1`.

### 6.2 Radio → Play from URL

Use **Radio → Play from URL**, paste the URL and press play. To keep it:
add it as a **favourite** so it appears in the Radio menu.

### 6.3 Apps menu

The plugin registers an app: **Apps → VBAN → Stream1**. That entry builds the
URL from the settings page, so you do not have to type it.

### 6.4 Settings page

**Settings → Plugins → VBAN** contains:

| Setting | Meaning |
|---------|---------|
| Sender host | IP of the VoiceMeeter machine (used by the Apps entry) |
| Port | UDP port (default 6980) |
| Stream name | VBAN stream name (default Stream1) |

The settings page only feeds the Apps entry; a manually entered
`vban://` URL always works on its own.

---

## 7. Track info (now playing)

VBAN carries audio and text in separate packets. The receiver understands the
VBAN TEXT sub-protocol and the plugin shows the title on the player.

### 7.1 VBAN TEXT via the included script

Run `tools/smtc-sendtext.ps1` on the machine that plays the audio (the VBAN
sender). It reads the Windows "now playing" information, which works with
MediaMonkey, Spotify, browsers and other players that report to the System Media
Transport Controls:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tools\smtc-sendtext.ps1 -Target 192.168.178.30 -Stream Stream1
```

Start it automatically at logon (hidden):

```powershell
schtasks /create /tn "VBAN now playing" /tr "powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File C:\path\to\smtc-sendtext.ps1 -Target 192.168.178.30 -Stream Stream1" /sc onlogon
```

Notes:

- Use **Windows PowerShell 5.1** (`powershell.exe`), not PowerShell 7: the WinRT
  media API is not available there.
- The stream name must match the audio stream (`Stream1`).
- The receiver writes `C:\ProgramData\Lyrion\vban-title-<stream>.txt`; the plugin
  polls it every 2 seconds. Without a sender the player keeps showing the stream
  name.

### 7.2 Via Lyrion's CLI

Any sender can push a title with the CLI command:

```
vban title Artist|Title|Album
```

The fields are separated by `|`; the last two may be empty. This is what the
companion MediaMonkey add-on uses. From PowerShell:

```powershell
$c = New-Object Net.Sockets.TcpClient
$c.Connect('192.168.178.30', 9090)
$w = New-Object IO.StreamWriter($c.GetStream())
$w.WriteLine('vban title Artist|Title|Album'); $w.Flush()
$c.Close()
```

---

## 8. Multiple players and multicast

Every player that plays the URL gets its own pipeline and therefore its own
`vban_url_receptor.exe`, which binds UDP 6980 on the Lyrion host.

- **Synced players** share one stream: Lyrion streams to the sync master, so a
  sync group needs only one receiver and works with plain unicast.
- **Independent players** cannot share a unicast stream (one socket per port).
  Two options:
  - Send a separate VoiceMeeter outgoing stream per player on its own port, for
    example `Stream1` on 6980 and `Stream2` on 6981, and play
    `vban://<sender-ip>:6981/Stream2` on the second player.
  - Use **multicast**: point VoiceMeeter at a multicast group such as
    `239.255.42.42` and play `vban://239.255.42.42/Stream1`. The receiver joins
    the group and every pipeline on the host receives its own copy.

Multicast notes:

- The switch/router must pass multicast (IGMP snooping/querier on managed
  switches; unmanaged switches usually flood it to all ports).
- Allow inbound UDP 6980 in Windows Firewall (same rule as above); some networks
  also need IGMP allowed.
- The receiver joins the group on the default multicast interface. On a
  multi-homed host, make sure the LAN interface carries the default route, or add
  `--interface=<lan-ip>` to the receiver command.
- VoiceMeeter must accept the multicast address as the target. If it does not,
  use the separate-stream/port option.

---

## 9. Sample rates in detail

Lyrion's `pcm` stream format has no container, so the rate has to be agreed
out-of-band. The plugin passes the rate Lyrion announces to the player to the
receiver with `--target-rate`, and the receiver resamples the incoming VBAN
stream to that rate (linear interpolation, clamped 0.5–2.0).

| VBAN in | Player rate | Receiver | Result |
|---------|-------------|----------|--------|
| 44100 | 44100 | pass-through | correct |
| 48000 | 44100 | 48000 → 44100 | correct, pitch preserved |
| 48000 | 48000 | pass-through | correct |

The receiver measures the real input rate once per second. The VBAN header rate
is used as the stable reference; the ratio is only corrected after the first
5 seconds and only when the measured rate deviates more than 5% from the header
(some senders label the wrong rate). Small corrections are deliberately not
applied, because they make the pitch wobble.

If the audio is too fast or too high, read section 5.3 first.

---

## 10. Diagnostics

### 10.1 Stats file

The receiver writes a small status file once per second:

```
C:\ProgramData\Lyrion\vban-stats-Stream1.txt
```

```
inputRate=47938.1
ratio=1.0884
targetRate=44100
headerRate=48000
datagrams=3744
frames=385529
duration=8.04
```

| Field | Meaning |
|-------|---------|
| inputRate | measured frames per second arriving over UDP |
| ratio | input frames per output frame (1.0 = pass-through) |
| targetRate | rate the receiver writes, equal to what the player is told |
| headerRate | rate in the VBAN header |
| datagrams / frames | counters since the stream started |
| duration | seconds since the receiver started |

Expected combinations:

- 44100 in, 44100 target: `ratio=1.0000`, `headerRate=44100`
- 48000 in, 44100 target: `ratio=1.0884`, `headerRate=48000`

If `ratio` drifts away from these values while the source is steady, something
is wrong with the sender (dropped packets, changing rate).

### 10.2 Copy the output to a file (`--tee`)

The receiver can write its PCM output to a file as well, for analysis:

```
vban_url_receptor.exe --target-rate=44100 --tee=C:\temp\vban.pcm "vban://192.168.178.79/Stream1"
```

This is a diagnostic option; the file grows at about 176 KB/s. Write errors are
ignored on purpose so a full disk cannot stop the audio.

### 10.3 Logs

- `C:\ProgramData\Lyrion\Logs\server.log` — Lyrion's log.
- Useful loggers: `plugin.VBAN`, `player.source`, `player.streaming`
  (**Settings → Advanced → Logging**). Set them back to ERROR when you are done.
- The receiver logs its own lines to stderr, which Lyrion captures in
  `server.log` for the pipeline.

### 10.4 Version check

- Plugin version: **Settings → Plugins → VBAN**, or `install.xml`.
- Receiver version:

  ```powershell
  & 'C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\Bin\MSWin32-x64-multi-thread\vban_url_receptor.exe' --version
  ```

---

## 11. Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| Silence; `server.log` shows a pipeline error | Firewall (inbound UDP 6980) or the VoiceMeeter stream is Off. |
| Player shows `play` but no audio; log shows `Buffering... 0 / 153600` and `Read to end of file or pipe` | A stale `vban_url_receptor.exe` still owns UDP 6980. End it with `Get-Process vban_url_receptor -ErrorAction SilentlyContinue \| Stop-Process -Force`, then play again. |
| Audio too fast/slow or wrong pitch | Section 5.3: source rate must equal the engine rate. If that is correct, the player is told a different rate than the receiver produces (outdated `custom-convert.conf`); reinstall the plugin. |
| Clicks or dropouts | Use a wired network, one stream per player, and check `gaps` in the receiver stats. |
| Second independent player stays silent | Unicast cannot be shared; use multicast or a second stream/port (section 8). |
| Plugin disappears after a Lyrion restart | Lyrion removes plugins that are installed but not registered as enabled. Install through the repository, or enable VBAN under Settings → Plugins before restarting. |
| Plugin not in the plugin list | `install.xml` must be in `...\InstalledPlugins\Plugins\VBAN\`; restart Lyrion and check **Settings → Information** for load errors. |
| `vban_url_receptor.exe` blocked by SmartScreen | `Unblock-File` the binary, or right-click → Properties → Unblock. |
| Track title stays empty | No VBAN TEXT sender running, or the stream name does not match (section 7). |
| Title shows only the track name, no artist/album | The sender did not provide those fields; the plugin shows what it receives. |

---

## 12. Limitations

- **Windows only.** The upstream plugin still works on Linux.
- **Live pass-through.** The receiver does not buffer ahead, so a very unstable
  network can cause short dropouts; silence fill keeps the stream alive.
- **One receiver per UDP port.** Independent players need multicast or separate
  streams/ports.
- **PCM 16 bit stereo output.** Higher bit depths and multichannel sources are
  converted down; channels 1 and 2 are used.

---

## 13. Uninstall

1. **Settings → Plugins → VBAN → Uninstall** (repository install), or stop
   Lyrion and delete `...\InstalledPlugins\Plugins\VBAN\`.
2. Restart Lyrion.
3. Optionally delete `C:\ProgramData\Lyrion\vban-stats-*.txt`,
   `vban-title-*.txt` and `vban-meta-*.txt`.
4. Remove the firewall rule if you no longer need it:

   ```powershell
   Remove-NetFirewallRule -DisplayName 'VBAN receptor (UDP 6980)'
   ```

---

## 14. Credits and license

- Windows port, plugin and Go receiver: **pjzzzz** (2026).
- Original plugin: **Markus Buschhoff**, `vban-squeeze`
  (<https://github.com/markusbuschhoff/vban-squeeze>), based on WaveInput by
  Bryan Alton.
- License: GPL-3.0. The original plugin is distributed under the same terms as
  Perl itself.

Source, releases and issue tracker:
<https://github.com/pjzzzz/vban-squeeze-win>
