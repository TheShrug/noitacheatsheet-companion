---
name: ticket-close
description: Close-out ripple pass for the GitHub Issues work queue — run automatically whenever an issue is being marked done or dropped (alongside /ticket close), not only when asked. Sweeps the fallout of the finished work: downstream-issue notes, doc parity, ADR gaps, dependency review, parity with the site's copy of ported logic, and follow-up issues worth queueing, then proposes the changes.
---

# Ticket close-out — the ripple pass

The `/ticket` skill handles the *mechanics* of closing (status flip, acceptance boxes, ADR check).
This skill is the second half: **what did finishing this work change about everything else?** Run it
in the same pass as closing — a ticket closed without the sweep leaves the queue and docs describing
a repo that no longer exists.

Work through the checklist, then act as follows: apply the uncontroversial items directly (stale
doc line, downstream-ticket note), and **propose** anything scope-shaped (new tickets, standards,
new dependencies) instead of doing it unilaterally. Present the whole sweep compactly at the end —
including "checked, nothing needed" lines, so the user can see what was considered. Commit and push
when the sweep is done — closing a ticket is a commit point (see CLAUDE.md, "Committing").

> The queue is this repo's GitHub Issues, so nothing moves on close — the issue you're sweeping is
> the one you just closed, and the **downstream** items you scan below are the still-open issues.
> Edits to an issue body or a comment land immediately and aren't part of the commit; only the
> repo-side edits from this sweep are.

## Checklist

1. **Downstream issues** (open, queued/blocked): did this work land a seam, convention, or gotcha
   that sharpens a later ticket's scope? Update that issue's `## Notes` (or add a comment) with the
   concrete pointer so the next session doesn't rediscover it. Also check whether this work
   *changed a later ticket's assumptions* — a removed fallback another ticket relied on is the most
   dangerous kind of staleness.

2. **New work surfaced**: bugs found but not fixed, tech debt deliberately taken on, stopgaps that
   need a real fix. Each needs a home — an existing issue's notes or a proposed new one. A caveat
   that lives only in the closed issue is effectively lost.

3. **Docs**: `CLAUDE.md` (build/run/gotchas still accurate?), `README.md` (does it still describe
   what the app actually sends?), `SECURITY.md`, and the ADR index table in `docs/adr/README.md`
   (every ADR on disk has a row). The ADR itself should already exist via the `/ticket` close flow
   — this step is about *references* to reality, not the decision record.

4. **Parity with the site** (`/parity`): if the work touched anything ported from
   `TheShrug/NoitaSpellCasters` — the save paths, the wand parse, the upload contract — run
   `/parity` and say what it found. This is the standing cost of the second copy existing, and the
   only thing that keeps it from being paid by a player getting a wrong wand.

5. **Dependency review**: did this work add a module? If so, `go.mod` and `go.sum` are both
   committed, the addition is named in the ADR or issue that justified it, and `README.md`'s
   dependency claim is still literally true. **A dependency added quietly is a broken promise** —
   this repo's pitch is that a stranger can read it before installing it.

6. **Dev-environment parity**: anything installed or changed live in the running container during
   the work must also be in `Dockerfile` or `.devcontainer/devcontainer.json` — and conversely,
   one-off tools used only for verification should *not* be baked in. Say which way each live
   change went.

7. **Test artifacts**: scratch files, real save folders, downloaded clips, or background processes
   created while verifying are cleaned up. Real player data must never end up committed — see
   `.gitignore`'s `scratch/` entry.

8. **Standards worth codifying**: did the work settle a repeatable pattern (error-message shape,
   a verification recipe, a naming scheme)? Propose where it should live — an ADR if it was a real
   decision, a ticket note if it's scoped to an epic, or a project skill if it's a workflow.

## Output shape

End with a short table or list: each checklist item → what was updated / proposed / explicitly not
needed. Then the one question that matters, if any — don't bury decisions in prose.
