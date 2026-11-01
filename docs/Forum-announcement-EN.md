# Forum announcement (EN) — ready to paste

> Paste the text below into a new thread in **3rd Party Software** on
> forums.lyrion.org. Suggested title:
>
> **ANNOUNCE: VBAN plugin (Windows) — play VoiceMeeter audio on your Squeezebox**
>
> There are three variants:
>
> - `Forum-announcement-EN.html` — open in a browser, select all, copy, paste
>   into the forum editor (rich paste).
> - `Forum-announcement-EN-body.html` — the same content as a bare HTML
>   fragment, for the editor's **Source** (`</>`) mode.
> - this file — plain text, for places that do not accept HTML.
>
> The text below is plain text on purpose: it pastes cleanly into the forum
> editor. Bold the section headers afterwards if you like. Attach or link
> `docs/VBAN-Howto-EN.md` (or its PDF) for the full manual.

---

ANNOUNCE: VBAN plugin (Windows) — play VoiceMeeter audio on your Squeezebox

TL;DR: if you run VoiceMeeter on a Windows PC, this plugin puts that audio on any Squeezebox or Squeezelite player through Lyrion — no extra hardware, no second audio path.

This is a Windows port of vban-squeeze by Markus Buschhoff. The original is Linux-only because it depends on the Linux `vban_receptor` binary; this port ships a native Windows receiver instead.

Typical use: you already run VoiceMeeter on a Windows PC (radio, mixer, ASIO sources, a microphone, whatever) and you want that mix on the Squeezebox in another room — without extra hardware and without a second audio path.

How it works

  VoiceMeeter  --VBAN/UDP-->  Lyrion (Windows)  --PCM-->  Squeezebox
                             vban_url_receptor.exe

The plugin ships a small native Windows receiver (Go, no runtime needed). It listens for VBAN on UDP, converts 8/16/24/32-bit and float samples to 16-bit stereo PCM, and hands the stream to Lyrion's normal transcoder pipeline. Lyrion then treats it like any other radio stream.

Features

- Native Windows receiver, no Perl wrapper and no Linux binary
- Plays on Squeezebox and Squeezelite players
- Entry in Apps and in the Radio menu (Settings -> Plugins -> VBAN for host/port/stream)
- Output rate follows the rate Lyrion announces to the player, so player and stream can never disagree
- Multicast support for several independent players (vban://239.255.42.42/Stream1)
- Silence fill: when the VBAN source goes away the stream keeps running and resumes by itself when audio returns
- Track info (title/artist) on the player, via VBAN TEXT or Lyrion's CLI
- Switching between VBAN and a normal radio stream works without restarting Lyrion

Requirements

- Lyrion Music Server on Windows (7.9 or newer; tested with 9.1.1)
- VoiceMeeter Banana / Potato / VoiceMeeter 8 as the VBAN sender
- Any Squeezebox or Squeezelite player that supports raw PCM
- Inbound UDP 6980 open in Windows Firewall on the Lyrion host

Installation

1. Settings -> Plugins -> Additional Repositories, add:

   https://raw.githubusercontent.com/pjzzzz/vban-squeeze-win/main/vban.xml

2. Install "VBAN" from the plugin list and restart Lyrion.

3. Allow inbound UDP 6980 (run once in an elevated PowerShell):

   New-NetFirewallRule -DisplayName 'VBAN receptor (UDP 6980)' -Direction Inbound -Protocol UDP -LocalPort 6980 -Action Allow -Profile Any

Manual installation (zip from the Releases page) is described in the README. Direct download:
https://github.com/pjzzzz/vban-squeeze-win/releases/latest

VoiceMeeter setup

- VBAN button (top right) -> Outgoing Streams -> pick the bus you want
- IP address: the Lyrion server
- Sample rate: the SAME value as the VoiceMeeter engine (menu -> System Settings -> Sample Rate)
- Channels: 2, Format: PCM 16 bits, Stream name: Stream1
- Switch the stream On and VBAN On

Then play this URL on a player (Radio -> Play from URL, or add it as a favourite):

   vban://<sender-ip>/Stream1
   vban://192.168.178.79/Stream1

The sender IP is shown at the top of the VoiceMeeter VBAN window.

One important gotcha: sample rates

VoiceMeeter's ASIO driver runs at the engine rate. The program that feeds VoiceMeeter must output that same rate, otherwise VoiceMeeter converts it internally and the audio can come out roughly 8.8% too fast and too high (48000 vs 44100). If that happens, set the source output rate to the engine rate. For MediaMonkey 5: Tools -> Options -> Output Plug-ins -> ASIO output -> Configure -> Sample rate 48000 (when the engine is 48000). This was the single most confusing problem while building this, so it is worth checking first.

Track info

The receiver understands VBAN TEXT and shows the title on the player. Two ways to feed it:

- tools/smtc-sendtext.ps1 (included): reads the Windows "now playing" information and sends VBAN TEXT. Works with MediaMonkey, Spotify, browsers and other players that report to the Windows media controls.
- A small companion add-on for MediaMonkey 5 can push the title through Lyrion's CLI (vban title <text>) and needs no extra process: [mm5-addon source](https://github.com/pjzzzz/vban-squeeze-win/tree/main/mm5-addon) | [direct .mmip download](https://github.com/pjzzzz/vban-squeeze-win/raw/main/mm5-addon/MMSync-Lyrion-Bridge.mmip).

Multiple players

- Synced players share one stream: no extra configuration.
- Independent players: either send a second VoiceMeeter stream on another port (vban://<sender-ip>:6981/Stream2), or use multicast (vban://239.255.42.42/Stream1) with IGMP snooping enabled on the switch.

Troubleshooting (quick)

- Silence and a pipeline error in server.log: firewall (UDP 6980) or the VoiceMeeter stream is Off.
- "Buffering... 0 / 153600" and "Read to end of file or pipe": a stale vban_url_receptor.exe still owns the port; end it with Get-Process vban_url_receptor | Stop-Process and play again.
- Audio too fast or too high: check the sample-rate gotcha above.
- Plugin not in the list: install.xml must be in Cache\InstalledPlugins\Plugins\VBAN\; restart Lyrion.

Diagnostics

The receiver writes C:\ProgramData\Lyrion\vban-stats-Stream1.txt once per second:

   inputRate=47938.1
   ratio=1.0884
   targetRate=44100
   headerRate=48000

inputRate is what arrives, targetRate is what the player gets, ratio is the conversion between them. If targetRate and the player's rate differ, something is wrong in the installation (usually an outdated custom-convert.conf). A full how-to with all settings, multicast, diagnostics and troubleshooting is attached / linked below.

Links

- Source and releases: https://github.com/pjzzzz/vban-squeeze-win
- Full manual (EN): see the attached how-to
- Original Linux plugin by Markus Buschhoff: https://github.com/markusbuschhoff/vban-squeeze
- License: GPL-3.0 (original plugin: same terms as Perl)

When reporting a problem, please include: plugin version, Lyrion version and OS, player model, whether you use unicast or multicast, and the relevant lines from server.log plus the stats file. That makes it much easier to help.

Feedback and test reports are welcome. Windows-only for now; the upstream Linux plugin still works on Linux.
