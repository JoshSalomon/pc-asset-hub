---
id: "mem_db88bc9f"
topic: "FF-6 Operational UI Editing — implementation patterns and gotchas"
tags:
  - FF-6
  - operational-ui
  - editing
  - containment
  - links
  - gotchas
phase: 0
difficulty: 0.7
created_at: "2026-05-07T17:06:40.113917+00:00"
created_session: 17
---
## FF-6 Operational UI Editing — Implementation Patterns

**BumpInstanceVersion pattern**: Structural mutations (containment, links) must bump the instance version to maintain IAV (instance attribute value) consistency. `BumpInstanceVersion()` helper increments version and carries forward all IAV rows in a transaction. Used by: SetParent (bumps child + old/new parent), CreateContainedInstance (bumps parent), CreateAssociationLink/DeleteAssociationLink (bumps source, + target if bidirectional). Uses `DeleteByInstanceVersion` for idempotent cleanup.

**Bidirectional links from target side**: LinkModal must show incoming bidirectional associations too, not just outgoing. Filter: `assoc.type === 'bidirectional'` from reverse refs. Use `source_entity_type_name` (not `target`) for the label since the perspective is flipped. Load target instances from the OTHER side's entity type.

**PF6 Select in Playwright system tests**: PatternFly v6 Select components don't work with `getByRole('option')`. Use `getByText()` to find menu items inside the dropdown. Wait for the menu to be visible before clicking options. The Select toggle is a `MenuToggle` component — click it first to open.

**Contained instance disambiguation in LinkModal**: When multiple instances of the same type exist as contained children, show "name (parentName)" format using an `instanceNames` map built from the containment tree.

**SetParent/RemoveFromContainer name collision**: When removing from container or changing parent, the instance may collide with an existing root-level instance of the same name. Must check for collision BEFORE the mutation and return 409 (not let it hit DB unique constraint as 500). TD-147 tracks the TOCTOU race condition.

**Tree sort stability**: `buildNodes` in `useContainmentTree.ts` must sort by entity type name then instance name at every level to prevent tree order instability on re-renders.

**Action error clearing**: Clear `actionError` on instance selection change (not just entity type change) to prevent stale errors showing for the wrong instance.
