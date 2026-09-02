# Architecture Decision Records

Significant design and architecture decisions are recorded here as ADRs — one decision per file,
numbered and dated. They capture *why* a choice was made so it survives the conversation it was made
in. When a decision is later reversed, don't delete its ADR — add a new one that supersedes it and
update the older one's status.

ADRs record decisions; the durable principles they must not violate live in
[`../PHILOSOPHY.md`](../PHILOSOPHY.md). When an ADR conflicts with a principle, the principle wins.

Format (lightweight MADR): `# N. Title`, then **Date**, **Status** (Proposed / Accepted /
Superseded), **Context**, **Decision**, **Consequences**.

Same format and numbering as [`TheShrug/NoitaSpellCasters`](https://github.com/TheShrug/NoitaSpellCasters/blob/main/docs/adr/README.md),
deliberately — this repo's numbering starts fresh at 1 and is unrelated to that one's. When an ADR
here depends on one there, name the repo: "NoitaSpellCasters ADR 6", never a bare "ADR 6".

> **Numbers are allocated per branch, so two open branches will claim the same one.** Check
> `origin/main` before adding an ADR, not just your checkout.

## Index

| # | Title | Status |
| - | ----- | ------ |
| [1](0001-go-with-a-second-small-parser.md) | Go, with a second small parser rather than sharing the site's C# | Proposed |
| [2](0002-one-companion-endpoint-and-a-local-queue.md) | One companion upload endpoint, and a queue the player confirms locally | Proposed |
| [3](0003-tray-plus-a-localhost-review-queue.md) | A tray icon whose menu opens a review queue served on localhost | Proposed |
