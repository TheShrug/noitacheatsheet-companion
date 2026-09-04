# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this
repository.

## Working with me

I'm an experienced engineer — talk to me like a coworker messaging back and forth on Slack, not a
report. Be concise and direct: lead with the answer, skip the preamble, and don't restate what I
just said or explain things I didn't ask about. A couple of sentences usually beats a wall of
bullets.

For small, one-off, or reversible tasks: check the environment first (just look — run the command,
read the file), take the obvious minimal path, and do it. Surface feasibility and a recommendation,
not a menu of options. Reserve design-space enumeration and clarifying questions for load-bearing or
hard-to-reverse decisions. If I repeat a steer like "keep it simple," treat the repetition as a sign
you've drifted and course-correct.

Once a bug's cause is established, stop investigating and fix it. Demonstrate the mechanism once,
then go to the edit — don't re-prove the same root cause with successive probes or gate the fix
behind more confirmation queries.

## What this repo is, and the two rules that follow

A background process that runs on **other people's computers**, reads their save files, and is
distributed as an **unsigned binary**. Nobody has vouched for it; the only reason to trust it is
that it can be read. Two rules fall out of that, and they outrank convenience every time. The longer
form is [`docs/PHILOSOPHY.md`](docs/PHILOSOPHY.md).

### Readability is a functional requirement

Write for a suspicious stranger skimming the diff on GitHub before deciding whether to run this.

- **Stdlib first.** Every dependency is code someone is trusting on top of ours. Adding one needs a
  reason written in the ADR or issue that introduces it, and an update to the README and
  `SECURITY.md`, both of which currently claim there are none. Don't add one to save yourself
  twenty lines.
- **Obvious over clever.** No reflection where a type switch works, no code generation, no build
  tags that change behaviour, no init-order tricks. If the straightforward version is slower, it is
  still the right version at this scale — one player, a handful of files.
- **Comment the why, never the what.** `// increment i` is noise; `// The gif folder is shared
  across saves, so this cannot be scoped to the active one` is the reason someone doesn't break it
  next year.
- **Errors name the thing and the fix.** A player mid-run will not read documentation. `Could not
  find Noita's data folder` followed by the four paths searched and the `--root` flag is the
  standard — see `noita.ExplainNotFound`.

### Security is the feature, not a review step

- **Never widen what the app can reach without saying so.** New file reads outside Noita's folder,
  new network destinations, new listeners, new persisted secrets — each one changes the table in
  `SECURITY.md`, and that table is a promise. Update it in the same commit.
- **Loopback only.** The review queue binds `127.0.0.1`, never `0.0.0.0`, and needs a `Host` check
  and a CSRF token before it ships ([ADR 3](docs/adr/0003-tray-plus-a-localhost-review-queue.md)).
- **Noita's folders are read-only to us.** No marker files, no renames, no tidying up clips. A tool
  that corrupts a run is unforgivable.
- **Nothing uploads without an explicit click.** No opportunistic sync, no "it had been queued a
  while". This is the consent step, not a UX detail.
- **Never log or commit real player data.** Save files and clips go in `scratch/`, which is
  gitignored. A real `player.xml` in a test fixture must be deliberately reduced first.

## Build & run

### Devcontainer (recommended)

`.devcontainer/devcontainer.json` builds the `dev` stage of the repo's own `Dockerfile`, so the
toolchain that runs the tests is the one that builds releases. Open the repo and run **Dev
Containers: Reopen in Container**. Then `make test`.

The Go version is pinned in **`go.mod` and nowhere else** — the Dockerfile's base image and CI's
`setup-go` both read from there. Bump it in one place.

### Host machine

Go alone is enough for everything except `make dist`:

```sh
go build ./cmd/companion    # build
go test ./... -race         # test
```

There are no modules to fetch and no code generation, so a clean checkout builds immediately.

> **`make` is not on this Windows host** — not in Git Bash, not in PowerShell, only in WSL. That is
> a property of the machine, not the repo.

### When `make` is not on PATH

A missing `make` means you are outside the dev environment. **Never hand-translate a target** — the
recipe is the spec, and a target that wraps three commands will lose one in translation. Either
reopen in the devcontainer, or run the underlying `go` commands above, which for this repo are
genuinely equivalent because nothing wraps Docker except `dist`.

## Local dev interface

The fleet's verbs, so moving between repos costs nothing:

| | |
| --- | --- |
| `make` | List the targets |
| `make build` | Build `./dist/companion` for this machine |
| `make test` | gofmt, `go vet`, and the suite with `-race` |
| `make run` | Build, then report where Noita's files are on this machine |
| `make dist` | Release binaries for every platform, built in Docker |

**There is no `make database`.** This app has no database — it runs on a player's machine and its
only state is a local queue file. A repo where a verb is meaningless and one where it was forgotten
look identical from outside, so this sentence exists.

`make run` reports paths today. When the review queue lands it becomes the background process and
prints `http://127.0.0.1:7331` as its last line, matching the fleet contract. That port is a default
on someone else's machine, not an entry in the homelab port table.

