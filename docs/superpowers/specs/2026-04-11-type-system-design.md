# Comprehensive Type System with Versioning — Design Spec

## Context

The AI Asset Hub currently supports only 3 attribute types: `string`, `number`, `enum`. Enums are shared mutable objects with no versioning (TD-58) — mutations affect all catalog versions retroactively, risking data corruption. This design introduces a comprehensive type system where **type definitions** are first-class versioned objects that replace enums and support 9 base types with optional constraints. Type definition versions are pinned in catalog versions, solving TD-58 and enabling rich, reusable types.

**Decisions made:**
- Type definitions are versioned, CV-pinned objects (not inline on attributes)
- Enums become a kind of type definition (unified system)
- System type definitions (immutable) exist for each base type
- Attributes reference a `type_definition_version_id` (replaces `type` + `enum_id`)
- Constraints stored as JSONB on TypeDefinitionVersion
- List element types are base-types-only for now (expandable later)
- Enums tab replaced by Types tab in UI
- No migration needed — data can be wiped and recreated

---

## 1. Data Model

### TypeDefinition (replaces Enum)

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| name | VARCHAR | Unique, human-readable name |
| description | VARCHAR | Optional description |
| base_type | VARCHAR(20) | One of: string, integer, number, boolean, date, url, enum, list, json |
| system | BOOL | True for immutable built-in types |
| created_at | TIMESTAMP | |
| updated_at | TIMESTAMP | |

### TypeDefinitionVersion (new, like EntityTypeVersion)

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| type_definition_id | UUID FK | References TypeDefinition |
| version_number | INT | Auto-incremented |
| constraints | JSONB | Type-specific constraints (see below) |

**Unique constraint:** `(type_definition_id, version_number)`

### Constraints schema per base_type

```json
// string
{"max_length": 255, "multiline": true, "pattern": "^[A-Z]+"}

// integer
{"min": 0, "max": 65535}

// number
{"min": 0.0, "max": 100.0}

// boolean
{}

// date
{}

// url
{}

// enum
{"values": ["red", "green", "blue"]}

// list
{"element_base_type": "string", "max_length": 10}

// json
{"schema": {}}  // optional JSON Schema, nice-to-have
```

All fields within constraints are optional. Empty `{}` means no constraints.

### System Type Definitions (seeded on startup)

| Name | Base Type | Constraints |
|------|-----------|-------------|
| string | string | {} |
| integer | integer | {} |
| number | number | {} |
| boolean | boolean | {} |
| date | date | {} |
| url | url | {} |

System types: `system=true`, one immutable version (V1), cannot be deleted/modified.

### Attribute Model (modified)

**Before:**
```
id, entity_type_version_id, name, description, type, enum_id, ordinal, required
```

**After:**
```
id, entity_type_version_id, name, description, type_definition_version_id, ordinal, required
```

`type` and `enum_id` replaced by single `type_definition_version_id` FK.

### CatalogVersionTypePin (new)

| Column | Type | Description |
|--------|------|-------------|
| id | UUID | Primary key |
| catalog_version_id | UUID FK | References CatalogVersion |
| type_definition_version_id | UUID FK | References TypeDefinitionVersion |

System types don't need pinning (immutable). When adding an entity type pin whose attributes use unpinned custom types, auto-pin the latest version.

### InstanceAttributeValue (modified)

**Before:** `ValueString`, `ValueNumber`, `ValueEnum`
**After:** `ValueString`, `ValueNumber`, `ValueJSON`

| Value column | Used by |
|-------------|---------|
| ValueString | string, url, date (ISO 8601), boolean ("true"/"false"), enum |
| ValueNumber | number (float64), integer (validated as whole number) |
| ValueJSON | list (JSON array), json (JSON object) |

`ValueEnum` removed (merged into `ValueString`).

---

## 2. API Changes

### Type Definition CRUD (new endpoints)

```
POST   /api/meta/v1/type-definitions                     -- create type definition (V1)
GET    /api/meta/v1/type-definitions                     -- list all
GET    /api/meta/v1/type-definitions/:id                 -- get by ID
PUT    /api/meta/v1/type-definitions/:id                 -- update (creates new version)
DELETE /api/meta/v1/type-definitions/:id                 -- delete (if not referenced)
GET    /api/meta/v1/type-definitions/:id/versions        -- list versions
GET    /api/meta/v1/type-definitions/:id/versions/:v     -- get specific version
```

**CreateTypeDefinitionRequest:**
```json
{
  "name": "guardrailID",
  "description": "Hex string identifier for guardrails",
  "base_type": "string",
  "constraints": {
    "max_length": 12,
    "pattern": "[0-9A-F]*"
  }
}
```

