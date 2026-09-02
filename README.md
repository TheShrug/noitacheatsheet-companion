# noitacheatsheet-companion

A small desktop app that watches Noita's own gif folder and your active save, and gets a wand onto
[noitacheatsheet.com](https://noitacheatsheet.com) without you alt-tabbing mid-run to drag two files
into a browser.

> **Status: early. It cannot upload anything yet.**
> One command works today — `companion paths`, which finds Noita's folders on your machine and tells
> you what it found. The watcher, the review queue and the upload are open work; see
> [Issues](https://github.com/TheShrug/noitacheatsheet-companion/issues) and
> [`docs/adr/`](docs/adr/README.md) for what is decided and what isn't.

## What it does, and what it will never do

Noita records its own animated gifs, and the upload page already asks for exactly two files: a clip
from `save_rec/screenshots_animated`, and your save's `player.xml`. This app uses the same two
inputs. It is not a Noita mod, does not inject anything into the game, and does not need unsafe mods
enabled.

| It does | It never does |
| --- | --- |
| Read your `player.xml` and the gifs Noita recorded | Write anything into Noita's folders |
| Show you a queue of clips and the wands on that save | Upload without you clicking upload |
| Send a clip and one wand, when you say so | Send your save file, your seed, or anything else |
| Run in the background with a tray icon | Run at boot unless you set that up yourself |

**Nothing is uploaded until you confirm it.** There is no silent draft and no background sync. If
you never open the queue, nothing has left your machine. That is a design commitment, not a current
limitation — see [`docs/PHILOSOPHY.md`](docs/PHILOSOPHY.md).

## Install

Grab the binary for your platform from
[Releases](https://github.com/TheShrug/noitacheatsheet-companion/releases), and check it against the
`SHA256SUMS` published beside it.

### Windows will warn you, and the warning is accurate

The binaries are **unsigned**. Windows SmartScreen shows *"Windows protected your PC"* and hides the
run button behind **More info → Run anyway**.

That warning does not mean the file is known-bad. It means nobody has paid for a code-signing
certificate — which is true, and is not going to change soon. It also means SmartScreen has not
checked this file for you, so the check is yours to do:

1. Compare the SHA-256 of your download against `SHA256SUMS` on the release
   (`Get-FileHash companion-windows-amd64.exe` in PowerShell).
2. If you would rather not trust a binary at all, [build it yourself](#build-it-yourself). It is one
   command and no dependencies.

Antivirus software sometimes flags new, unsigned Go binaries generically. Same answer: verify the
hash, or build from source.

## Where it looks for Noita

```
companion paths
```

Prints the data folder, the active save, and the clip folder — or, if it found nothing, every path
it searched. Point it somewhere else with `--root`, giving the folder that contains `save00` and
`save_rec`:

| Platform | Where that normally is |
| --- | --- |
| Windows | `%UserProfile%\AppData\LocalLow\Nolla_Games_Noita` |
| Linux | inside the Proton prefix for AppID `881100` — four Steam roots are tried |
| macOS | no default. Noita has no macOS build, so it lives in whichever Wine bottle you made — pass `--root` |

The "active save" is whichever `saveNN` has the most recently written `player.xml`. Noita does not
record which save is in use anywhere readable, so that is a proxy, and a good one unless you are
alternating between two runs in the same minute.

**If it finds nothing on Linux, that is worth reporting.** The Proton path list came from public
documentation rather than from a real install, so a mismatch is probably the list being wrong.
Paste the output of `companion paths` into an issue with your distribution and how you installed
Steam.

## Build it yourself

```sh
go build ./cmd/companion
```

No dependencies to fetch, no code generation, no build tags. `go.mod` requires Go 1.24 and lists no
modules — deliberately, and if that ever stops being true it will be because an ADR said why.

With `make` and Docker, the fuller interface:

```sh
make            # list the targets
make test       # gofmt, vet, and the suite
make run        # build, then report where Noita's files are
make dist       # release binaries for every platform, built in Docker
```

`make dist` builds through this repo's own `Dockerfile`, which is the same stage CI releases from —
so a binary you build matches the published one byte for byte, given the same commit and version.

## Related

- [`docs/noita-save-layout.md`](docs/noita-save-layout.md) — what Noita actually writes, and how a
  clip is tied back to the run that produced it. Checked against a real save, not inferred.
- [noitacheatsheet.com](https://noitacheatsheet.com) — the site this uploads to.
- [NoitaSpellCasters#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66) — the epic this
  app came out of.
- [`SECURITY.md`](SECURITY.md) — what this thing can reach, and how to report a problem.

## Licence

MIT. See [`LICENSE`](LICENSE).
