---
id: "mem_339ff9f4"
topic: "LTM memory files are tracked in git and must be staged with commits"
tags:
  - ltm
  - git
  - workflow
phase: 0
difficulty: 0.2
created_at: "2026-04-05T10:07:59.575842+00:00"
created_session: 17
---
## LTM files are part of the project

LTM memory files live in `.claude/ltm/` and are checked into git. When creating or updating memories, always stage:
- `.claude/ltm/index.json`
- `.claude/ltm/memories/*.md` (new or modified memory files)

These should be included in commits alongside the code changes they relate to. Do NOT treat them as local-only files.
