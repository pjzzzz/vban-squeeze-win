# vban-squeeze-win

Windows port of [vban-squeeze](https://github.com/markusbuschhoff/vban-squeeze):
play **VBAN** audio streams from VoiceMeeter on your **Squeezebox** devices
through [Lyrion Music Server](https://lyrion.org/) running on Windows.

The original plugin is Linux-only because it depends on the Linux
`vban_receptor` binary and a Perl wrapper. This port ships a native Windows
receiver (`vban_url_receptor.exe`) that speaks the VBAN protocol, writes raw
PCM to stdout, and plugs into Lyrion's existing transcoder pipeline.

```
VoiceMeeter (Windows)                 Lyrion Music Server (Windows)
┌────────────────────┐   VBAN/UDP    ┌──────────────────────────────────────┐
│ Outgoing Stream 1  │ ────────────► │ vban_url_receptor.exe                │
│ 44100 / 16-bit / 2 │    port 6980  │   │ raw PCM (stdout)                 │
└────────────────────┘               │   ▼                                  │
                                     │ socketwrapper.exe → Pipeline         │
                                     │   │                                  │
                                     │   ▼                                  │
                                     │ Squeezebox / Squeezelite player      │
                                     └──────────────────────────────────────┘
```

## Requirements

| Component | Requirement |
|-----------|-------------|
| Lyrion Music Server | Windows, 7.9 or newer (tested with 9.1.1) |
| VBAN sender | VoiceMeeter Banana / Potato / VoiceMeeter 8 |
| Stream format | 16-bit PCM, 2 channels, **44100 Hz** (recommended) |
| Player | Any Squeezebox or Squeezelite player (raw PCM support) |

Lyrion's `pcm` stream format is fixed at 44100 Hz / 16-bit / stereo. The
receiver converts 8/16/24/32-bit and float samples to 16-bit stereo and
resamples any input rate to 44100 (for example when VoiceMeeter's engine runs
at 48000 Hz). Linear interpolation is used; 44100 Hz input passes through
unchanged.

## Quick start

1. Lyrion runs on Windows (7.9+).
2. Plugin installed and **VBAN** enabled under **Settings → Plugins**.
3. Inbound UDP 6980 allowed in Windows Firewall.
4. VoiceMeeter sends a VBAN stream to the Lyrion IP: 44100 Hz, 16-bit PCM,
   2 channels, name `Stream1`.
5. Play `vban://<sender-ip>/Stream1` on a Squeezebox.

## Installation

The repository install is the recommended route: Lyrion then registers the
plugin as installed, enables it, and can update it later. Use the manual route
only when you cannot add a repository (for example while testing a private
build).

### Installation through the plugin repository (recommended)

Add this URL under **Settings → Plugins → Additional Repositories**:

```
https://raw.githubusercontent.com/pjzzzz/vban-squeeze-win/main/vban.xml
```

Then install **VBAN** from the plugin list and restart Lyrion.

The repository XML intentionally omits the `sha` field: the zip is fetched from
GitHub Releases over HTTPS, and a hash pinned in the repository would go stale
as soon as a new binary is built.

### Firewall (both install routes)

Allow inbound VBAN traffic (UDP 6980) in Windows Firewall:

```powershell
New-NetFirewallRule -DisplayName 'VBAN receptor (UDP 6980)' `
    -Direction Inbound -Protocol UDP -LocalPort 6980 `
    -Action Allow -Profile Any
```

`-Profile Any` also covers the Public profile; if you prefer to restrict it,
use `-Profile Private,Domain` and make sure the server's network is not
classified as Public. For extra hardening, limit the rule to the sending
machine: add `-RemoteAddress 192.168.178.79`.

Without this rule the receiver starts, but no packets ever arrive.

### Manual installation (fallback)

1. Download `vban-<version>.zip` from the
   [Releases](https://github.com/pjzzzz/vban-squeeze-win/releases) page.
2. Stop Lyrion Music Server.
3. Extract the zip so that you end up with:
   `C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\install.xml`
   If Lyrion is installed elsewhere, use that installation's
   `Cache\InstalledPlugins\Plugins\VBAN\` folder instead (for example
   `D:\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\`).
4. Unblock the receiver binary (it was downloaded from the internet):

   ```powershell
   Unblock-File 'C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\Bin\MSWin32-x64-multi-thread\vban_url_receptor.exe'
   ```

5. Start Lyrion Music Server.
6. Open **Settings → Plugins** and confirm that **VBAN** is enabled.

Also apply the firewall rule above if you have not done so yet.

## VoiceMeeter setup

1. Open VoiceMeeter and click the **VBAN** button (top right).
2. Under **Outgoing Streams** pick the bus you want to stream (A1…A5, B1…B3).
3. Set:
   - **IP address**: the IP of the Lyrion server (for example `192.168.178.30`)
   - **Sample rate**: `44100`
   - **Channels**: `2`
   - **Format**: `PCM 16 bits`
   - **Stream name**: `Stream1` (or any name, must match the URL)
4. Switch the stream **On**, and switch **VBAN On** (top left).

The IP address of the sending machine is shown at the top of the VBAN window;
you need it for the playback URL.

## Playing the stream

In Lyrion, use **Radio → Play from URL** (or add it as a favorite):

```
vban://<sender-ip>/<streamname>
vban://192.168.178.79/Stream1
vban://192.168.178.79:6980/Stream1
vban://239.255.42.42/Stream1        (multicast group, see below)
```

`host` is the sender's unicast address for normal streams. When it is a
multicast group (`224.0.0.0/4`), the receiver joins that group instead; see
[Multiple players and multicast](#multiple-players-and-multicast). `port` is
the UDP port to listen on and the stream name must match the VBAN stream.
Defaults are `127.0.0.1`, port `6980` and stream `Stream1`.

## Multiple players and multicast

Every player that plays the URL gets its own pipeline and therefore its own
`vban_url_receptor.exe`, which binds UDP 6980 on the Lyrion host.

- **Synced players** share one stream: Lyrion streams to the sync master, so a
  sync group needs only one receiver and works with plain unicast.
- **Independent players** cannot share a unicast stream (one socket per port).
  Two options:
  - Send a separate VoiceMeeter outgoing stream per player on its own port,
    for example `Stream1` on 6980 and `Stream2` on 6981.
  - Use **multicast**: point VoiceMeeter at a multicast group such as
    `239.255.42.42` and play `vban://239.255.42.42/Stream1`. The receiver joins
    the group, and every pipeline on the host receives its own copy of the
    stream.

Multicast notes:

- The switch/router must pass multicast (IGMP snooping/querier on managed
  switches; simple unmanaged switches usually flood it to all ports).
- Allow inbound UDP 6980 in Windows Firewall (the same rule as above); some
  networks also need IGMP allowed.
- The receiver joins the group on the default multicast interface. On a
  multi-homed host, make sure the LAN interface carries the default route.
- VoiceMeeter must accept the multicast address as the target. If it does not,
  use the separate-stream/port option.

Quick start (multicast):

1. Pick a group and port, for example `239.255.42.42` on `6980`.
2. In VoiceMeeter set the outgoing stream's IP to `239.255.42.42` (44100 Hz,
   2 channels, PCM 16 bits, stream name `Stream1`).
3. Play `vban://239.255.42.42/Stream1` on two players.
4. If nothing arrives, check IGMP on the switch and the firewall rule. On a
   multi-homed Lyrion host, add `--interface=<lan-ip>` to the receiver command
   so it joins the group on the right NIC.

Glossary: *multicast group* — one address that many receivers listen to;
*IGMP snooping* — a switch feature that forwards multicast only to interested
ports; *sync master* — the player in a sync group that Lyrion streams to.

## Track info (now playing)

VBAN carries audio and text in separate packets. The receiver understands the
VBAN TEXT sub-protocol and the plugin shows the title on the player.

Run `tools/smtc-sendtext.ps1` on the machine that plays the audio (the VBAN
sender). It reads the Windows "now playing" information, which works with
MediaMonkey, Spotify, browsers and other players that report to the System
Media Transport Controls:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\tools\smtc-sendtext.ps1 -Target 192.168.178.30 -Stream Stream1
```

Start it automatically at logon (hidden):

```powershell
schtasks /create /tn "VBAN now playing" /tr "powershell.exe -NoProfile -WindowStyle Hidden -ExecutionPolicy Bypass -File C:\path\to\smtc-sendtext.ps1 -Target 192.168.178.30 -Stream Stream1" /sc onlogon
```

Notes:

- Use **Windows PowerShell 5.1** (`powershell.exe`), not `pwsh`: the WinRT media
  API is not available in PowerShell 7.
- **MediaMonkey users** do not need this script: the MMSync addon can send the
  title directly through Lyrion's CLI (`vban title <text>`).
- The stream name must match the audio stream (`Stream1`).
- The receiver writes `C:\ProgramData\Lyrion\vban-title-<stream>.txt`; the
  plugin polls it every 2 seconds and updates the player. Without the sender the
  player keeps showing the stream name.

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| Silence, `server.log` shows a pipeline error | Check the firewall rule (inbound UDP 6980) and that VoiceMeeter's stream is **On**. |
| Player shows `play` but no audio; log shows `Buffering... 0 / 153600` and `Read to end of file or pipe` | A stale `vban_url_receptor.exe` still owns UDP 6980, so the new pipeline receiver cannot bind. End it with `Get-Process vban_url_receptor -ErrorAction SilentlyContinue \| Stop-Process -Force`, then play again. |
| Plugin disappears after a Lyrion restart | Lyrion's extension manager removes plugins that are installed but not registered as enabled. Install through the repository, or make sure **VBAN** is enabled under **Settings → Plugins** before restarting. |
| Plugin does not appear in the plugin list | `install.xml` must be at `C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\install.xml`; restart Lyrion; check **Settings → Information** for plugin load errors. |
| Audio plays too fast/slow or at the wrong pitch | The VBAN stream is not 44100 Hz. Set it to 44100 in VoiceMeeter. The receiver logs an error on stderr. |
| Clicks or dropouts | Use a wired network, keep the VBAN stream on a single player, and check for packet loss (`gaps` in the receiver stats). |
| Second player stays silent | Unicast: two pipelines cannot share UDP 6980. Use multicast (`vban://239.255.42.42/Stream1`) or a second stream on another port (`vban://<sender-ip>:6981/Stream2`). Synced players share one stream and are not affected. |
| `vban_url_receptor.exe` blocked by SmartScreen | Run `Unblock-File` on the binary (see installation), or right-click the file → **Properties** → **Unblock**. |
| Another app takes over playback (old URL returns) | Another controller (for example the MMSync/MediaMonkey bridge) sends a play command on track change. Disable its bridge/toggle in that app, or use a separate player for VBAN. |

Lyrion writes transcoder output and the receiver's stderr to
`C:\ProgramData\Lyrion\Logs\server.log` (the exact `Logs` folder follows your
Lyrion data directory). Enable the `plugin.VBAN` log category
(**Settings → Advanced → Logging**) for plugin messages.

## Testing without Lyrion

The receiver can be tested on its own. On the Lyrion server, in a console:

```powershell
cd 'C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\Bin\MSWin32-x64-multi-thread'
.\vban_url_receptor.exe "vban://192.168.178.79/Stream1" -d test.pcm -t 10 --stats
```

A 10-second test at 44100 Hz / 16-bit / stereo produces about 1,764,000 bytes.
The stats block shows packets, frames, peak level, gaps and the detected format.

No VoiceMeeter at hand? Use the included test sender from a checkout:

```powershell
.\tools\sendtest.ps1 -Target 127.0.0.1 -Port 6980 -Stream Stream1 -Seconds 5
```

`-d/--device` writes to a file and is meant for manual diagnostics on the
console; the Lyrion plugin never passes it. `--allow=IP` restricts the receiver
to packets from the given source address(es).

## Security notes

VBAN itself has no authentication, and the receiver has to listen on all
interfaces to receive the sender's packets. Mitigations:

- Keep the firewall rule scoped to your LAN and, ideally, to the sender's IP.
- Optionally start the receiver with `--allow=<sender-ip>`.
- With multicast, any local process that joins the group can receive the
  stream; use `--allow` and keep the LAN trusted.
- The plugin validates the `vban://` URL before it is placed in a transcoder
  command line and refuses host or stream names containing shell
  metacharacters.
- Releases include a `SHA256SUMS.txt`; verify the zip before installing:

  ```powershell
  Get-FileHash .\vban-1.3.4.zip -Algorithm SHA256
  ```

## Building from source

Requirements: Go 1.22 or newer, PowerShell 7.

```powershell
git clone https://github.com/pjzzzz/vban-squeeze-win.git
cd vban-squeeze-win

go test ./...
go vet ./...

go build -trimpath -ldflags "-s -w" `
    -o plugin/VBAN/Bin/MSWin32-x64-multi-thread/vban_url_receptor.exe ./receiver

.\tools\make-zip.ps1 -Version 1.3.4
```

The plugin zip is written to `dist\vban-1.3.4.zip`. GitHub Actions builds the
same artifact on every push and attaches it to tagged releases.

## Limitations

- **Windows only.** The Linux plugin from upstream still works on Linux.
- **Sample rate**: Lyrion's `pcm` format is 44100 Hz. Other input rates are
  resampled to 44100 (linear interpolation).
- **Independent players need multicast or separate streams.** Synced players
  share one stream; with unicast, two pipelines cannot both bind UDP 6980.
- The receiver is a live pass-through: it does not buffer ahead, so a very
  unstable network can cause clicks.

## License and credits

This project is licensed under the **GNU General Public License v3.0**; see
[LICENSE](LICENSE) and [NOTICE](NOTICE).

- [vban-squeeze](https://github.com/markusbuschhoff/vban-squeeze) — © 2017
  Markus Buschhoff (plugin structure, `vban://` URL scheme, transcoder rules)
- [vban](https://github.com/quiniouben/vban) — © 2015-2026 Benoît Quiniou
  (VBAN protocol implementation and packet layout)
- WaveInput — © 2008 Bryan Alton and others (plugin skeleton)

The Perl plugin files remain available under the same terms as Perl itself.
