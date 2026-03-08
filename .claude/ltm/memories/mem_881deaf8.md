---
id: "mem_881deaf8"
topic: "Entity Type Diagram (US-32) — implementation lessons"
tags:
  - diagram
  - topology
  - patternfly
  - US-32
  - visualization
  - lessons
phase: 0
difficulty: 0.8
created_at: "2026-03-04T22:16:34.168927+00:00"
created_session: 14
---
## Entity Type Diagram — PatternFly React Topology

### Library
- `@patternfly/react-topology` v6.4.0 (OCP Console native, already installed)
- Import `observer` from PF topology itself, NOT from `mobx-react` (not installed separately)

### Official Demo Pattern (MUST FOLLOW)
From `packages/demo-app-ts/src/demos/topologyPackageDemo/TopologyPackage.tsx`:

1. **Create controller ONCE** via `useState(() => { const c = new Visualization(); c.registerLayoutFactory(...); c.registerComponentFactory(...); return c })`
2. **Two-phase initialization**:
   - Phase 1: `controller.fromModel(model, true)` then `controller.getGraph().layout()` — in a useEffect watching data
   - Phase 2: Separate useEffect watching `controller.hasGraph()` — re-runs layout once surface has dimensions
3. **`GRAPH_LAYOUT_END_EVENT`** — listen for it to call `controller.getGraph().fit(60)` after layout completes
4. **`VisualizationSurface` renders immediately** — no hiding/showing, no opacity tricks

### Custom Node (UML Class Box)
- Use `DefaultNode` as wrapper (handles anchors, selection, shape)
- Pass `showLabel={false}` and render name+attributes as SVG `children`
- Children coordinate system: (0,0) is TOP-LEFT of the node shape
- `DefaultNode` with `NodeShape.rect` gives rectangular shape with proper anchor registration
- Wrap in `observer()` for MobX reactivity

### Custom Edge (Association Lines)
- Use `element.getStartPoint()`, `getEndPoint()`, `getBendpoints()` for SVG path — NOT node positions
- Use `EdgeConnectorArrow` component for arrowheads (from PF topology)
- For bidirectional: SVG `<marker>` defs with filled (target) and hollow (source) arrowheads
- Label positioned at 40% along edge (avoids overlap with arrowhead at target)

### Layout
- **DagreLayout** (hierarchical) — instant positioning, no animation, best for UML diagrams
- **ColaLayout** (force-directed) — iterative, causes visible node movement on render
- `nodeDistance: 120, linkDistance: 200` for readable spacing
- Layout type is a future configurable option (FF-5 in PRD)

### Key Bugs Encountered
1. **Nodes stacked at (0,0)**: Happened when using custom SVG `<g>` without `DefaultNode` — no anchor registration
2. **Arrowheads hidden behind nodes**: Same cause — edges routed center-to-center without proper anchors
3. **Empty pane on hidden tab**: PF Tabs renders all content but hides inactive — SVG surface has zero dimensions when hidden. Fix: show spinner until data loads, use `hasGraph` two-phase pattern
4. **CV diagram 401 error**: `setAuthRole(role)` must be called before API calls in child pages
5. **Tab state lost on refresh**: Persist active tab in URL via `useSearchParams`

### Architecture
- Shared `EntityTypeDiagram` component in `ui/src/components/EntityTypeDiagram.tsx`
- Main page: "Model Diagram" tab — interactive (double-click node → navigate, click edge → edit modal)
- CV detail page: "Diagram" tab — read-only, no click handlers
- Edge click on main page opens edit association modal in App.tsx (not navigation)

