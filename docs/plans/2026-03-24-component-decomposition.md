# UI Component Decomposition Plan (TD-23 + TD-35)

## Problem

Three page components have grown too large to maintain, test, and extend:

| Component | Lines | useState | Modals | Handlers |
|-----------|-------|----------|--------|----------|
| `CatalogDetailPage.tsx` (meta) | 1208 | 71 | 8 | 10 |
| `EntityTypeDetailPage.tsx` (meta) | 1198 | 71 | 7 | 12 |
| `OperationalCatalogDetailPage.tsx` | 428 | 15 | 0 | 5 |

Every feature addition (TD-22 system attrs, TD-42 child attrs) inflates these files further. Coverage is hard to improve because testing requires orchestrating the entire page to reach specific modal/handler paths.

## Approach

Extract **custom hooks** for state/data management and **sub-components** for modals. Each page becomes a thin orchestrator (~150-250 lines) that wires hooks to UI.

**Constraints:**
- Pure refactoring — zero behavior changes, zero API changes
- Existing browser tests must continue to pass without modification (they test pages as black boxes)
- Each phase is independently mergeable
- No new dependencies

---

## Phase 1: CatalogDetailPage.tsx (meta)

**Goal:** 1208 lines → ~200 line page + 8 new files

### Hooks to extract

#### `ui/src/hooks/useCatalogData.ts`

Manages catalog loading, pins, schema, and enum values.

**State moved:**
- `catalog`, `loading`, `error`
- `pins`, `activeTab`
- `schemaAttrs`, `schemaAssocs`
- `enumValues`

**Functions moved:**
- `loadCatalog()` — fetch catalog + pins, set initial activeTab
- `loadSchema()` — fetch snapshot for activeTab, populate schema attrs/assocs/enums
- `setActiveTab()` — tab change handler

**Interface:**
```typescript
function useCatalogData(catalogName: string | undefined, role: Role) {
  return {
    catalog, loading, error, pins, activeTab, setActiveTab,
    schemaAttrs, schemaAssocs, enumValues,
    loadCatalog, // for refresh after mutations
  }
}
```

#### `ui/src/hooks/useInstances.ts`

Manages instance list and CRUD operations.

**State moved:**
- `instances`, `instTotal`, `instLoading`
- Create modal: `createOpen`, `newInstName`, `newInstDesc`, `newInstAttrs`, `createError`
- Edit modal: `editTarget`, `editName`, `editDesc`, `editAttrs`, `editError`
- Delete modal: `deleteTarget`, `deleteError`

**Functions moved:**
- `loadInstances()` — fetch instance list for activeTab
- `handleCreate()` — create instance with attrs
- `handleEdit()` — update instance
- `handleDelete()` — delete instance
- `openCreate()` / `openEdit(inst)` / `openDelete(inst)` — modal openers

**Interface:**
```typescript
function useInstances(catalogName: string | undefined, entityTypeName: string, schemaAttrs: SnapshotAttribute[]) {
  return {
    instances, instTotal, instLoading,
    createOpen, openCreate, closeCreate, newInstName, setNewInstName, ...,
    editTarget, openEdit, closeEdit, editName, setEditName, ...,
    deleteTarget, openDelete, closeDelete,
    handleCreate, handleEdit, handleDelete,
    loadInstances, // for refresh
  }
}
```

#### `ui/src/hooks/useInstanceDetail.ts`

Manages selected instance detail panel — children, references, parent name.

**State moved:**
- `selectedInstance`, `parentName`
- `children`, `childrenLoading`
- `forwardRefs`, `reverseRefs`, `refsLoading`

**Functions moved:**
- `selectInstance(inst)` — load parent name, children, references
- Parent name resolution (API call with fallback)

**Interface:**
```typescript
function useInstanceDetail(catalogName: string | undefined, entityTypeName: string, schemaAssocs: SnapshotAssociation[]) {
  return {
    selectedInstance, selectInstance, clearSelection,
    parentName, children, childrenLoading,
    forwardRefs, reverseRefs, refsLoading,
  }
}
```

