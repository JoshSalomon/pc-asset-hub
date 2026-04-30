---
id: "mem_c98f8c56"
topic: "Coverage "v8 artifact" excuse is almost always a missing test"
tags:
  - coverage
  - v8-artifact
  - laziness
  - skill-update
phase: 0
difficulty: 0.7
created_at: "2026-04-19T12:06:57.716712+00:00"
created_session: 17
---
In Session 023, the coverage-generate agent claimed `buildTypedAttrs.ts` L18 (`if (isEdit) result[k] = null`) was a "v8 instrumentation artifact — file unchanged" and the coverage-review agent accepted this justification. The human pushed back, a 3-line browser test was written (`buildTypedAttrs({ hostname: '' }, [makeAttr('hostname', 'string')], true)`), and the line was immediately covered.

**Lesson:** "v8 artifact", "instrumentation changed", and "file unchanged" are never valid excuses for uncovered lines. Always write the test first. If the test covers the line, it was never an artifact.

Updated both skills:
- `coverage-review/SKILL.md`: Added REJECT rules for v8/instrumentation/file-unchanged claims, plus Session 023 incident description
- `coverage-test/SKILL.md`: Added explicit prohibition on v8/artifact excuses, requiring test-first proof
