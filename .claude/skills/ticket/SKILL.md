---
name: ticket
description: Manage the work queue, which lives in this repo's GitHub Issues — one issue per work item (feat/tckt/bug/chore/spike), typed and staged by label. Use when asked to add/queue a task, see the backlog or queue, pick what's next, start/close/block/drop a ticket, or break a feature into tickets.
---

# Ticket — the work queue on GitHub Issues

The queue is **`TheShrug/noitacheatsheet-companion`'s GitHub Issues**. One issue per work item; the
issue number is the id. Same convention as the rest of the fleet, adopted here on day one rather
than migrated into — `TheShrug/NoitaSpellCasters` reached it from Markdown files in `docs/queue/`,
and its ADR 26 is the reasoning if you ever need it.

The epic for this whole app is
[NoitaSpellCasters#66](https://github.com/TheShrug/NoitaSpellCasters/issues/66), filed on the site's
repo because the site is what it uploads to. Work *on this app* is filed here — see CLAUDE.md,
"Work queue", for which side a given item belongs on.

Everything runs through `gh`, from inside the checkout so the repo is inferred. This skill is the
front-end; the convention works by hand too.

> Scope: ADRs are **not** in this queue — they stay canonical in `docs/adr/`. An issue references an
> ADR by linking it; a "decide X" item produces a new ADR as its output.

## The convention

| Concept | On GitHub |
| --- | --- |
| id | The issue number |
| type | Label `type: feat` / `tckt` / `bug` / `chore` / `spike` — exactly one, always |
| status `queued` | Open, no `status:` label |
| status `active` | Open + `status: active` |
| status `blocked` | Open + `status: blocked`, with a comment saying what it's blocked on |
| status `done` | Closed, reason **completed** |
| status `dropped` | Closed, reason **not planned** |
| parent (epic) | A native **sub-issue** of the `feat` |
| body | `## Goal`, `## Acceptance criteria` (checkboxes), `## Notes` — see `.github/ISSUE_TEMPLATE/ticket.yml` |

Types: `feat` (epic/capability), `tckt` (atomic unit of work), `bug`, `chore`, `spike`
(time-boxed investigation whose output is knowledge).

Two things `gh` does **not** cover, so they go through the GraphQL API (see `new` and `list` below):
sub-issue links, and reading an issue's parent.

## new — create an item

1. Pick the type and write the body in the three-heading shape. Keep a `tckt` genuinely atomic —
   one sittable change.
2. File it:

   ```bash
   gh issue create --title "Short imperative title" --body-file <(...) --label "type: tckt"
   ```

3. If it belongs to a `feat`, link it as a sub-issue of that epic (this replaces the old `parent:`
   frontmatter — do not just mention the parent in prose):

   ```bash
   gh api graphql -f query='mutation($p:ID!,$c:ID!){addSubIssue(input:{issueId:$p,subIssueId:$c}){clientMutationId}}' \
     -F p="$(gh issue view <parent#> --json id -q .id)" \
     -F c="$(gh issue view <child#>  --json id -q .id)"
   ```

**Breaking a feature into tickets:** create the `feat` first to get its number, then each child
`tckt`, then link each one as a sub-issue. GitHub renders the children on the epic and shows a
progress bar as they close — no hand-maintained children list.

## list — show the queue

```bash
gh api graphql -f query='query{repository(owner:"TheShrug",name:"noitacheatsheet-companion"){
  issues(states:OPEN,first:100,orderBy:{field:CREATED_AT,direction:ASC}){
    nodes{number title labels(first:10){nodes{name}} parent{number}}}}}' \
  --jq '.data.repository.issues.nodes[] | "#\(.number) [\(.labels.nodes|map(.name)|join(", "))] parent=\(.parent.number // "-") \(.title)"'
```

Present grouped by status (active → blocked → queued), showing number, type, and title, with
children indented under their `feat`. Closed items are the history and are **not** shown by default
— for those, `gh issue list --state closed` (add `--search "reason:not-planned"` for the dropped
ones specifically).

`gh issue list --json` has no `parent` field, which is why `list` uses GraphQL. Plain
`gh issue list --label "status: active"` is fine when the hierarchy doesn't matter.

## next — what to work on

The lowest-numbered open issue that is `queued` (or `active` if one is already in progress) whose
parent, if any, is still open. Show it and offer to `start` it.

## start / done / block / drop — change status

```bash
gh issue edit   <n> --add-label "status: active"                  # start
gh issue reopen <n>                                               # start, if it was closed
gh issue edit   <n> --add-label "status: blocked" --remove-label "status: active"
gh issue comment <n> --body "Blocked on …"                        # block: say what on
gh issue close  <n> --reason completed                            # done
gh issue close  <n> --reason "not planned" --comment "Why …"      # drop
```

- `done` — tick the acceptance-criteria checkboxes that are met (`gh issue edit <n> --body-file`)
  **before** closing, then close as completed. Remove any `status:` label on the way out.
- `drop` — always leave a comment saying why; a dropped issue with no reason is a number nobody can
  interpret later.
- Reopening is just `gh issue reopen` — nothing moves, nothing is renumbered.

**Never delete an issue to close it.** Closed issues are the project history, which is the whole
point of the numbering.

**Before marking `done` (or `dropped`), check for an undocumented decision.** If the work landed on
a particular approach among real alternatives, reversed an earlier plan, or changed something a
future contributor would need the *why* for (not just the *what* the diff already shows) — write the
ADR now, in the same pass, before closing the issue. Don't wait to be asked; an issue closed without
one is the decision going undocumented. Link it from the issue. Routine/mechanical items with no
real fork in the road don't need one — see
[`docs/adr/README.md`](../../../docs/adr/README.md) for the format.

**After marking `done`/`dropped`, run the `/ticket-close` skill in the same pass** — the ripple
sweep over everything the finished work touched: downstream-issue notes, doc/config-template/
devcontainer parity, test-artifact cleanup, and follow-up issues worth queueing. It runs
automatically on close, not only when asked.

## Conventions

- **One decision per ADR, one atomic change per `tckt`.** If a `tckt` grows two unrelated goals, split it.
- **Commit and push on `start` and on `done`/`dropped`.** Ticket boundaries are this repo's commit
  points — see CLAUDE.md, "Committing". Reference the issue in the commit message (`#69`) so the work
  and the ticket are linked from both directions; `Closes #69` on the PR closes it automatically.
- **Context belongs in the issue, updated as you go** — a comment when something surprising turns up,
  an edit to `## Notes` when the scope sharpens. The issue is the only durable record now; there is
  no file in the repo to edit instead.
- `gh` needs auth. Inside the devcontainer that means `gh auth login` once (or a `GH_TOKEN` in the
  environment) — without it every verb here fails.
