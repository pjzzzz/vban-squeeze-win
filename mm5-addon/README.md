# MMSync — Lyrion Audio Bridge (MediaMonkey 5 add-on)

A small companion add-on for **MediaMonkey 5** on the Windows PC that feeds
VoiceMeeter. It does two independent things:

1. **Player bridge** (optional): on play / pause / stop in MediaMonkey it sends
   the matching command to a Lyrion player and points it at the swyh-rs stream
   URL (`/stream/swyh.rf64` or `.flac`).
2. **Track info** (optional): on every track change it sends
   `artist|title|album` to the VBAN plugin through Lyrion's CLI
   (`vban title`), so the Squeezebox shows the current track while playing
   VBAN audio.

Both switches are independent. **If you use the VBAN plugin, turn the bridge
off and keep only "Send track info to VBAN"** — the bridge would otherwise
switch the player back to the old swyh-rs stream URL on every track change.

## Requirements

- MediaMonkey 5.0 or newer (Windows)
- Lyrion Music Server with the **VBAN** plugin 1.7.x or newer (for the
  `artist|title|album` format)
- The Lyrion JSON-RPC port reachable from the MediaMonkey PC (default 9002)

The add-on talks to Lyrion over plain HTTP on your local network; there is no
TLS and no authentication. Keep the Lyrion port on a trusted network and do not
expose it to the internet.

## Install

1. Download `MMSync-Lyrion-Bridge.mmip` from this folder (or the release assets).
2. In MediaMonkey: **Tools → Options → Add-ons → Add…** and pick the `.mmip`.
3. Restart MediaMonkey.

## Configure

**Tools → Options → Add-ons → MMSync - Lyrion Audio Bridge → Configure**

| Field | Meaning |
|-------|---------|
| Server address | IP or host name of the Lyrion server (for example `192.168.1.10`) |
| Server port | Lyrion web UI / JSON-RPC port, default `9002` |
| Player MAC | MAC address of the player to control, format `AA:BB:CC:DD:EE:FF` |
| Stream URL | swyh-rs stream, `http(s)://host:port/stream/swyh.rf64` (RF64 recommended) |
| Stream name | VBAN stream name, must match the plugin (`Stream1` by default) |

Buttons:

- **Test connection** — asks Lyrion for the player status through JSON-RPC.
- **Test stream** — checks that the stream URL answers.

No server address, MAC or stream URL is pre-filled: nothing is sent until you
configure the add-on.

## Behaviour

| MediaMonkey event | Sent to Lyrion |
|-------------------|----------------|
| play | `playlist play <streamUrl>` |
| pause | `pause` |
| unpause | `play` |
| stop | `stop`, or `power 0` when "Power off the player on stop" is enabled |
| track change | `vban title <artist>\|<title>\|<album>` (if enabled) |

Commands are sent to `http://<server>:<port>/jsonrpc.js` as
`slim.request [<playerMac>, [...]]`. Only one request is in flight at a time;
the last pending command is sent when the previous one returns.

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| Nothing happens on play | Bridge switch off, invalid MAC, or the stream URL does not match `…/stream/swyh.rf64` or `.flac`. |
| "Connection failed" | Wrong address/port, Lyrion not running, or a firewall between the two machines. |
| Player switches back to the old stream while using VBAN | The bridge is still enabled. Turn it off; keep only "Send track info to VBAN". |
| No title on the player | VBAN plugin older than 1.7.x, wrong stream name, or the plugin's title timer not running. Check `C:\ProgramData\Lyrion\vban-title-<stream>.txt` on the Lyrion host. |

## Uninstall

**Tools → Options → Add-ons**, select the add-on and choose **Uninstall**.
Stored settings are removed with it.