## Tests

`go test ./... -race -count=1`, run by `make test` and by CI on **both Linux and Windows** — the path
code is the part most likely to break and the part that differs per OS, so a Linux-only run never
executes the Windows branch of `Candidates()` at all.

`-count=1` defeats the test cache on purpose: a cached PASS tells you nothing about this commit.

Tests build their fixtures in `t.TempDir()` and touch no real machine, so the suite needs no network,
no game install and no save file. **That is also its limit** — it proves the logic, not that the
paths are right on a real box. Only running `companion paths` on a real machine does that, and for
Linux/Proton nobody has yet (see Known gotchas).

## Releases

**Nothing deploys from this repo.** There is no server, no Coolify application, no webhook. The
artifact is a binary a player downloads, which makes releases the one irreversible thing here:
a bad deploy elsewhere in the fleet is rolled back in a minute, and a bad binary is on people's
machines until they update.

- `.github/workflows/ci.yml` runs on every push and PR, **including `main`** — unlike the app repos,
  which exclude `main` because their deploy workflow gates it. Excluding it here would leave `main`
  with no gate at all.
- `.github/workflows/release.yml` runs on a `v*` tag, builds through the `Dockerfile`, publishes
  binaries plus `SHA256SUMS`, and states in the release notes that they are unsigned.
- Tag only from `main`, only after CI is green **on that commit**, and check the run measured what
  you think — a green run over the wrong commit is the fleet's recurring failure.

## Architecture

```
cmd/companion/      the CLI entry point
internal/noita/     finding Noita's folders, and reading what's in them
```

`internal/` deliberately: nothing here is a public API, and keeping it uni-importable means we can
change shape freely without someone depending on it.

The app never resolves spell or perk identity. It sends what it read and the server resolves it
against its own enums, which is what keeps the parse small and keeps the site's enums authoritative
([ADR 1](docs/adr/0001-go-with-a-second-small-parser.md), and the constraint on
[ADR 2](docs/adr/0002-one-companion-endpoint-and-a-local-queue.md)).

## Parity with the site

