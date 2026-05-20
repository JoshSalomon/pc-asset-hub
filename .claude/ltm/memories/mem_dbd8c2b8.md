---
id: "mem_dbd8c2b8"
topic: "Coverage generator agents must write tests, not just measure"
tags:
  - feedback
  - coverage
  - subagent-dispatch
phase: 0
difficulty: 0.8
created_at: "2026-05-13T12:54:15.351597+00:00"
created_session: 19
---
The coverage-generate subagent (Haiku) failed on its first dispatch — it measured coverage gaps (157 backend + 64 UI uncovered lines) but wrote ZERO tests, then stopped to ask questions. This is unacceptable.

**Why:** The user pays for results, not reports about what needs to be done. The skill (coverage-test-subskill) has explicit steps 3b, 4, and 5 requiring test writing. The agent must execute ALL steps.

**How to apply:** When dispatching coverage-generate agents, the prompt MUST explicitly state: "Write tests for every uncovered line. Do NOT stop to ask questions. Do NOT skip lines. Execute steps 3b, 4, and 5 of the skill." Include the specific uncovered files and line counts so the agent knows the scope upfront. If the agent returns without having written tests, it failed and must be re-dispatched immediately with stronger instructions.
