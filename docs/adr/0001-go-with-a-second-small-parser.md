# 1. Go, with a second small parser rather than sharing the site's C#

**Date:** 2026-09-01
**Status:** Proposed — the repository is scaffolded on this assumption, and this ADR is where it is
ratified or rejected. Nothing here is expensive to reverse yet; the parser and the watcher are not
written.

## Context

[NoitaSpellCasters#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66) requires an ADR
settling the stack before any code. Two candidates:

- **.NET.** The site's `player.xml` → wand logic is C# LINQ-to-XML in
  `NoitaCheatSheet/Client/Services/WandService.cs`, producing a `WandXmlModel` against the
  `SpellType`/`Perk` enums. A self-contained .NET binary could reference `Shared` and
  `DatabaseModels` and inherit both for free.
- **Go.** One static binary per platform, no runtime to install, small enough that a stranger can
  read it before running it.

Three things decide it:

1. **`TheShrug/NoitaSpellCasters` is private, and this repo is public.** The .NET case rests
   entirely on referencing `Shared` and `DatabaseModels`, and a public repo cannot reference a
   private one. Taking that path means publishing those projects as packages, or vendoring copies —
   which reintroduces the duplication the .NET case exists to avoid, plus a publishing step.
2. **Players must be able to audit this before installing it.** It runs in the background on their
   machine and reads their save files. A ~70 MB self-contained runtime is not something anyone
   reads; a few hundred lines of stdlib Go is.
3. **The duplication is cheaper than issue #66 assumed.** The issue argued a Go port means
   "keeping two copies correct across Noita updates — the exact drift `make regen-enums` exists to
   stop". That reasoning was borrowed from the enum generator, where it is true: spell and perk
   data really do change when Nolla ships an update. It does not transfer to the wand *shape*.
   Noita 1.0 released in 2020 and the save format has been stable since; `deck_capacity`,
   `actions_per_round`, `shuffle_deck_when_empty` and the rest are not a moving target.

## Decision

**Go, stdlib-first, with a small second parser.** The port covers exactly two things: the save/gif
paths, and the subset of `player.xml` a wand upload needs. Not the enums, not the stats model —
those stay on the server, which is where the enum values are authoritative anyway.

The port is kept honest by three cheap mechanisms rather than by an expensive one:

- A `PARITY:` comment at the top of every ported file naming its C# counterpart, and a matching note
  on the C# side, so neither can be edited without seeing the other exists.
- The [`/parity` skill](../../.claude/skills/parity/SKILL.md), listing the pairs and run as part of
  `/ticket-close` whenever ported code is touched.
- Tests over real save files, once we have some.

**Dependencies are opt-in, not default.** Every module added needs a reason recorded in the ADR or
issue that introduced it. Today there are none.

## Consequences

- Two copies of the path list and the wand parse exist, and nothing but discipline keeps them in
  step. A drifted copy fails **silently** — a wrong wand, or a watcher that sees no clips — which is
  why the parity mechanisms are part of the decision rather than a follow-up.
- The site's enums stay the single source of truth for spell and perk identity, because the app
  never maps them; it sends what it read and the server resolves it. This is what keeps the port
  small, and it is a constraint on [ADR 2](0002-one-companion-endpoint-and-a-local-queue.md)'s
  endpoint shape.
- Cross-compiling is a single `GOOS=... go build` for as long as there is no cgo. The tray in
  [ADR 3](0003-tray-plus-a-localhost-review-queue.md) breaks that, and pays for it there.
- macOS is buildable but unverified — Noita has no macOS build, so it only exists inside a Wine
  bottle whose path nobody can guess. `--root` covers it; nothing else does.
- Reversing this means rewriting the app, not adapting it. The gate is the private/public split
  above: if `Shared` and `DatabaseModels` are ever published as packages, the .NET case gets its
  strongest argument back.
