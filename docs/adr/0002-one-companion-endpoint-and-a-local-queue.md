# 2. One companion upload endpoint, and a queue the player confirms locally

**Date:** 2026-09-01
**Status:** Proposed. The endpoint is not built; it is a change to
[`TheShrug/NoitaSpellCasters`](https://github.com/TheShrug/NoitaSpellCasters) and needs its own
ticket and ADR on that side.

## Context

Uploading a gif does not create a wand on the site today. It takes two calls:

1. `POST /api/wands/uploadwand` — transcodes, runs the moderation gate, stores to R2, returns a
   `WandShortID`. Writes **no database row**, and carries no `[Authorize]`.
2. `POST /api/wands` — writes the row, carrying the stats and the spell/perk lists, and taking
   ownership from a **client-supplied `CreatedByUserGuid`** in the body.

That shape exists because the *browser* needs it: the upload page parses `player.xml` client-side,
shows the player their wands, and then posts. Handing the same two calls to a background uploader
inherits two problems it does not have to:

- **A failure between the calls leaves an orphaned R2 object** — a transcoded clip with no wand. A
  human dragging files retries by hand and notices; a background process does it silently, on a
  schedule, forever.
- **Attribution is client-assertable.** `CreatedByUserGuid` is whatever the caller says it is, on an
  anonymous endpoint. Issue #66 already flags this as worth its own bug; a background uploader makes
  it a far more attractive target than it is when a human has to drag a file.

Meanwhile issue #66's other open fork was whether the site gains a **draft state** — upload silently
as private, review on the site later. `Wand` has no visibility or published column and listings key
off `Mp4Url != null`, so anything uploaded is immediately public. Adding a draft state is a schema
change plus a review surface.

## Decision

**One endpoint on the site, purpose-built for the companion**, taking the clip and the wand in a
single authenticated request and doing both halves server-side, inside one transaction as far as R2
allows. The app never calls `uploadwand` or `POST /api/wands`.

**Attribution comes from a bearer token, not the body.** Login already mints a JWT with roles; the
app holds one. The endpoint ignores any `CreatedByUserGuid` it is handed.

**No draft state.** The queue lives on the player's machine and nothing is sent until they confirm
it — which satisfies #66's "nothing is published without the player having seen it" without a schema
change or a second review surface. The player finishes a run, and reviews the queue whenever they
feel like it.

The site's existing gates are **not** bypassed: same transcode, same moderation
(NoitaSpellCasters ADR 6), same 100 MB request cap. A clip arriving this way must be
indistinguishable from one dragged onto the page, and that is a test on the site's side.

## Consequences

- The wand parse stays small here, because the app sends what it read and the server resolves spell
  and perk identity against its own enums — the constraint from
  [ADR 1](0001-go-with-a-second-small-parser.md).
- **This app is blocked on a change to another repo.** Until the endpoint exists there is nothing to
  upload to, and the honest options are to wait or to build against the two existing calls and throw
  that away. Waiting is the plan.
- The app needs a credential store: a token on disk, on a machine we do not control. Its scope,
  lifetime, and what happens when it expires mid-run are unresolved and belong in their own ticket.
- The anonymous, spoofable `uploadwand` path still exists for the browser. This decision routes
  around it rather than fixing it; it stays a live issue on the site.
- Automated uploads multiply the transcode footprint, which is already unbounded and queued as
  [NoitaSpellCasters#58](https://github.com/TheShrug/NoitaSpellCasters/issues/58). That should land
  first, or at least not last.
