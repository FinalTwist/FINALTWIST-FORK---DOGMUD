---
name: dogmud-end-of-day
description: Use at the end of a working day or session, when the owner signs off, says EOD, or asks to wrap up. Covers the daily archive sweep of docs/superpowers/specs and plans (shipped work to completed/, dropped work to abandoned/, every reference repointed), the repo-root tidy that archives old screenshots without touching binaries, and the session handoff memory.
---

This skill is the owner's end-of-day SOP (ruled 2026-09-25). It keeps
`docs/superpowers/specs` and `docs/superpowers/plans` holding only live work,
keeps the repo root free of stale screenshots, and leaves a handoff the next
session can start from. Run all three parts every EOD, even when a part turns
out to have nothing to do.

## 1. Archive sweep: specs and plans

The top level of each folder holds only LIVE work. Everything else moves:

| Destination | What goes there |
|---|---|
| `completed/` | Work whose PR merged. Superseded notes whose successor shipped go here too, beside it |
| `abandoned/` | Work that was dropped or will never be built, per its own banner or an owner ruling |
| stays at top | An arc spec whose arc is still open, a spec or plan in progress, or a spec that covers an open follow-up chunk |

**Prove shipped from git, never from memory.** The first first-parent commit
on master that touches a file is the merge that brought it in:

```bash
git log --first-parent master --reverse --format='%h %ad %s' --date=short -- <path> | head -1
```

A plan can land before its work does (a PR can carry the plans for later
PRs), so check that the PR that shipped the WORK merged too:
`gh pr list --repo pruuk/DOGMud --state merged --search "<n>"`. An arc spec
stays at the top until the arc's LAST piece ships; read its roadmap or its
own open-items section to decide.

**Move with `git mv`, then repoint every reference.** Moved paths are cited
from Go comments, test constants, content YAML comments, tools, skills,
`context.md` files and `docs/README.md`. List the stale ones:

```bash
git grep -nF -f moved.txt -- . ':!docs/superpowers/specs/completed/*.md' ':!docs/superpowers/plans/completed/*.md' \
  | grep -vE "completed/(<moved names, |-joined>)"
```

- `specs/<name>` and `plans/<name>` rewrite to `specs/completed/<name>`. The
  rewrite is idempotent.
- Bare-filename relative links (`](name.md)`) need a hand fix.
- **Links FROM moved files break too.** A moved spec that links a sibling
  which stayed at the top now needs `../`; a moved plan's `../specs/x` becomes
  `../../specs/completed/x`. Run a relative-link check over every tracked
  `.md` and fix only the breaks this move caused; leave pre-existing ones.

⚠️ Git Bash `sed -i` strips every CR from the files it touches, even for a
within-line substitution. That is harmless only where the index stores LF
(`git ls-files --eol` shows `i/lf`), because `core.autocrlf` restores CRLF on
checkout. Check it before trusting a `sed` pass, and use the Edit tool on any
file whose index is CRLF. Either way, `git diff --numstat` must show only the
lines you meant to change.

⚠️ `_datafiles/config.yaml` carries skip-worktree. If a moved path is cited
there, build the change from the `git show HEAD:` blob (`dogmud-balance-config`).

Afterwards run the guard tests whose constants name spec paths
(`go test -count=1 -run 'Identifier|WireFreeze|ConsistentAttack' .`) and
`gofmt -l` on any touched Go file.

## 2. Repo-root tidy

- **Screenshots and loose images** older than today move to
  `_archive/screenshots/` (gitignored). TODAY'S stay: the owner drops fresh
  screenshots in the root for Claude to read. Never overwrite an archive entry
  of the same name; skip it and say so.
- **Stray logs or scratch files** in the root move to `_archive/logs/` or
  `_archive/notes/`.
- **Never delete or move a binary** (`*.exe`). They are build outputs the
  owner may be running, and killing or replacing the owner's server is a
  tripwire. List them if they look stale; do not act.
- **Never touch `novel/`**, `.mcp.json` or `.claude/settings.local.json`.
- Anything untracked and unrecognised: report it, do not move it.

## 3. Handoff

Write or update the day's `project-session-handoff-YYYY-MM-DD` memory:
what merged, what is open and on which branch, what is waiting on someone
else, and every trap the day found. Point `MEMORY.md`'s current-status block
at it and archive the previous dated block to `STATUS-ARCHIVE.md`.

## Commit

The archive sweep is its own commit on the working branch (or a
`chore/eod-archive-<date>` branch off master when no branch is open), with
named paths only. The message states how many moved to each destination, why
anything stayed at the top, and that the link check found no new breaks.
