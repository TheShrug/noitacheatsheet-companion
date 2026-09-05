# Security

This app runs in the background on your machine, reads your Noita save files, and sends data to a
website. Here is exactly what that means, so you can decide whether to run it.

## Reporting a vulnerability

Open a [security advisory](https://github.com/TheShrug/noitacheatsheet-companion/security/advisories/new)
— that is private, unlike an issue. If you cannot, open an issue saying only that you have a
security report and asking for a contact; don't put the details in it.

Please include what an attacker would have to control to exploit it, and what they would get. There
is no bounty and no SLA. This is a hobby project maintained by one person, and pretending otherwise
would be worse than saying so.

## What the app can reach

| | |
| --- | --- |
| **Reads** | Your Noita data folder — `player.xml` in the active save, and the gifs in `save_rec/screenshots_animated`. Read-only; the app writes nothing into Noita's folders |
| **Writes** | Its own config and queue, in your user config directory. Nothing else |
| **Network — out** | `noitacheatsheet.com`, and only when you confirm an upload |
| **Network — in** | A review queue on `127.0.0.1` only, port 7331 by default. Never bound to `0.0.0.0`, so nothing outside your machine can reach it. It refuses any request not addressed to loopback on that port, and any state-changing request without a token from a page it served |
| **Privileges** | Runs as you. It does not need and must never ask for administrator or root |

## What gets uploaded, and when

Only when you click upload on a specific clip: **that clip, and the one wand you picked from your
save.** Not the save file, not the seed, not your other wands, not a machine identifier.

There is no background sync and no "silent draft" — those were considered and rejected in
[ADR 2](docs/adr/0002-one-companion-endpoint-and-a-local-queue.md). If you never open the queue,
nothing has left your machine.

**Today, nothing has left it either way.** The upload is not written: the endpoint it needs does not
exist on the site yet. Confirming a clip in the review queue writes a timestamp into the local queue
file and nothing else, and the app opens no outbound connection at all. The consent step was built
first on purpose, so that there is never a version of this app that can upload without one.

## Known limitations, stated plainly

These are real and current. They are here rather than in a changelog because you should know them
before installing, not after.

- **The binaries are unsigned.** Windows SmartScreen will warn about them and the warning is
  correct: nobody has vouched for this file. Verify the SHA-256 against the release's `SHA256SUMS`,
  or build from source — see the README.
- **The localhost review queue is an attack surface.** Anything else running as your user can reach
  a loopback port, and a malicious web page can try to guess it. Five things stand between that and
  your clips, and you can read all of them in `internal/server`:

  - the listener is `127.0.0.1` and a port, written literally, with no flag that can widen it;
  - a request whose `Host` is not `127.0.0.1`, `localhost` or `[::1]` **on that exact port** is
    refused — this is what stops a name an attacker controls from resolving to your machine;
  - anything that changes something must also carry an `Origin` or `Referer` of ours, or neither;
  - and a CSRF token, random per run of the app, which only a page we served can show it;
  - a clip is served by an opaque id for its queue entry, never by a path taken from the request,
    so no request can name a file that is not already queued.

  What that does **not** defend against: another program running as you can fetch that page and
  read the token, exactly as you can. There is no defence against that short of not running this.
- **The app will hold an API token for your account.** Where it is stored and what it is scoped to
  is unresolved. It will be documented here before it exists, not after.
- **The site's upload path has its own weaknesses** that this app routes around rather than fixes —
  notably an anonymous upload endpoint with client-asserted attribution. Tracked on
  [NoitaSpellCasters#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66).

## Dependencies

Currently **none** — `go.mod` lists no modules and the app is stdlib only. That is deliberate: every
dependency is code you are trusting on top of this repo's, and this repo's whole pitch is that you
can read it first.

That will not survive the tray icon, which needs one
([ADR 3](docs/adr/0003-tray-plus-a-localhost-review-queue.md)). When it lands, it will be named
here and in the README, and the app will still run without it via `--no-tray`.