### Modal components to extract

Each modal gets its own file with self-contained state for form fields (the hook provides only the open/close/submit interface; form-level state stays in the modal).

#### `ui/src/components/CreateInstanceModal.tsx`

**Props:**
```typescript
interface Props {
  isOpen: boolean
  onClose: () => void
  entityTypeName: string
  schemaAttrs: SnapshotAttribute[]
  enumValues: Record<string, string[]>
  onSubmit: (name: string, description: string, attrs: Record<string, unknown>) => Promise<void>
  error: string | null
}
```

**Internal state:** `newInstName`, `newInstDesc`, `newInstAttrs`

Renders system attrs (name/description) mapped to top-level fields, custom attrs from schema.

#### `ui/src/components/EditInstanceModal.tsx`

**Props:**
```typescript
interface Props {
  instance: EntityInstance | null  // null = closed
  onClose: () => void
  schemaAttrs: SnapshotAttribute[]
  enumValues: Record<string, string[]>
  onSubmit: (version: number, name: string, description: string, attrs: Record<string, unknown>) => Promise<void>
  error: string | null
}
```

**Internal state:** `editName`, `editDesc`, `editAttrs`

Pre-populates from instance on open.

#### `ui/src/components/AddChildModal.tsx`

**Props:**
```typescript
interface Props {
  isOpen: boolean
  onClose: () => void
  catalogName: string
  parentInstance: EntityInstance
  containmentAssocs: SnapshotAssociation[]
  pins: CatalogVersionPin[]
  onSubmit: (childType: string, mode: 'create' | 'adopt', data: CreateOrAdoptData) => Promise<void>
  error: string | null
}
```

**Internal state:** `childTypeName`, `addChildMode`, `newChildName`, `newChildDesc`, `newChildAttrs`, `childSchemaAttrs`, `childEnumValues`, `adoptInstanceId`, `availableInstances`

Loads child schema on type selection. Handles create vs adopt mode.

#### `ui/src/components/LinkModal.tsx`

**Props:**
```typescript
interface Props {
  isOpen: boolean
  onClose: () => void
  catalogName: string
  instance: EntityInstance
  entityTypeName: string
  schemaAssocs: SnapshotAssociation[]
  pins: CatalogVersionPin[]
  onSubmit: (targetId: string, assocName: string) => Promise<void>
  error: string | null
}
```

**Internal state:** `linkAssocName`, `linkTargetId`, `linkTargetInstances`

Loads target instances when association selected.

#### `ui/src/components/SetParentModal.tsx`

**Props:**
```typescript
interface Props {
  isOpen: boolean
  onClose: () => void
  catalogName: string
  instance: EntityInstance
  pins: CatalogVersionPin[]
  onSubmit: (parentType: string, parentId: string) => Promise<void>
  onClearParent: () => Promise<void>
  error: string | null
}
```

**Internal state:** `parentTypeName`, `parentInstanceId`, `parentInstances`

### Remaining in CatalogDetailPage.tsx (~200 lines)

- Imports and hook wiring
- Tab rendering (entity type tabs with instance tables)
- Instance detail panel layout (delegates to `useInstanceDetail` for data)
- Toolbar buttons (create, validate, publish/unpublish, copy, replace)
- Validation results section (uses existing `useValidation` hook)
- Copy/Replace modals (simple enough to keep inline, or extract later)
- `EnumSelect` helper component (already self-contained, stays)

### Test strategy

- **Existing browser tests pass unchanged** — they render the full page via MemoryRouter and interact with it as a user would. The refactoring doesn't change any behavior.
- **New hook tests (optional, future):** Each hook can be tested in isolation with `renderHook` and mock API calls. This would be simpler than page-level tests and could improve coverage further.

---

## Phase 2: EntityTypeDetailPage.tsx (meta)

**Goal:** 1198 lines → ~250 line page + 8 new files

### Hooks to extract