**UpdateTypeDefinitionRequest** (creates new version):
```json
{
  "description": "Updated description",
  "constraints": {
    "max_length": 16,
    "pattern": "[0-9A-F]*"
  }
}
```

### CV Type Pins (new endpoints)

```
POST   /api/meta/v1/catalog-versions/:id/type-pins       -- add type pin
DELETE /api/meta/v1/catalog-versions/:id/type-pins/:pid   -- remove type pin
GET    /api/meta/v1/catalog-versions/:id/type-pins        -- list type pins
```

### Attribute API Changes

**CreateAttributeRequest** (modified):
```json
{
  "name": "guardrail_id",
  "description": "The guardrail identifier",
  "type_definition_version_id": "uuid-of-guardrailID-v1",
  "required": true
}
```

Replaces `type` + `enum_id` fields.

**AttributeResponse** includes resolved type info:
```json
{
  "name": "guardrail_id",
  "type_definition_version_id": "...",
  "type_name": "guardrailID",
  "base_type": "string",
  "constraints": {"max_length": 12, "pattern": "[0-9A-F]*"},
  "required": true
}
```

### Instance Value Changes

Instance creation/update accepts values based on base_type:
- string/url/date/enum: string value
- integer/number: numeric value
- boolean: true/false
- list: JSON array
- json: JSON object

---

## 3. Validation

### Type Definition Constraints Validation (on create/update type definition)

| Base Type | Constraint | Validation |
|-----------|-----------|------------|
| string | max_length | Positive integer |
| string | multiline | Boolean |
| string | pattern | Valid regex |
| integer | min, max | Integer, min <= max |
| number | min, max | Float, min <= max |
| enum | values | Non-empty string array, unique values |
| list | element_base_type | Must be a valid base type (no list/json/enum) |
| list | max_length | Positive integer |
| json | schema | Valid JSON Schema (nice-to-have, defer) |

### Instance Value Validation (in catalog validation service)

| Base Type | Validation |
|-----------|------------|
| string | Length <= max_length (if set), matches pattern (if set) |
| integer | Whole number, min <= value <= max (if set) |
| number | min <= value <= max (if set) |
| boolean | Must be "true" or "false" |
| date | Must be valid ISO 8601 date/datetime |
| url | Must be valid URL |
| enum | Must be one of the values in the pinned type definition version |
| list | Valid JSON array, length <= max_length (if set), each element valid for element_base_type |
| json | Valid JSON, optionally validate against schema (nice-to-have) |

---

## 4. UI Changes

### Schema Management -- Types Tab (replaces Enums tab)

**Type list view:**
- Shows all type definitions: name, base type, latest version, description
- System types shown with "System" badge (cannot edit/delete)
- Custom types: create, edit (new version), delete
- Filter by base type

**Type detail view:**
- Name, description, base type, current constraints
- Edit constraints creates new version (copy-on-write)
- Version history panel
- Shows which attributes/entity types reference this type

**Create type definition:**
- Name, description, base type dropdown
- Dynamic constraints form based on selected base type:
  - string: max_length input, multiline toggle, pattern input
  - integer/number: min/max inputs
  - enum: ordered values list (add/remove/reorder)
  - list: element type dropdown, max_length input
  - json: schema textarea (optional)

### Attribute Management -- Type Selector

**Add Attribute modal** (modified):
- Name, description, required toggle
- Type selector: grouped dropdown or searchable selector
  - System types first (string, integer, number, boolean, date, url)
  - Custom types grouped by base type
  - "Create new type..." inline action (opens type creation form)

### Instance Forms -- Dynamic Rendering

| Base Type | Form Control |
|-----------|-------------|
| string (not multiline) | TextInput (with max_length indicator if set) |
| string (multiline) | TextArea |
| integer | NumberInput (step=1, min/max if set) |
| number | NumberInput (min/max if set) |
| boolean | Switch/Toggle |
| date | DatePicker |
| url | TextInput with URL validation |
| enum | Select dropdown with values |
| list | Repeatable input group (add/remove items) |
| json | TextArea / code block |

### Data Viewer -- Value Display

| Base Type | Display |
|-----------|---------|
| string | Text (or pre-formatted if multiline) |
| integer/number | Formatted number |
| boolean | "Yes" / "No" |
| date | Formatted date |
| url | Clickable link |
| enum | Text value |
| list | Comma-separated or bullet list |
| json | Formatted/collapsible JSON |

---

## 5. Copy-on-Write Versioning (solves TD-58)

When a custom type definition is modified:
1. System creates new `TypeDefinitionVersion` with `version_number++`
2. Existing CV type pins still reference the old version
3. To use the new version: update the CV's type pin to the new version
4. Instances in catalogs pinned to the old CV continue validating against the old type definition version

