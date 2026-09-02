#!/usr/bin/env node
// PreToolUse hook (Read|Write|Edit): refuse to use Claude Code's auto-memory store.
//
// Rationale: anything durable enough to be worth remembering across sessions is durable enough
// to belong in the repo, where it's reviewable, diffable, and visible to every contributor —
// not in a per-user directory under ~/.claude that only one machine can see. So reaching for
// memory is treated as a signal that something is *undocumented*, and the fix is to write the
// documentation rather than the memory.
//
// This is a backstop. `autoMemoryEnabled: false` in settings.json already turns the feature off;
// this catches any direct Read/Write/Edit against the memory tree and redirects to the right file.

let raw = "";
process.stdin.on("data", (chunk) => (raw += chunk));
process.stdin.on("end", () => {
  let input;
  try {
    input = JSON.parse(raw);
  } catch {
    // Malformed payload: stay out of the way rather than blocking real work.
    process.exit(0);
  }

  const filePath = input?.tool_input?.file_path ?? "";
  const normalized = filePath.replace(/\\/g, "/");

  // The auto-memory store lives at ~/.claude/projects/<sanitized-cwd>/memory/. Match on that
  // shape rather than one hard-coded absolute path so it still fires if the project directory
  // is renamed or autoMemoryDirectory is repointed at another .claude tree.
  const isMemoryPath =
    /\/\.claude\/projects\/[^/]+\/memory(\/|$)/.test(normalized) ||
    /\/\.claude\/.*\/memory\/MEMORY\.md$/.test(normalized);

  if (!isMemoryPath) process.exit(0);

  const reason = [
    "This project does not use Claude Code's auto-memory store. Wanting to write one is a signal —",
    "work out which of these two it is before doing anything else:",
    "",
    "1. IT'S UNDOCUMENTED. Write the doc instead, in whichever fits narrowest:",
    "     - CLAUDE.md              how to work in this repo; conventions, build/deploy facts",
    "     - docs/adr/              a decision with real alternatives (see docs/adr/README.md)",
    "     - a GitHub issue       work to do, or context specific to one ticket (`/ticket`)",
    "     - docs/PHILOSOPHY.md     a durable design principle",
    "     - .claude/skills/*/      a repeatable workflow to re-run by name",
    "     - a README near the code it describes",
    "",
    "2. IT'S A SYMPTOM. A fact you have to *remember* to work in this codebase is often an",
    "   architectural problem or an inconsistency — a footgun you route around, two subsystems",
    "   disagreeing, a convention honoured in some places and not others. Recording it just makes",
    "   the defect survivable instead of fixed. Raise it with the user to discuss and triage:",
    "   a type: bug / type: chore issue if it's real work, an ADR if the fix is a decision.",
    "",
    "Prefer (2) when it plausibly applies — \"remember that X is weird\" is nearly always a bug",
    "report wearing a disguise. See CLAUDE.md, \"Documentation, not memory\".",
  ].join("\n");

  process.stdout.write(
    JSON.stringify({
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: reason,
      },
    })
  );
  process.exit(0);
});
