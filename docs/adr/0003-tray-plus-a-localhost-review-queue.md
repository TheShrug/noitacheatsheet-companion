# 3. A tray icon whose menu opens a review queue served on localhost

**Date:** 2026-09-01
**Status:** Proposed. Neither half is built.

## Context

The app runs in the background while someone plays. It therefore has no window, and a background
process with no window and no icon is one nobody can find, pause, or quit — and one whose only
evidence of existing is that files leave their machine. That is the wrong thing to build for a tool
whose entire pitch is that it can be trusted.

But the queue itself is a real interface: gif thumbnails, the wands parsed from the save beside
each one, a per-row upload button. Building that natively means a GUI toolkit — Fyne pulls in
roughly forty modules — in a repo whose pitch is that a stranger can read it before installing it.

The two requirements pull opposite ways only if the tray has to *render* the queue.

## Decision

**Split them.** The tray is an entry point, not a UI:

```
Open queue      -> opens http://127.0.0.1:7331 in the default browser
Pause watching
Quit
```

The queue is served by the app itself from `net/http` and `html/template`, bound to `127.0.0.1` and
never to `0.0.0.0`. Rendering, thumbnails, and the confirm step are all HTML, written once and
identical on every platform.

The tray costs exactly one dependency and the cgo that comes with it. That is the whole price, it is
confined to one package behind a small interface, and the app must **run headless without it** —
`--no-tray` starts the watcher and the server and prints the URL, which is also how it runs on a
platform whose tray build is broken.

## Consequences

- **cgo, and what it costs.** Cross-compiling stops being free: Windows builds need mingw-w64
  (already installed in the Dockerfile's `base` stage for exactly this), Linux builds link against
  GTK/appindicator and stop being static, and macOS needs a real macOS runner. `CGO_ENABLED=0` in
  the release build has to change when this lands, and the `--no-tray` path is what keeps a broken
  tray from being a broken app.
- **A local HTTP server is an attack surface on the player's machine.** Bound to loopback it is
  reachable by anything else running as that user, and by any web page that can guess the port.
  Before this ships it needs, at minimum: a `Host` header check, a CSRF token on every state-changing
  request, and no cross-origin reads. That is a ticket, and it is not optional.
- Port **7331** is a fixed default, overridable, and printed as the last line on startup — the same
  contract as `make run` everywhere else in the fleet. It is not in the homelab port table, which
  covers apps sharing one server; this one runs on machines we do not own.
- One dependency is not zero. `README.md` says so plainly rather than claiming a purity the repo
  does not have.
