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
| **Network — in** | A review queue on `127.0.0.1` only. Never bound to `0.0.0.0`, so nothing outside your machine can reach it |
| **Privileges** | Runs as you. It does not need and must never ask for administrator or root |

## What gets uploaded, and when

Only when you click upload on a specific clip: **that clip, and the one wand you picked from your
save.** Not the save file, not the seed, not your other wands, not a machine identifier.

There is no background sync and no "silent draft" — those were considered and rejected in
[ADR 2](docs/adr/0002-one-companion-endpoint-and-a-local-queue.md). If you never open the queue,
nothing has left your machine.

## Known limitations, stated plainly

These are real and current. They are here rather than in a changelog because you should know them
before installing, not after.

- **The binaries are unsigned.** Windows SmartScreen will warn about them and the warning is
  correct: nobody has vouched for this file. Verify the SHA-256 against the release's `SHA256SUMS`,
  or build from source — see the README.
- **The localhost review queue is an attack surface.** Anything else running as your user can reach
  a loopback port, and a malicious web page can try to guess it. Before the queue ships it needs a
  `Host` header check and a CSRF token on every state-changing request; that work is tracked in the
  issue queue and the ADR names it as non-optional.
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
