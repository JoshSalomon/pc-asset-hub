---
id: "mem_bf8081fc"
topic: "FF-15 session lessons: Phase 6 before Phase 7, TDD discipline, arithmetic verification"
tags:
  - feedback
  - process
  - feat-dev
  - coverage
  - tdd
  - lessons-learned
phase: 0
difficulty: 0.9
created_at: "2026-05-19T15:25:43.248321+00:00"
created_session: 17
---
## Key Lessons from FF-15 Export Plugins Session

### Phase 6 must run before Phase 7
Skipping quality review and going straight to coverage resulted in 200+ tests validating wrong VirtualServer behavior. The test completeness agent (QR4) would have caught that the VirtualServer tests were testing "all tools from all servers" instead of "filtered by selected VS instance." Coverage on wrong code creates false confidence.

### TDD discipline
User had to remind 5+ times to write RED test first. Common failure mode: "I see the fix, let me just edit the code." Fix: treat any edit to a non-test file without a preceding RED test as a stop signal.

### Arithmetic reconciliation
Presented "+12 net uncovered but only 8 justified" without noticing the 4-line gap. User caught it. The missing 4 were real testable error paths in instance_service.go that were then covered with tests. Always compute `(new_total - baseline_total) - (new_covered - baseline_covered) = net_new_uncovered` and verify it matches justified count.

### "Pre-existing" test failures
Dismissed 7 system test failures as "pre-existing" when they were caused by DNS-1123 validation change and publish preview modal change. Both were branch changes breaking tests. Any failure on the branch that passes on main is a regression.

### Deploy before claiming done
Multiple times reported code ready without deploying, leading to live test failures against stale code.

### Coverage agent dispatch
First Haiku coverage agent measured but wrote zero tests. The prompt must explicitly say "Write tests for every uncovered line. Do NOT stop to ask questions." Check LTM for prior dispatch failures before writing coverage agent prompts.

**Why:** These lessons affect every future feature session. Phase ordering, TDD discipline, and arithmetic verification are the highest-impact improvements.

**How to apply:** Read these before starting any feat-dev Phase 4+. The CLAUDE.md rules now encode the hard gates.
