---
id: "mem_fce6059b"
topic: "Incoming bidirectional associations — target_entity_type_name is the CURRENT type, source_entity_type_name is the OTHER type"
tags:
  - bidirectional
  - associations
  - snapshot
  - gotcha
phase: 0
difficulty: 0.7
created_at: "2026-05-07T15:12:29.865374+00:00"
created_session: 17
---
For incoming bidirectional associations in version snapshots, the field names are counterintuitive:
- `target_entity_type_name` = the CURRENT entity type (the one whose snapshot you're viewing)
- `source_entity_type_name` = the OTHER entity type (the one that defined the association)

The association model stores `target_entity_type_id` as the type it points TO. For incoming, "you" are the target, so `target_entity_type_id` is your own ID. The source (the type that defined the association) is in `source_entity_type_id/name`.

When rendering UI for incoming bidirectional associations (LinkModal dropdown, target instance loading), always use `source_entity_type_id/name` to refer to the other type. See `LinkModal.tsx` lines 51 and 97.
