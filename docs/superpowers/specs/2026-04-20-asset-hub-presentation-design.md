# PC Asset Hub — Presentation Design Spec

## Purpose

A self-contained HTML presentation explaining the PC Asset Hub to the MCP Gateway, Registry, and Catalog teams. Covers what the Asset Hub is, the gaps it fills in the MCP ecosystem, and three focus areas: MCP Tool Catalog, Virtual MCP Servers, and Gateway Registration.

## Audience

Technical teams already familiar with MCP, Kubernetes, and CRDs. No need to explain those foundations — focus on what the Asset Hub adds.

## Deliverable

Single HTML file at `docs/presentation/asset-hub-overview.html`. Self-contained: all styles inline, diagrams in CSS/SVG, fonts loaded from Google Fonts CDN. No external image dependencies.

## Slide Structure (9 slides)

### Slide 1: Title
- **Background:** Dark (`#151515`)
- **Content:** "PC Asset Hub" as display heading, "Project Catalyst" tagline, Red Hat logo mark (SVG inline), Josh Salomon attribution
- **Animation:** Fade-up stagger on heading, tagline, attribution

### Slide 2: The Gap in the MCP Ecosystem
- **Background:** White
- **Content:** Brief framing: MCP gives us servers and tools, but the ecosystem is missing critical pieces. Three gap callouts:
  1. No registry — no way to track what servers and tools are available
  2. No curation — no way to select which tools to expose for a given use case
  3. No lifecycle — no governance over changes to what's deployed
- **Animation:** Heading fades up, then three callout cards animate in one by one (staggered slide-right)
- **Accent color:** Red (`#ee0000`) for the gap highlights

### Slide 3: Asset Hub — Schema + Catalog
- **Background:** White
- **Content:** Two-panel layout. Left panel: "Schema Management" — define entity types (MCP Server, Tool, Virtual Server), their attributes, and relationships between them. Right panel: "Catalog Management" — populate with real data, curate which assets are included, validate, publish. Animated arrow between them showing schema feeds catalog.
- **Animation:** Left panel slides in from left, right panel slides in from right, arrow fades in between
- **Accent color:** Teal (`#37a3a3`)

### Slide 4: The Data Model
- **Background:** White
- **Content:** UML-style entity relationship diagram built in CSS:
  - MCP Server entity box (attributes: endpoint, containerized, image URL, execution command)
  - MCP Tool entity box (attributes: type [read/write/readwrite], idempotent)
  - Containment arrow from MCP Server → MCP Tool (1 to 1..n)
  - Virtual Server entity box (name, description only)
  - Directional reference from Virtual Server → MCP Tool (1..n to 0..n)
  - Caption: "All defined dynamically — no code changes needed"
- **Animation:** Entity boxes appear one by one (MCP Server first, then Tool with containment arrow drawing in, then Virtual Server with reference arrow). Staggered build-up.
- **Accent color:** Teal (`#63bdbd`) for entity borders, arrows

### Slide 5: MCP Tool Catalog
- **Background:** White
- **Content:** Animated diagram showing the curation flow:
  - Left: Pool of available MCP servers as boxes (Jira with tools: get-story, add-watcher; GitHub with tools: get-issue, create-PR)
  - Center: Arrow/flow indicating "Create Catalog → Select Servers"
  - Right: Catalog container showing the selected servers and their tools inside it
  - Caption: "Your curated inventory of what's available in this deployment"
- **Animation:** Server boxes appear on left, then animate/flow into the catalog container on the right. Tools follow their servers (containment).
- **Accent color:** Orange (`#f5921b`)

### Slide 6: Virtual MCP Servers
- **Background:** White
- **Content:** Diagram showing composition from the catalog:
  - Top: Catalog with all tools visible
  - Bottom-left: "Read-Only Reporting" virtual server — lines connect to get-issue and get-story (highlighted)
  - Bottom-right: "DevOps Actions" virtual server — lines connect to create-PR and add-watcher (highlighted)
  - Caption: "Same catalog, different views, different consumers"
- **Animation:** Catalog appears, then virtual server boxes appear below, then connection lines draw from virtual servers to their selected tools with tools highlighting
- **Accent color:** Purple (`#5e40be`, `#876fd4`)

### Slide 7: Catalog Lifecycle
- **Background:** White
- **Content:** Horizontal state flow diagram:
  - Main flow: Draft → Validate → Testing → Production → Publish
  - Update flow below: Copy → Edit → Validate → Atomic Replace (with "data version +1" annotation)
  - "Old catalog archived for rollback" note
- **Animation:** States light up sequentially left to right. Then update flow appears below with its own sequential animation.
- **Accent color:** Teal (`#37a3a3`) for state nodes, orange for the update flow

