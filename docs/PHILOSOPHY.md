# Core philosophy

The principles below are stable. Features and implementation are expected to change; these should
not. When a proposed change conflicts with one of these, the principle wins — or the principle is
discussed and changed deliberately, not eroded by accident.

## 1. A stranger has to be able to audit this before running it

This is a background process on someone else's computer, reading their files and sending things over
the network, distributed as an **unsigned binary**. The only reason anyone should trust it is that
they, or someone they trust, can read it. That makes reviewability a functional requirement, not a
style preference.

Concretely: stdlib first, every dependency justified in writing, no build step that obscures what
ships, and no clever code where obvious code fits. If a choice makes the source harder to read in
exchange for something else, the something else needs to be worth it out loud.

## 2. Nothing leaves the machine without the player choosing it

The app watches, parses, and queues on its own. It **uploads only when a person clicks upload.**
There is no "silent draft", no opportunistic sync, no batch that goes out because the queue got
long. A player who never opens the queue has never uploaded anything.

This is also why the review queue is not optional UI. It is the consent step.

## 3. The site's gates are never bypassed

A clip arriving through this app gets the same transcode, the same moderation gate, and the same
limits as one dragged onto the upload page. The companion endpoint exists to make attribution
trustworthy and to avoid orphaned uploads — not to take a shortcut around anything the site does to
every other upload.

If this app can do something the web page cannot, that is a bug in one of them.

## 4. It reads the player's files; it does not touch them

Noita's save folder belongs to Noita. The app opens `player.xml` and the gif folder read-only and
writes nothing into them — no marker files, no renames, no cleanup of clips it has already handled.
Its own state lives in its own config directory.

A tool that corrupts a run is unforgivable in a game where runs are the whole point.

## 5. It just works, or it says exactly why not

The player is mid-run, or has just finished one. They are not going to read documentation or edit a
config file. Defaults have to be right on an ordinary install, and every failure has to name the
thing that failed and what to do about it — `companion paths` printing the four folders it searched
beats "Noita not found", every time.

The corollary: **a plausible wrong guess is worse than an honest gap.** Where the path is genuinely
unknowable — a macOS Wine bottle — the app asks rather than guesses.

## 6. Two copies of a fact, and one way to notice

This repo re-implements logic the site already has in C#. That is a deliberate, bounded trade
([ADR 1](adr/0001-go-with-a-second-small-parser.md)), and it survives only because the duplication
is *marked* — a `PARITY:` comment on both sides and a skill that checks them. A ported thing that is
not marked is the failure this principle exists to prevent.