#### `ui/src/hooks/useEntityTypeData.ts`

**State moved:** `entityType`, `loading`, `error`, `attributes`, `associations`, `versions`, `enums`, `entityTypes`

**Functions moved:** `loadEntityType()`, `loadAttributes()`, `loadAssociations()`, `loadVersions()`

#### `ui/src/hooks/useAttributeManagement.ts`

**State moved:** `addAttrOpen`, `attrName`, `attrDesc`, `attrType`, `attrEnumId`, `attrRequired`, `addAttrError`, `editAttrOpen`, `editAttrName`, `editAttrDesc`, `editAttrType`, `editAttrEnumId`, `editAttrRequired`, `editAttrError`

**Functions moved:** `handleAddAttribute()`, `handleRemoveAttribute()`, `handleReorderAttribute()`, `handleEditAttribute()`, `openEditAttr()`

#### `ui/src/hooks/useAssociationManagement.ts`

**State moved:** `addAssocOpen`, `assocName`, `assocTargetId`, `assocType`, `assocSourceRole`, `assocTargetRole`, `assocSourceCardinality`, `assocTargetCardinality`, `assocSourceCardCustom`, `assocTargetCardCustom`, `assocSourceCardMin/Max`, `assocTargetCardMin/Max`, `addAssocError`, `editAssocTarget`, `editAssocError`

**Functions moved:** `handleAddAssociation()`, `handleDeleteAssociation()`, `handleEditAssociationSave()`

### Modal components to extract

#### `ui/src/components/AddAttributeModal.tsx`

Form: name, description, type (string/number/enum), enum selector, required checkbox.

#### `ui/src/components/EditAttributeModal.tsx`

Same fields, pre-populated from selected attribute.

#### `ui/src/components/AddAssociationModal.tsx`

Form: name, target entity type, association type, source/target roles, cardinality (standard + custom).

#### `ui/src/components/CopyAttributesModal.tsx`

Source entity type selector, version picker, attribute checkbox list with conflict detection.

#### `ui/src/components/RenameEntityTypeModal.tsx`

Name input + deep copy warning flow.

Note: `EditAssociationModal` already exists as a shared component — no extraction needed.

### Remaining in EntityTypeDetailPage.tsx (~250 lines)

- Hook wiring, tab layout (Attributes, Associations, Version History, Diagram)
- Attribute table rendering
- Association table rendering
- Version history table + diff display
- Copy/Delete entity type handlers (simple, inline)
- Toolbar buttons

---

## Phase 3: OperationalCatalogDetailPage.tsx

**Goal:** 428 lines → ~150 line page + 2 new files

### Hook to extract

#### `ui/src/hooks/useContainmentTree.ts`

**State moved:** `tree`, `treeLoading`, `expandedNodes`, `selectedInstance`, `forwardRefs`, `reverseRefs`, `refsLoading`

**Functions moved:** `loadTree()`, `toggleNode()`, `selectTreeNode()`, `navigateToTreeNode()`

### Component to extract

#### `ui/src/components/InstanceDetailPanel.tsx`

Renders the right pane: instance attributes, containment info, forward/reverse references with clickable links, breadcrumb.

**Props:** `instance`, `entityTypeName`, `forwardRefs`, `reverseRefs`, `refsLoading`, `onNavigateToRef`

### Remaining in OperationalCatalogDetailPage.tsx (~150 lines)

- Catalog loading (simple — name, CV label, validation status)
- Two-pane layout: tree panel (left) + detail panel (right)
- Tab bar (Overview, Tree Browser, Schema Diagram)

---

## Execution order

| Phase | Files | Est. effort | Risk |
|-------|-------|-------------|------|
| Phase 1 (CatalogDetailPage) | 8 new, 1 modified | 3-4 hours | Medium — most complex, most modals |
| Phase 2 (EntityTypeDetailPage) | 8 new, 1 modified | 2-3 hours | Low — same pattern as Phase 1 |
| Phase 3 (OperationalCatalogDetailPage) | 2 new, 1 modified | 1 hour | Low — smallest file, fewest concerns |