### Slide 8: Gateway Registration
- **Background:** White
- **Content:** Left-to-right architecture diagram:
  - Asset Hub publishes → Catalog CR created on cluster (show CR snippet: endpoint, catalog name, data version)
  - Gateway watches CR → discovers catalog → calls API for full tool metadata
  - On update: data version increments → gateway invalidates cache → re-fetches
  - Caption: "Zero-downtime catalog updates"
- **Animation:** Architecture builds left to right: Asset Hub box appears, arrow draws to CR, CR appears with fields, arrow draws to Gateway, Gateway appears. Then update flow animates below.
- **Accent color:** Teal (`#004d4d`, `#63bdbd`)

### Slide 9: Under the Hood (Closing)
- **Background:** Dark (`#151515`)
- **Content:** Tech stats in a clean grid:
  - Go + Echo backend
  - React + PatternFly 6 UI
  - PostgreSQL storage
  - K8s Operator
  - 2,700+ tests, 97%+ coverage
  - "Schema-agnostic: catalog MCP tools today, models and prompts tomorrow"
  - Red Hat logo mark as bookend, repo link
- **Animation:** Stats grid items fade-up with stagger
- **Accent color:** Red (`#ee0000`) for emphasis

## Visual Design

### Colors
- **Background:** White (`#ffffff`) for content slides 2–8; dark (`#151515`) for bookend slides 1, 9
- **Text on light:** Primary `#151515`, secondary `#4d4d4d`, muted `#707070`
- **Text on dark:** Primary `#ffffff`, secondary `#c7c7c7`, muted `#8c8c8c`
- **Brand red:** `#ee0000` (primary accent, used sparingly for emphasis)
- **Section accents:** Teal (`#37a3a3`, `#63bdbd`), Orange (`#f5921b`, `#fccb8f`), Purple (`#5e40be`, `#876fd4`)
- **Diagram borders:** `#e0e0e0` on light backgrounds, accent colors for highlighted elements
- **Diagram fills:** Light tints of accent colors (e.g., `#daf2f2` for teal entities, `#ece6ff` for purple)

### Typography
- **Headings:** Red Hat Display (Google Fonts), weight 700/900
- **Body:** Red Hat Text (Google Fonts), weight 400/500
- **Monospace:** Red Hat Mono for code/CR snippets
- **Scale:** Display ~3.5rem, H2 ~2rem, body ~1.1rem, labels ~0.75rem

### Layout
- Full viewport slides (100vw x 100vh)
- Content max-width: 1100px, centered
- Generous padding: ~5rem horizontal on desktop, responsive down to 1.25rem

## Animation System

### Entrance Animations
- `fade-up`: opacity 0 → 1, translateY(24px) → 0
- `slide-right`: opacity 0 → 1, translateX(-40px) → 0
- `slide-left`: opacity 0 → 1, translateX(40px) → 0
- `scale-in`: opacity 0 → 1, scale(0.92) → 1
- `fade-in`: opacity 0 → 1
- `draw-line`: SVG stroke-dashoffset animation for arrows/connections

### Stagger System
- Elements within `[data-stagger]` containers get `--index` CSS variable
- Delay: `calc(var(--index) * 100ms)`
- Easing: `cubic-bezier(0.22, 1, 0.36, 1)` (pop-out)

### Diagram Build Animations
- Entity boxes: scale-in with stagger
- Arrows/connections: stroke-dashoffset from 100% to 0 (draw effect)
- Labels on arrows: fade-in after arrow completes
- Highlight effect: background-color transition to accent tint

### Slide Transitions
- Outgoing: opacity to 0, slight scale down (0.96), 220ms
- Incoming: opacity to 1, scale to 1, 350ms with pop easing
- Background color crossfade: 600ms

## Interaction

### Navigation
- Arrow keys (left/right), Page Up/Down, Home/End
- Number keys for direct slide access
- Click navigation dots
- Prev/Next buttons

### Controls
- Bottom-center pill with prev/next buttons, dot indicators, slide counter
- On light slides: dark control bar; on dark slides: same style as walkthrough.html
- Escape key for overview grid (optional)

## Responsive
- Breakpoint at 768px: reduce font sizes, tighter padding
- Diagrams scale down gracefully (flexbox/grid based)

## Accessibility
- `aria-label` on all slides
- `aria-current` on active dot
- `aria-live="polite"` on counter
- `prefers-reduced-motion`: all animations collapse to simple fade, no stagger
- Print styles: all slides visible, page-break between, light theme forced

## File Location
`docs/presentation/asset-hub-overview.html`