This fully resolves TD-58 -- enum value changes are no longer destructive.

---

## 6. Removed/Changed Tables

| Table | Action |
|-------|--------|
| `enums` | Removed -- replaced by `type_definitions` with `base_type=enum` |
| `enum_values` | Removed -- values stored in TypeDefinitionVersion.constraints.values |
| `attributes.type` | Removed -- replaced by `type_definition_version_id` FK |
| `attributes.enum_id` | Removed -- replaced by `type_definition_version_id` FK |
| `instance_attribute_values.value_enum` | Removed -- enum values stored in `value_string` |

New tables: `type_definitions`, `type_definition_versions`, `catalog_version_type_pins`

---

## 7. Files to Modify

### Backend (Go)
- `internal/domain/models/models.go` -- new TypeDefinition, TypeDefinitionVersion models; modify Attribute, InstanceAttributeValue
- `internal/domain/repository/` -- new type_definition.go repo interface; remove enum.go
- `internal/infrastructure/gorm/models/` -- GORM models
- `internal/infrastructure/gorm/repository/` -- GORM repo implementations
- `internal/service/meta/` -- new type_definition_service.go; modify attribute_service.go; remove enum_service.go
- `internal/service/operational/` -- modify instance_service.go, validation_service.go
- `internal/api/meta/` -- new type_definition_handler.go; modify attribute_handler.go, catalog_version_handler.go; remove enum_handler.go
- `internal/api/dto/dto.go` -- new DTOs for type definitions
- `cmd/api-server/main.go` -- register new routes, seed system types

### Frontend (TypeScript/React)
- `ui/src/types/index.ts` -- new TypeDefinition types; modify Attribute types
- `ui/src/api/client.ts` -- new API client methods for type definitions
- `ui/src/pages/meta/` -- new TypeDefinitionListPage, TypeDefinitionDetailPage; remove EnumListPage, EnumDetailPage
- `ui/src/components/` -- modify AttributeFormFields, CreateInstanceModal, EditInstanceModal; new type-specific form controls
- `ui/src/pages/meta/CatalogVersionDetailPage.tsx` -- add type pins to BOM
- `ui/src/App.tsx` -- update routes (Types tab replaces Enums)
- `ui/src/pages/operational/` -- type-aware value rendering

### Tests
- All existing enum tests become type definition tests
- New tests for each base type's validation
- New tests for type definition versioning and CV pinning
- Browser tests for Types tab UI
- Live browser tests for type-aware instance forms

### Docs
- `PRD.md` -- update section 2.5 (Attributes) and 6.1.4 (Enum Management)
- `docs/test-plan.md` and `docs/test-plan-detailed.md`
- `docs/td-log.md` -- resolve TD-58

---

## 8. Verification

1. **Unit tests**: Type definition CRUD, versioning, constraint validation per base type
2. **Integration tests**: Type definition + attribute + CV pin + instance value round-trip
3. **API tests**: All new endpoints, attribute creation with type_definition_version_id
4. **Browser tests**: Types tab (create/edit/delete), attribute creation with type selector, instance forms for all types
5. **Live browser tests**: End-to-end type definition -> attribute -> instance -> validation flow
6. **Manual verification**: `HEADLESS=false SLOWMO=300 make test-e2e` to watch type-aware forms

---

## 9. Implementation Phases

### Phase 1: Foundation
- DB schema (new tables, modified tables, drop enum tables)
- Domain models and repository interfaces
- GORM implementations
- Seed system types on startup
- Type definition CRUD service + handler + routes

### Phase 2: Attribute Integration
- Modify attribute model (type_definition_version_id)
- Modify attribute service (copy-on-write with type refs)
- CV type pin support (service, handler, routes)
- Auto-pin when adding entity type pins
- Version snapshot updates (resolve type info)

### Phase 3: Instance Values
- Modify instance attribute value storage (drop ValueEnum, add ValueJSON)
- Type-aware value mapping in instance service
- Validation service updates for all base types
- Instance CRUD with type-aware value handling

### Phase 4: UI -- Types Tab
- TypeDefinitionListPage (replaces EnumListPage)
- TypeDefinitionDetailPage (replaces EnumDetailPage)
- Type creation form with dynamic constraints
- Version history
- Routes and navigation

### Phase 5: UI -- Attribute & Instance Integration
- Modify Add Attribute modal (type selector)
- Modify instance form rendering (AttributeFormFields)
- Type-specific form controls (DatePicker, Switch, Select, repeatable list)
- Data viewer type-aware rendering

### Phase 6: Tests & Polish
- Full test coverage (unit, integration, API, browser, live)
- PRD updates
- TD-58 resolved
- Documentation