This repo re-implements logic that already exists in C# in
[`TheShrug/NoitaSpellCasters`](https://github.com/TheShrug/NoitaSpellCasters) — the save paths, and
(when it lands) the wand parse. That is a deliberate, bounded trade, and it only survives because the
duplication is marked:

- Every ported file opens with a `PARITY:` comment naming its C# counterpart.
- The **`/parity` skill** lists the pairs and how to read the other side (that repo is private —
  `gh api`, not a raw URL).
- `/ticket-close` runs it whenever ported code was touched.

**Drift fails silently.** A stale path makes the watcher see no clips, which looks exactly like the
player not having recorded any. Treat a mismatch found on a real Linux machine as the *site's* list
being wrong, not the machine.

## Work queue

Work lives in **this repo's GitHub Issues**, one issue per item, driven by the `/ticket` skill. The
convention is the fleet's:

- exactly one **`type:`** label — `feat` (epic), `tckt` (atomic unit of work), `bug`, `chore`,
  `spike` (time-boxed investigation whose output is knowledge);
- **status** is the issue's own state — open with no `status:` label is queued, `status: active` /
  `status: blocked` say the rest, `done` is closed as *completed*, `dropped` is closed as *not
  planned*;
- an epic's children are native **sub-issues**;
- the body is `## Goal` / `## Acceptance criteria` / `## Notes`, scaffolded by
  `.github/ISSUE_TEMPLATE/ticket.yml`.

Decisions are **not** tickets: they go in `docs/adr/`, and a "decide X" issue produces an ADR.

**Which repo does an item belong on?** Work on this app is filed here. Work on the site — the
companion upload endpoint, anything about moderation, transcoding or the wand schema — is filed on
`NoitaSpellCasters`, even when this app is the only thing that needs it. The epic for the whole
feature is [NoitaSpellCasters#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66).

The `/ticket` skill needs an authenticated `gh`; without one every verb it offers fails.

## Branches and pull requests

- **`main` is the integration branch** and the default base for everything. Nothing is committed to
  it directly. Unlike the app repos, `main` is not production — nothing deploys from here — but it
  is what gets tagged, so a broken `main` is a release waiting to happen.
- **A working branch per ticket**, cut from `main` and merged back through a pull request.
  `.github/workflows/ci.yml` runs on every branch and PR.

Name that branch:

```
<issue>-<type>-<slug>
```

```
^[0-9]+-(tckt|feat|bug|chore|spike)-[a-z0-9]+(-[a-z0-9]+)*$
```

- `<issue>` is the **issue number in this repo** — not a PR number. A PR number doesn't exist yet
  when the branch is cut, and renaming a branch after opening the PR detaches it from its head.
- `<type>` matches the issue's one `type:` label.
- `<slug>` is lowercase `a-z0-9-`; `.` and `_` collapse to `-`; aim for ≤ 40 characters. The issue
  holds the full title, so this is a handle, not a summary.

So issue #4 `type: tckt` "Watch the gif folder for completed writes" becomes
`4-tckt-watch-gif-folder`.

**No owner prefix.** The name used to start `TheShrug/`. It was dropped 2026-09-03: in a
single-maintainer fleet every branch carried it, so it distinguished nothing, and Orca's own
`branchPrefix` setting prepends the git username silently — two layers adding a prefix at once,
which is how `TheShrug/TheShrug-79-...` got created. Orca is set to `None` now, so `--name` is
the whole branch name. Existing `TheShrug/...` branches are grandfathered by the same date rule
below.

**No issue, no branch** — the number is mandatory, so every branch traces back to the queue. Still
reference the issue number in the PR title.

The fleet-wide policy and its reasoning live in the `homelab` vault at `Conventions/Branching.md`.
It's restated here rather than linked because that vault is private.

**Cut from `origin/main`, and fetch first.** A branch cut from a stale local `main` starts life
missing merged work and will conflict with it later. The sharper trap: **ADR numbers are allocated
per branch, so two concurrent branches will happily claim the same one.** Check `origin/main` before
adding an ADR, and if two branches are open at once, deliberately reserve different numbers.

## Committing

Commit **and push** at both ends of a ticket: once when starting it (issue labelled `status: active`,
scaffolding in place) and once when it's complete. Don't wait to be asked per ticket — ticket
boundaries are the commit granularity here, and pushing at the start keeps in-progress work off a
single machine. Reference the issue number in the commit message. Outside of ticket work, ask before
committing.

## Documentation, not memory

This repo sets `autoMemoryEnabled: false` and has a hook (`.claude/hooks/no-auto-memory.js`) that
refuses reads and writes against Claude Code's auto-memory store. Anything durable enough to remember
across sessions is durable enough to belong in the repo, where it's reviewable, diffable, and visible
to everyone — not in a per-user directory that only one machine can see.

So reaching for memory is a signal. Work out which of two things it is:

1. **It's undocumented.** Write it down, in the narrowest file someone would actually read at the
   moment they need it: this file for conventions and build facts, `docs/adr/` for a decision with
   real alternatives, `docs/PHILOSOPHY.md` for a durable principle, `SECURITY.md` for anything about
   what the app can reach, a GitHub issue for ticket-scoped context, `.claude/skills/` for a
   repeatable workflow.

2. **It's a symptom.** A fact that has to be *remembered* to work in this codebase is often an
   architectural problem — a footgun you route around, two things disagreeing, a name that means the
   wrong thing. Writing it down as a memory makes the defect survivable instead of fixed. **Raise it
   so we can triage it** — a `type: bug` or `type: chore` issue if it's real work, an ADR if the fix
   is a decision.

Prefer (2) when it plausibly applies.

## Recording decisions

Significant architecture/tooling decisions — picking one real alternative over another, reversing an
earlier plan, **adding a dependency**, anything a future contributor would need the *why* for — get
an ADR in `docs/adr/` (see [`docs/adr/README.md`](docs/adr/README.md) for the format). Write it
proactively when the decision is made. Routine changes with no real fork in the road don't need one.

ADRs 1–3 are **Proposed**, not Accepted: the repo is scaffolded on them and they are cheap to
reverse while the parser and watcher are unwritten. Don't quietly treat them as settled.

## Known gotchas

- **Windows is verified; Linux/Proton is not.** `companion paths` was run on a real Windows install
  on 2026-09-01 and resolved the data root, active save and clip folder correctly. The Proton path
  list was assembled from public documentation on both sides — here and in the site's
  `NoitaSavePaths.cs` ([NoitaSpellCasters#67](https://github.com/TheShrug/NoitaSpellCasters/issues/67))
  — and nobody has run it. The first person to try settles it; until then treat a Linux miss as a
  likely bug in the list, not in the machine.
- **"Active save" is a heuristic.** Noita records nowhere readable which `saveNN` is in use, so we
  take the newest `player.xml`. Right in every ordinary case; wrong for someone alternating between
  two runs within the same minute.
- **The gif folder is not inside the save folder.** `save_rec/screenshots_animated` sits beside
  them and is shared across every save, so clips cannot be scoped to a run by their location. There
  is a test asserting this, because getting it wrong makes the watcher silently see nothing.
- **`player.xml` has no run identifier — the gif filename does.** Checked against a real save on
  2026-09-01: `noita-<date>-<time>-<seed>-<frame>.gif`, where the seed field is the run's
  `world_seed`, verified against `stats/sessions/*_stats.xml`. So clips group by run from a
  directory listing alone, with no XML parsing. Key state on `(seed, session start)` rather than
  seed alone — a replayed fixed seed produces two runs with the same one. Full findings and method
  in [`docs/noita-save-layout.md`](docs/noita-save-layout.md).
- **`CGO_ENABLED=0` today, and the tray changes that.** Cross-compiling is currently free. The tray
  dependency makes Windows builds need mingw-w64 (already in the Dockerfile's `base` stage for this
  reason) and Linux builds non-static.
