# Noita's save layout, as observed

What Noita actually writes, checked against a real install rather than inferred. Everything here was
verified on **Windows, 2026-09-01**, against a save folder with 1,288 session stat files going back
to 2020. Where something is unverified it says so.

This exists because the app's whole job is finding and correlating these files, and a wrong
assumption here fails silently — a watcher pointed at the wrong folder looks exactly like a player
who never recorded anything.

## The tree

```
Nolla_Games_Noita/
├── save00/                     the active save (also save01 … save0N)
│   ├── player.xml              the player entity: wands, inventory, perks
│   ├── world_state.xml         run-scoped state, including the session id
│   ├── session_numbers.salakieli   encrypted; not readable
│   ├── stats/
│   │   └── sessions/
│   │       ├── YYYYMMDD-HHMMSS_stats.xml    one per run, named by run start
│   │       └── YYYYMMDD-HHMMSS_kills.xml
│   └── world/, persistent/
└── save_rec/
    └── screenshots_animated/   every gif, from every run, in one folder
```

**The gif folder is not inside a save folder.** It sits beside them and is shared by every save, so
a clip cannot be attributed to a run by where it lives. There is a test asserting this
(`TestGifsAreNotUnderTheSaveFolder`) because getting it wrong is silent.

## Is there a run identifier?

Yes — three of them, in ascending order of usefulness.

### 1. `player.xml` has none

It is the player entity serialization: `Entity` nodes tagged `wand`, their `AbilityComponent`,
`gun_config` and `gunaction_config`. Wands, inventory, perks. **Nothing run-scoped**, no seed, no
session id, no timestamp. Reading it tells you what the player is carrying *right now* and nothing
about which run that is.

So the obvious plan — parse `player.xml`, find a run id, key local state on it — does not work.

### 2. `world_state.xml` carries the session id

`WorldStateComponent` has:

```xml
session_stat_file="??STA/sessions/20260808-135431"
```

That timestamp is the run's **start** time, and it names the pair of files under
`stats/sessions/`. It is stable for the whole run and changes when a new run starts, which makes it
a genuine run identifier — and, being the start time, it sorts.

`stats/sessions/<id>_stats.xml` then carries the run's facts, including:

```xml
<stats … playtime="25.0333" world_seed="409296132" …>
```

### 3. The gif filename already carries the seed — this is the useful one

```
noita-20201123-153055-776668009-01028241.gif
      └─date─┘ └time┘ └──seed──┘ └frame?─┘
```

| Field | What it is | Confidence |
| --- | --- | --- |
| `20201123` | date the clip was written | certain |
| `153055` | time the clip was written | certain |
| `776668009` | **the run's `world_seed`** | **verified** — see below |
| `01028241` | a frame counter | likely; not verified |

**Verified** by taking the seed field from several gifs and grepping `stats/sessions/*_stats.xml`
for `world_seed="…"`. Each one matched exactly one session file, whose start timestamp preceded the
gif's:

| Seed in filename | Session stats file |
| --- | --- |
| `1973848030` | `20201101-121806_stats.xml` |
| `776668009` | `20201123-104435_stats.xml` |
| `1712275666` | `20201129-085539_stats.xml` |
| `940368919` | `20201028-093730_stats.xml` |

Clips recorded in the same run share the seed field — four clips across one afternoon all carrying
`776668009` — which is exactly the one-to-many run → clips grouping the queue needs.

## What this means for the app

**Grouping clips by run needs no XML parsing at all.** The seed is in the filename. A queue can be
built, grouped and displayed by reading nothing but a directory listing, which is a large
simplification over parsing a save file to get a key.

`world_state.xml` is still worth reading for the *current* run: it gives the session id, and through
it the run's stats, which is how you tell "the run happening now" from a clip recorded last week.

### The seed is not quite a unique run id

A player can replay a fixed seed, and Noita will happily write two runs with the same one. One real
sequence in this save shows the frame field going `00178501 → 00182240 → 00209835 → 00012445` under
a single seed on one day — consistent with a restart rather than one continuous run.

**So key local state on `(seed, session start)`, not on seed alone.** Seed alone is right almost
always, and quietly wrong for the player most likely to be uploading a lot of clips.

## Still unverified

- **Everything above is Windows.** The Linux/Proton prefix has never been checked against a real
  install, on either side of the port (see `/parity` and
  [NoitaSpellCasters#67](https://github.com/TheShrug/NoitaSpellCasters/issues/67)).
- **The fourth filename field.** Assumed to be a frame counter; nothing depends on it.
- **`session_numbers.salakieli` is encrypted** and was not opened. Nothing here needs it.
- **Whether an in-progress gif is distinguishable from a finished one** by name or by size — the
  watcher needs this and it is not answered here.
