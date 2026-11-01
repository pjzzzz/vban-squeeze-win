# Changelog

## 1.3.1 — 2026-11-01

- Apply a known title immediately when a pipeline starts, so the player does not
  briefly show the stream name.

## 1.3.0 — 2026-11-01

- Track info: the receiver parses VBAN TEXT packets (sub-protocol 0x40) and
  writes the title to `%TEMP%\vban-<stream>.title`; the plugin polls it and
  publishes the title to Lyrion (`setCurrentTitle`), so the player shows the
  current track.
- `tools/smtc-sendtext.ps1`: reads the Windows now-playing info (SMTC) and
  sends it as VBAN TEXT. Works with MediaMonkey, Spotify, browsers, etc.
## 1.2.2 — 2026-11-01

- The receiver keeps the PCM stream continuous: when no VBAN packets arrive it
  writes 100 ms blocks of silence, so Lyrion and the player never mark the
  stream as failed. Audio resumes by itself when the sender returns.
## 1.2.1 — 2026-11-01

- The pipeline no longer uses socketwrapper's `-w` stream check, so the
  receiver stays alive when the VBAN source is temporarily silent. Playback
  resumes when VoiceMeeter starts sending again, without pressing play.
- UI: the plugin is registered as an app (Apps → VBAN) and keeps its Radio
  menu entry; Settings → Plugins → VBAN configures sender IP, port and stream.
## 1.1.0 — 2026-11-01

- Multicast support: when the URL host is a multicast group (for example
  `vban://239.255.42.42/Stream1`), the receiver binds with `SO_REUSEADDR` and
  joins the group. Several players on the same Lyrion host can then play the
  same VBAN stream independently: every pipeline gets its own copy.
- Documentation: multicast setup (VoiceMeeter target, firewall, IGMP) and a
  clearer explanation of unicast vs. synced players.

## 1.0.1 — 2026-11-01

- Fix: the player stayed in `waitingToPlay` and the pipeline ended with
  `Buffering... 0 / ...` and `Read to end of file or pipe`. `scanUrl` no longer
  hands `vban://` URLs to `Slim::Utils::Scanner::Remote` (which tried to fetch
  them over HTTP and never called back); it reports the track as a live stream
  directly.
- Fix: a stale `vban_url_receptor.exe` holding UDP 6980 makes a new pipeline
  receiver fail to bind. Documented in the troubleshooting table.

## 1.0.0 — 2026-11-01

- First Windows port of vban-squeeze: Go receiver (`vban_url_receptor.exe`),
  Lyrion plugin with `vban://` protocol handler, packaging and CI.
