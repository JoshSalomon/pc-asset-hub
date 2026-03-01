---
id: "mem_43cb62f3"
topic: "Feature development methodology - feat-plan skill (8 phases)"
tags:
  - methodology
  - planning
  - process
  - feat-plan
  - approval-gates
phase: 0
difficulty: 0.6
created_at: "2026-03-01T17:35:13.007013+00:00"
created_session: 12
---
# Feature Development Methodology

The `feat-plan` skill (`~/.claude/skills/feat-plan/SKILL.md`) defines the process. Key points:

## 8 Phases with approval gates
1. **Requirements & PRD** → APPROVAL
2. **HL Test Plan** → APPROVAL
3. **Detailed Test Plan** → APPROVAL + choose execution mode
4. **Implementation** (TDD red/green, bottom-up)
5. **Deploy & Live Test**
6. **Quality Review** (3 parallel code-reviewer agents)
7. **Coverage & Verification**
8. **Documentation & Manual Approval** → APPROVAL

## Execution modes (chosen after Phase 3)
- **Supervised**: approval after every phase
- **Independent**: runs phases 4-8 without stopping, EXCEPT quality review issues always require approval

## Critical rules
- NEVER skip approval gates for phases 1-3
- NEVER batch phases 1-3 together
- Quality review fixes MUST use TDD (write failing test first)
- Small UX fixes don't need feat-plan — use TDD directly
- New endpoints go in `docs/architecture.md`, not `PRD.md`

