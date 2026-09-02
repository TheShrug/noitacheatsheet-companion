#!/usr/bin/env node
// SessionStart hook: surfaces the repo's unfinished GitHub Issues as additionalContext
// so Claude loads them into the Task list at the start of a session in this repo.
//
// The queue is GitHub Issues (ADR 26), so this needs the network and an authenticated
// `gh`. It must never be the reason a session is slow or broken: any failure — gh not
// installed, not logged in, offline, rate-limited, a repo with issues disabled — exits
// 0 with no output, and the call is capped at 5s.

const { execFileSync } = require('child_process');
const path = require('path');

const repoRoot = path.join(__dirname, '..', '..');

let issues;
try {
  const out = execFileSync(
    'gh',
    ['issue', 'list', '--state', 'open', '--limit', '100',
     '--json', 'number,title,labels,url'],
    { cwd: repoRoot, timeout: 5000, stdio: ['ignore', 'pipe', 'ignore'] }
  ).toString();
  issues = JSON.parse(out);
} catch {
  process.exit(0);
}

if (!Array.isArray(issues) || issues.length === 0) process.exit(0);

// type/status live in labels: "type: chore", "status: active". Queued is the absence
// of a status label; an untyped issue is still worth surfacing, so it reads as "untyped".
function label(issue, prefix, fallback) {
  const hit = (issue.labels || []).find((l) => l.name.startsWith(prefix + ':'));
  return hit ? hit.name.slice(prefix.length + 1).trim() : fallback;
}

const items = issues
  .map((i) => ({
    number: i.number,
    title: i.title,
    url: i.url,
    type: label(i, 'type', 'untyped'),
    status: label(i, 'status', 'queued'),
  }))
  .sort((a, b) => a.number - b.number);

const lines = items.map(
  (t) => `- #${t.number} (${t.type}, ${t.status}) — ${t.title}  [${t.url}]`
);

const context = [
  'This repo tracks work as GitHub Issues (ADR 26). The following issues are open ' +
    '(status queued/active/blocked). As your very first action this session, before ' +
    'replying to the user or doing anything else, call TaskCreate once per issue below ' +
    '(skip blocked ones or note the blocker in the description) — do not just list or ' +
    'describe them in text, actually call the tool. Use this EXACT string as each task ' +
    'subject: "#<number> (<type>, <status>) — <title>" (e.g. "#69 (chore, queued) — ' +
    'Cache Docker layers in the deploy workflow"); put the issue URL in the description:',
  ...lines,
].join('\n');

process.stdout.write(
  JSON.stringify({
    hookSpecificOutput: {
      hookEventName: 'SessionStart',
      additionalContext: context,
    },
  })
);