Each phase:
1. Create new hook/component files
2. Move state + logic from page → hook/component
3. Update page to import and use hooks/components
4. Run `make test-browser` — all existing tests must pass
5. Run coverage — verify no regression
6. Commit and merge

**Total: ~18 new files, ~2230 lines reorganized, 0 behavior changes.**

---

## Phase 4: Modal State Internalization + Shared Attribute Fields (Post-Refactor Polish)

Quality review of Phase 1 identified two important issues that should be addressed after the three decomposition phases complete:

### I1: Modal state internalization

**Problem:** Modals are currently "dumb prop tunnels" — form state (e.g., `newInstName`, `setNewInstName`) lives in the hooks/page and is passed down as individual prop pairs, resulting in 12-18 props per modal. The plan specified modals would own their form state internally with a clean `onSubmit(name, desc, attrs)` callback.

**Fix:** For each modal component:
1. Move form state (`useState` calls for name, description, attrs, etc.) from the hook/page INTO the modal component
2. Change `onSubmit` to accept the form values as parameters: `onSubmit: (name: string, description: string, attrs: Record<string, unknown>) => Promise<void>`
3. Remove the individual value/setter prop pairs
4. Add `onOpen` reset logic (clear form fields when modal opens)
5. This applies to all 10 modal components across Phases 1-3: `CreateInstanceModal`, `EditInstanceModal`, `AddChildModal`, `LinkModal`, `SetParentModal`, `AddAttributeModal`, `EditAttributeModal`, `AddAssociationModal`, `CopyAttributesModal`, `RenameEntityTypeModal`

**Expected result:** Modal props shrink from 12-18 to ~7 each. `useInstances` return surface shrinks from 30 to ~16 members. Page files shrink to the planned ~200 line target.

### I4: Shared attribute form fields component

**Problem:** The attribute form rendering pattern (system attr detection → enum select → number input → text input) is duplicated across `CreateInstanceModal`, `EditInstanceModal`, and `AddChildModal` (~30 lines each).

**Fix:** Extract a shared `AttributeFormFields` component:
```typescript
interface Props {
  schemaAttrs: SnapshotAttribute[]
  values: Record<string, string>
  onChange: (name: string, value: string) => void
  enumValues: Record<string, string[]>
  idPrefix: string
  includeSystem?: boolean  // false for AddChildModal (system attrs handled separately)
  systemName?: string      // for system name field value
  setSystemName?: (v: string) => void
  systemDesc?: string      // for system description field value
  setSystemDesc?: (v: string) => void
}
```

Also extract `buildTypedAttrs(rawAttrs: Record<string, string>, schemaAttrs: SnapshotAttribute[]): Record<string, unknown>` utility to centralize the string→number parsing logic used in 3 places.

### S2: Extract Copy and Replace modals

The Copy Catalog (~50 lines) and Replace Catalog (~55 lines) modals are still inline in the page. Extract to `CopyCatalogModal.tsx` and `ReplaceCatalogModal.tsx` with self-contained state and `onSubmit` callbacks. This removes ~100 lines and ~10 `useState` calls from the page.

### S3: Shrink `useInstances` return surface

After I1 is applied (modals own form state), `useInstances` return surface shrinks from 30 members to ~16. Remove the individual value/setter pairs that are no longer needed. The hook should return only: `instances`, `instTotal`, `instLoading`, `loadInstances`, `createOpen`, `openCreate`, `closeCreate`, `handleCreate`, `editTarget`, `openEdit`, `closeEdit`, `handleEdit`, `deleteTarget`, `openDelete`, `closeDelete`, `deleteError`, `handleDelete`.

### Execution

Phase 4 runs after Phases 1-3 complete. It modifies the same modal files created in those phases. Estimated effort: 2-3 hours. All existing tests must pass. Modal-level tests will need updating to match the new prop interfaces.
