---
name: parity
description: Check this app's Go copies of the site's logic against the C# they were ported from in TheShrug/NoitaSpellCasters — save paths, the wand parse, and the upload contract. Use before releasing, when a wand or a path comes out wrong, after a Noita update, and as step 4 of /ticket-close whenever ported code was touched.
---

# Parity — keeping the second copy honest

This app re-implements three things the site already implements in C#. That was a deliberate choice
([ADR 1](../../../docs/adr/0001-go-with-a-second-small-parser.md)) and it is a cheap one
*as long as somebody checks*. Nothing enforces it: the two repos build separately, and a drifted
copy produces a wrong wand rather than a failed build.

**The failure mode is silent.** A stale path makes the watcher see no clips, which looks exactly
like the player not having recorded any. A stale parse makes a wand upload with the wrong capacity,
which nobody notices until someone tries to build it.

## The pairs

| Here | There, in `TheShrug/NoitaSpellCasters` |
| --- | --- |
| `internal/noita/paths.go` | `NoitaCheatSheet/Shared/NoitaSavePaths.cs` |
| `internal/noita/wand.go` | `NoitaCheatSheet/Client/Services/WandService.cs`, `ParseSaveFile` |
| the upload request the app sends | the companion endpoint on `NoitaCheatSheet/Server/Controllers/WandsController.cs` |

Keep this table current. A ported thing missing from it is a ported thing nobody will check.

## How to read the other side

That repo is **private**, so there is no raw URL to fetch. Read it through `gh`, which is already
authenticated for the queue:

```bash
gh api repos/TheShrug/NoitaSpellCasters/contents/NoitaCheatSheet/Shared/NoitaSavePaths.cs \
  --jq '.content' | base64 -d
```

A local checkout at `../NoitaSpellCasters` is fine too, but **fetch it first** — a stale working
copy will report parity against a version that no longer exists, which is worse than not checking.

## What counts as drift

Compare *meaning*, not text. The two copies are deliberately not line-for-line:

| Difference | Verdict |
| --- | --- |
| A folder name, an AppID, a save-file name, an XML attribute the parse reads | **Drift.** Fix it here, in the same pass |
| C# lists one canonical path per platform; Go probes a list of candidates | Fine — documented at the top of `paths.go`. The site prints a path for a human; the app has to find one |
| C# returns a display `Note` for a picker; Go uses notes only in `--root` failure output | Fine |
| Go supports a Steam root or prefix user the C# list doesn't mention | Fine, and worth telling the site about — see below |
| A field one side reads and the other ignores | **Drift** until you can say why in a sentence |

## When the site is the one that is wrong

Two of these are known to be unverified on the C# side, and this app is what verifies them:

- **The Linux/Proton prefix shape** is confirmed from public sources, not from a Proton install
  anyone has run ([NoitaSpellCasters#67](https://github.com/TheShrug/NoitaSpellCasters/issues/67),
  and its comment on [#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66)). If a real
  machine disagrees with the list, **the list is wrong, not the machine.**
- **macOS has no path at all** on either side, by design.

So a mismatch found by running `companion paths` on a real box is a finding for *that* repo. File it
there, quote the actual path, and say which distribution and Steam install produced it.

## Run it

1. Fetch each C# file in the table above.
2. For each pair, list the facts the C# encodes — folder names, the AppID, attribute names, units —
   and confirm the Go copy encodes the same ones.
3. Report as a table: pair → `same` / `drifted (what)` / `not built yet`.
4. Anything drifted: fix it here if this side is wrong, or file it on NoitaSpellCasters if that side
   is. Never leave a known mismatch unrecorded — an undocumented mismatch is the state this skill
   exists to prevent.
