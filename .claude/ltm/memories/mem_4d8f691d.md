---
id: "mem_4d8f691d"
topic: "BumpInstanceVersion must clean stale IAVs before creating new ones"
tags:
  - version-bump
  - iav
  - idempotent
  - orphan-cleanup
  - debugging
phase: 0
difficulty: 0.8
created_at: "2026-05-07T15:12:29.906977+00:00"
created_session: 17
---
`BumpInstanceVersion` creates IAV rows at version N+1 via `SetValues` BEFORE calling `instRepo.Update`. If `Update` fails (e.g., unique constraint on name+parent scope), the IAV rows at N+1 are already written but the instance stays at version N. On retry, `BumpInstanceVersion` tries to create IAVs at N+1 again → 409 duplicate.

Discovered when SetParent failed due to a name collision at the new parent scope. User saw 500 on first attempt, then 409 "duplicate attribute value" on retry with a different parent.

**Fix:** `BumpInstanceVersion` now calls `iavRepo.DeleteByInstanceVersion(ctx, inst.ID, newVersion)` before creating new IAVs. Makes the operation idempotent. Any future operation that writes dependent data before a primary mutation should follow the same pattern: clean up potential stale data from previous failed attempts first.
