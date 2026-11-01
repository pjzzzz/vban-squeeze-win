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
| Stream format | **44100 Hz**, **16-bit PCM**, **2 channels** |
| Player | Any Squeezebox or Squeezelite player (raw PCM support) |

Lyrion's `pcm` stream format is fixed at 44100 Hz / 16-bit / stereo. The
receiver converts 8/16/24/32-bit and float samples to 16-bit stereo, but it
cannot change the sample rate. Set the VBAN outgoing stream to **44100 Hz**.

## Quick start

1. Lyrion runs on Windows (7.9+).
2. Plugin installed and **VBAN** enabled under **Settings → Plugins**.
3. Inbound UDP 6980 allowed in Windows Firewall.
4. VoiceMeeter sends a VBAN stream to the Lyrion IP: 44100 Hz, 16-bit PCM,
   2 channels, name `Stream1`.
5. Play `vban://<sender-ip>/Stream1` on a Squeezebox.

## Installation

### Manual installation

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

5. Allow inbound VBAN traffic (UDP 6980) in Windows Firewall:

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

6. Start Lyrion Music Server.
7. Open **Settings → Plugins** and confirm that **VBAN** is enabled.

### Installation through the plugin repository

Add this URL under **Settings → Plugins → Additional Repositories**:

```
https://raw.githubusercontent.com/pjzzzz/vban-squeeze-win/main/vban.xml
```

Then install **VBAN** from the plugin list and restart Lyrion.

The repository XML intentionally omits the `sha` field: the zip is fetched from
GitHub Releases over HTTPS, and a hash pinned in the repository would go stale
as soon as a new binary is built.

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
```

`host` and `port` are informational: the receiver listens on all interfaces and
selects packets by stream name, exactly like the Linux `vban_receptor`.
Defaults are `127.0.0.1`, port `6980` and stream `Stream1`.

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| Silence, `server.log` shows a pipeline error | Check the firewall rule (inbound UDP 6980) and that VoiceMeeter's stream is **On**. |
| Plugin does not appear in the plugin list | `install.xml` must be at `C:\ProgramData\Lyrion\Cache\InstalledPlugins\Plugins\VBAN\install.xml`; restart Lyrion; check **Settings → Information** for plugin load errors. |
| Audio plays too fast/slow or at the wrong pitch | The VBAN stream is not 44100 Hz. Set it to 44100 in VoiceMeeter. The receiver logs an error on stderr. |
| Clicks or dropouts | Use a wired network, keep the VBAN stream on a single player, and check for packet loss (`gaps` in the receiver stats). |
| Second player stays silent | Two players on the same stream share UDP port 6980; Windows delivers each datagram to only one socket. Use one player per stream, or send a second VBAN stream on another port: `vban://<sender-ip>:6981/Stream2`. |
| `vban_url_receptor.exe` blocked by SmartScreen | Run `Unblock-File` on the binary (see installation), or right-click the file → **Properties** → **Unblock**. |

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
- The plugin validates the `vban://` URL before it is placed in a transcoder
  command line and refuses host or stream names containing shell
  metacharacters.
- Releases include a `SHA256SUMS.txt`; verify the zip before installing:

  ```powershell
  Get-FileHash .\vban-1.0.0.zip -Algorithm SHA256
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

.\tools\make-zip.ps1 -Version 1.0.0
```

The plugin zip is written to `dist\vban-1.0.0.zip`. GitHub Actions builds the
same artifact on every push and attaches it to tagged releases.

## Limitations

- **Windows only.** The Linux plugin from upstream still works on Linux.
- **44100 Hz only** (Lyrion's `pcm` format). Other sample rates are passed
  through unchanged and play at the wrong pitch; the receiver logs an error.
- **One player per stream.** Two pipelines cannot both receive the same UDP
  port; see the troubleshooting table.
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
