# Changelog

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
