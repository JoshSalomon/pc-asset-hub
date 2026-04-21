# Asset Hub Overview Presentation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a 9-slide self-contained HTML presentation explaining the PC Asset Hub to MCP Gateway, Registry, and Catalog teams.

**Architecture:** Single HTML file with inline `<style>` and `<script>` blocks. CSS keyframe animations with stagger system for slide content. JS deck engine handles navigation, slide transitions, and animation triggers. Diagrams built as CSS boxes + inline SVG arrows. No build tools.

**Tech Stack:** HTML5, CSS3 (keyframes, grid, flexbox, custom properties), vanilla JS, Google Fonts (Red Hat Display/Text/Mono), inline SVG for arrows/connections.

**Spec:** `docs/superpowers/specs/2026-04-20-asset-hub-presentation-design.md`

**Reference:** `F:/Code/luminth/enterprise/walkthrough.html` (deck engine, nav, animation patterns)

**Output:** `docs/presentation/asset-hub-overview.html`

---

## File Structure

Single file: `docs/presentation/asset-hub-overview.html`

Sections within the file:
- `<style>`: CSS tokens, base/layout, animation library, controls, per-slide styles, responsive, print, reduced-motion
- `<main class="deck">`: 9 `<section class="slide">` elements
- `<nav class="deck-controls">`: Navigation bar
- `<script>`: Deck engine (nav, transitions, animation triggers, SVG draw, sequential animations)

---

## Task 1: Foundation — HTML Skeleton + CSS Tokens + Deck Engine

**Files:**
- Create: `docs/presentation/asset-hub-overview.html`

This task creates the complete structural foundation: HTML document, all CSS custom properties, base styles, slide layout system, animation keyframes, navigation controls, and the JS deck engine. All 9 slides are present as empty placeholders.

- [ ] **Step 1: Create the directory**

```bash
mkdir -p docs/presentation
```

- [ ] **Step 2: Write the foundation HTML**

Create `docs/presentation/asset-hub-overview.html` with:

**`<head>`:**
- Charset, viewport, title "PC Asset Hub — Overview"
- Google Fonts link: `Red Hat Display:wght@400;700;900`, `Red Hat Text:wght@400;500`, `Red Hat Mono:wght@400;500`
- Inline `<style>` block with all content below

**CSS tokens (`:root`):**
```css
/* Surface */
--bg-dark: #151515;
--bg-light: #ffffff;
--bg-light-alt: #f9f9f9;

/* Text on light */
--text-dark-primary: #151515;
--text-dark-secondary: #4d4d4d;
--text-dark-muted: #707070;

/* Text on dark */
--text-light-primary: #ffffff;
--text-light-secondary: #c7c7c7;
--text-light-muted: #8c8c8c;

/* Brand */
--rh-red: #ee0000;
--rh-red-dark: #a60000;
--rh-red-light: #fce3e3;

/* Accents */
--teal-50: #37a3a3;
--teal-40: #63bdbd;
--teal-70: #004d4d;
--teal-10: #daf2f2;
--orange-40: #f5921b;
--orange-20: #fccb8f;
--orange-10: #ffe8cc;
--purple-50: #5e40be;
--purple-40: #876fd4;
--purple-10: #ece6ff;

/* Diagram */
--border-diagram: #e0e0e0;
--border-subtle-dark: #383838;

/* Type */
--font-display: "Red Hat Display", sans-serif;
--font-text: "Red Hat Text", sans-serif;
--font-mono: "Red Hat Mono", monospace;

/* Scale */
--fs-display: 3.5rem;
--fs-h2: 2rem;
--fs-h3: 1.4rem;
--fs-body: 1.1rem;
--fs-body-lead: 1.25rem;
--fs-label: 0.75rem;
--fs-small: 0.85rem;

/* Spacing */
--space-xs: 0.5rem;
--space-sm: 0.75rem;
--space-md: 1rem;
--space-lg: 1.5rem;
--space-xl: 2rem;
--space-2xl: 3rem;
--space-3xl: 4.5rem;

/* Layout */
--content-max: 1100px;
--slide-padding: 5rem;

/* Animation */
--stagger-delay: 100ms;
--transition-exit: 220ms;
--transition-enter: 350ms;
--bg-cross: 600ms;
--ease-pop: cubic-bezier(0.22, 1, 0.36, 1);
```

**CSS base styles:**
- `*, *::before, *::after { box-sizing: border-box; }`
- `html, body`: margin 0, overflow hidden, font-family `var(--font-text)`, font-size `var(--fs-body)`, line-height 1.55, `-webkit-font-smoothing: antialiased`
- `main.deck`: position relative, 100vw x 100vh, overflow hidden, `transition: background var(--bg-cross) ease-in-out`
- `.slide`: position absolute, inset 0, 100% width, padding `var(--space-3xl) var(--slide-padding)`, flex column, justify center, opacity 0, visibility hidden, pointer-events none, transform `scale(0.96)`, transitions on opacity/transform/visibility matching walkthrough.html pattern
- Active slide selector: `main.deck[data-slide-current="N"] .slide[data-slide="N"]` for N=1..9 — opacity 1, visibility visible, pointer-events auto, transform scale(1), pop easing transitions
- `.slide-inner`: max-width `var(--content-max)`, width 100%, margin 0 auto

**CSS typography:**
- `.display`: font-family `var(--font-display)`, font-size `var(--fs-display)`, line-height 1.05, font-weight 900, letter-spacing -0.03em, margin 0
- `.h2`: font-family `var(--font-display)`, font-size `var(--fs-h2)`, font-weight 700, line-height 1.15, margin 0
- `.h3`: font-family `var(--font-display)`, font-size `var(--fs-h3)`, font-weight 700, line-height 1.2, margin 0
- `.lead`: font-size `var(--fs-body-lead)`, line-height 1.5, margin `var(--space-md) 0 0`, max-width 65ch
- `.label`: font-size `var(--fs-label)`, font-weight 700, letter-spacing 0.2em, text-transform uppercase
- `.mono`: font-family `var(--font-mono)`

**CSS dark/light slide helpers:**
- `.slide-dark`: background `var(--bg-dark)`, color `var(--text-light-primary)`
- `.slide-dark .lead, .slide-dark .body`: color `var(--text-light-secondary)`
- `.slide-light`: background `var(--bg-light)`, color `var(--text-dark-primary)`
- `.slide-light .lead, .slide-light .body`: color `var(--text-dark-secondary)`

**CSS slide-eyebrow:**
- `.slide-eyebrow`: font-size `var(--fs-label)`, font-weight 700, letter-spacing 0.2em, text-transform uppercase, color `var(--rh-red)`, margin-bottom `var(--space-lg)`

**CSS animation keyframes:**
```css
@keyframes fade-up { from { opacity:0; transform:translateY(24px); } to { opacity:1; transform:translateY(0); } }
@keyframes fade-in { from { opacity:0; } to { opacity:1; } }
@keyframes slide-right { from { opacity:0; transform:translateX(-40px); } to { opacity:1; transform:translateX(0); } }
@keyframes slide-left { from { opacity:0; transform:translateX(40px); } to { opacity:1; transform:translateX(0); } }
@keyframes scale-in { from { opacity:0; transform:scale(0.92); } to { opacity:1; transform:scale(1); } }
@keyframes draw-line { from { stroke-dashoffset: var(--dash-length, 200); } to { stroke-dashoffset: 0; } }
@keyframes highlight-bg { from { background-color: transparent; } to { background-color: var(--highlight-color); } }
@keyframes sequential-appear { 0% { opacity:0; transform:translateY(12px); } 100% { opacity:1; transform:translateY(0); } }
```

**CSS animation classes** (only trigger when slide has `.current`):
```css
main.deck .slide.current .anim-fade-up { animation: fade-up 400ms var(--ease-pop) both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
main.deck .slide.current .anim-fade-in { animation: fade-in 400ms ease-out both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
main.deck .slide.current .anim-slide-right { animation: slide-right 500ms var(--ease-pop) both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
main.deck .slide.current .anim-slide-left { animation: slide-left 500ms var(--ease-pop) both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
main.deck .slide.current .anim-scale-in { animation: scale-in 450ms var(--ease-pop) both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
main.deck .slide.current .anim-draw { animation: draw-line 600ms ease-out both; animation-delay: calc(var(--index,0) * var(--stagger-delay)); }
```

**CSS for sequential diagram animations** (used by slides 4-8 for build-up effects):
```css
main.deck .slide.current .seq-step {
  opacity: 0;
  transform: translateY(12px);
}
main.deck .slide.current .seq-step.visible {
  animation: sequential-appear 400ms var(--ease-pop) both;
}
```

**CSS controls** (adapted from walkthrough.html but for light/dark context):
```css
.deck-controls {
  position: fixed; bottom: 1.5rem; left: 50%; transform: translateX(-50%);
  display: flex; align-items: center; gap: var(--space-md);
  background: rgba(21,21,21,0.92); backdrop-filter: blur(8px);
  border: 1px solid var(--border-subtle-dark); border-radius: 999px;
  padding: 0.5rem 1.1rem; z-index: 50;
}
.deck-controls button {
  width: 38px; height: 38px; background: transparent; border: none;
  color: var(--text-light-secondary); font-size: 1.3rem; cursor: pointer;
  border-radius: 50%; display: inline-flex; align-items: center; justify-content: center;
  transition: background 180ms, color 180ms;
}
.deck-controls button:hover:not(:disabled) { background: rgba(238,0,0,0.15); color: var(--rh-red); }
.deck-controls button:disabled { opacity: 0.3; cursor: not-allowed; }
.deck-dots { display: flex; gap: 0.35rem; list-style: none; margin: 0; padding: 0; }
.deck-dots button {
  width: 10px; height: 10px; padding: 0; border-radius: 50%;
  background: var(--text-light-muted); transition: background 180ms, transform 180ms;
  color: transparent; font-size: 0;
}
.deck-dots button[aria-current="true"] {
  background: var(--rh-red); transform: scale(1.4);
  box-shadow: 0 0 12px rgba(238,0,0,0.5);
}
.deck-dots button:hover { background: var(--rh-red); }
.deck-counter {
  font-family: var(--font-mono); font-size: 0.8rem; color: var(--text-light-muted);
  letter-spacing: 0.06em; padding: 0 var(--space-sm); border-left: 1px solid var(--border-subtle-dark);
}
```

**CSS act tints** (background crossfade based on current slide):
```css
main.deck[data-slide-current="1"],
main.deck[data-slide-current="9"] { background: var(--bg-dark); }
main.deck[data-slide-current="2"],
main.deck[data-slide-current="3"],
main.deck[data-slide-current="4"],
main.deck[data-slide-current="5"],
main.deck[data-slide-current="6"],
main.deck[data-slide-current="7"],
main.deck[data-slide-current="8"] { background: var(--bg-light); }
```

**CSS responsive:**
```css
@media (max-width: 768px) {
  :root {
    --fs-display: 2.2rem;
    --fs-h2: 1.5rem;
    --fs-h3: 1.15rem;
    --fs-body-lead: 1.05rem;
    --slide-padding: 1.25rem;
  }
  .slide { padding: 2rem var(--slide-padding); }
  .deck-controls { padding: 0.3rem 0.6rem; gap: 0.5rem; }
  .deck-controls button { width: 32px; height: 32px; font-size: 1.05rem; }
  .deck-dots button { width: 8px; height: 8px; }
}
```

**CSS reduced motion:**
```css
@media (prefers-reduced-motion: reduce) {
  * { animation-duration: 0.01ms !important; animation-iteration-count: 1 !important; transition-duration: 180ms !important; }
  main.deck .slide.current .anim-fade-up,
  main.deck .slide.current .anim-slide-right,
  main.deck .slide.current .anim-slide-left,
  main.deck .slide.current .anim-scale-in,
  main.deck .slide.current .anim-fade-in { animation: fade-in 180ms ease-out both; }
  main.deck .slide.current .seq-step.visible { animation: fade-in 180ms ease-out both; }
}
```

**CSS print:**
```css
@media print {
  html, body, main.deck { overflow: visible; height: auto; }
  main.deck { background: #fff !important; }
  .deck-controls { display: none !important; }
  .slide {
    position: relative; opacity: 1 !important; visibility: visible !important;
    transform: none !important; pointer-events: auto;
    page-break-after: always; break-after: page;
    background: #fff; color: #151515; padding: 0.5in; min-height: auto;
  }
  .slide-dark { background: #fff !important; color: #151515 !important; }
  .display, .h2, .h3, .lead { color: #151515 !important; }
  .slide-eyebrow { color: #a60000 !important; }
}
```

**HTML `<body>`:**

`<main class="deck" id="deck" data-slide-current="1">` containing 9 `<section>` elements, each with:
```html
<section class="slide slide-dark" data-slide="1" aria-label="Slide 1: Title">
  <div class="slide-inner"><!-- content added in later tasks --></div>
</section>
```
Slides 1 and 9 get `slide-dark`, slides 2-8 get `slide-light`.

Navigation bar:
```html
<nav class="deck-controls" aria-label="Deck navigation">
  <button class="btn-prev" type="button" aria-label="Previous slide" data-action="prev">&larr;</button>
  <ol class="deck-dots" role="tablist" aria-label="Slide progress">
    <!-- 9 <li><button> elements with data-goto="1" through data-goto="9" -->
  </ol>
  <button class="btn-next" type="button" aria-label="Next slide" data-action="next">&rarr;</button>
  <span class="deck-counter" aria-live="polite">01 / 09</span>
</nav>
```

**`<script>`:**

Adapt the deck engine from walkthrough.html. Key functions:

```javascript
(function () {
  'use strict';
  const TOTAL = 9;
  const deck = document.getElementById('deck');
  const dots = Array.from(document.querySelectorAll('.deck-dots button'));
  const btnPrev = document.querySelector('.btn-prev');
  const btnNext = document.querySelector('.btn-next');
  const counter = document.querySelector('.deck-counter');
  const reducedMotion = matchMedia('(prefers-reduced-motion: reduce)').matches;
  const state = { current: 1 };

  function clamp(n) { return Math.max(1, Math.min(TOTAL, n)); }

  function render() {
    deck.setAttribute('data-slide-current', String(state.current));
    counter.textContent = String(state.current).padStart(2,'0') + ' / ' + String(TOTAL).padStart(2,'0');
    dots.forEach(function(d,i) {
      if (i+1 === state.current) d.setAttribute('aria-current','true');
      else d.removeAttribute('aria-current');
    });
    Array.from(deck.querySelectorAll('.slide')).forEach(function(s) {
      var n = parseInt(s.getAttribute('data-slide'),10);
      if (n === state.current) { s.classList.add('current'); s.removeAttribute('aria-hidden'); }
      else { s.classList.remove('current'); s.setAttribute('aria-hidden','true'); }
    });
    btnPrev.disabled = state.current === 1;
    btnNext.disabled = state.current === TOTAL;
    requestAnimationFrame(fireSlideAnimations);
  }

  function goTo(n) {
    var next = clamp(n);
    if (next === state.current) return;
    state.current = next;
    render();
    syncHash();
  }
  function next() { goTo(state.current + 1); }
  function prev() { goTo(state.current - 1); }

  function syncHash() {
    var h = '#s=' + state.current;
    if (location.hash !== h) history.replaceState(null,'',h);
  }
  function readHash() {
    var m = /^#s=(\d{1,2})$/.exec(location.hash || '');
    if (m) return clamp(parseInt(m[1],10));
    return 1;
  }

  // Stagger indices
  function applyStaggerIndices(slide) {
    Array.from(slide.querySelectorAll('[data-stagger] > *')).forEach(function(el,i) {
      el.style.setProperty('--index', String(i));
    });
  }

  // Sequential diagram animations — runs steps with delays
  function runSequentialSteps(slide) {
    var steps = Array.from(slide.querySelectorAll('.seq-step'));
    steps.forEach(function(step) { step.classList.remove('visible'); });
    if (reducedMotion) {
      steps.forEach(function(step) { step.classList.add('visible'); });
      return;
    }
    steps.forEach(function(step, i) {
      var delay = parseInt(step.getAttribute('data-seq-delay') || String((i + 1) * 400), 10);
      setTimeout(function() { step.classList.add('visible'); }, delay);
    });
  }

  // SVG draw animation setup
  function setupSVGDraw(slide) {
    Array.from(slide.querySelectorAll('.draw-path')).forEach(function(path) {
      var len = path.getTotalLength();
      path.style.setProperty('--dash-length', String(len));
      path.style.strokeDasharray = len;
      path.style.strokeDashoffset = len;
    });
  }

  function fireSlideAnimations() {
    var cur = deck.querySelector('.slide.current');
    if (!cur) return;
    applyStaggerIndices(cur);
    setupSVGDraw(cur);
    runSequentialSteps(cur);
  }

  // Keyboard
  var digitBuffer = '';
  var digitTimer = null;
  function clearBuffer() { digitBuffer = ''; if (digitTimer) { clearTimeout(digitTimer); digitTimer = null; } }
  function onKey(e) {
    var k = e.key;
    if (k==='ArrowRight'||k==='ArrowDown'||k==='PageDown'||k===' ') { e.preventDefault(); next(); clearBuffer(); return; }
    if (k==='ArrowLeft'||k==='ArrowUp'||k==='PageUp'||k==='Backspace') { e.preventDefault(); prev(); clearBuffer(); return; }
    if (k==='Home') { e.preventDefault(); goTo(1); clearBuffer(); return; }
    if (k==='End') { e.preventDefault(); goTo(TOTAL); clearBuffer(); return; }
    if (/^[0-9]$/.test(k)) {
      e.preventDefault();
      digitBuffer += k;
      if (digitTimer) clearTimeout(digitTimer);
      if (digitBuffer.length === 2) {
        var two = parseInt(digitBuffer,10);
        if (two >= 1 && two <= TOTAL) goTo(two);
        clearBuffer();
        return;
      }
      digitTimer = setTimeout(function() {
        var n = parseInt(digitBuffer,10);
        if (n >= 1 && n <= 9) goTo(n);
        clearBuffer();
      }, 600);
    }
  }

  // Wire up
  btnPrev.addEventListener('click', prev);
  btnNext.addEventListener('click', next);
  dots.forEach(function(d) {
    d.addEventListener('click', function() { goTo(parseInt(d.getAttribute('data-goto'),10)); });
  });
  window.addEventListener('keydown', onKey);

  document.addEventListener('DOMContentLoaded', function() {
    state.current = readHash();
    render();
    window.addEventListener('hashchange', function() {
      state.current = readHash();
      render();
    });
    if (deck.tabIndex < 0) deck.tabIndex = -1;
  });

  window.__deck = { goTo: goTo, next: next, prev: prev, state: state };
}());
```

- [ ] **Step 3: Verify foundation**

Open `docs/presentation/asset-hub-overview.html` in a browser. Verify:
- 9 empty slides navigable via arrow keys and dot clicks
- Counter shows "01 / 09" and updates
- Dark background on slides 1 and 9, white on 2-8
- Background crossfades smoothly between slides
- Controls bar visible at bottom center
- Google Fonts loaded (check Network tab)

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: scaffold presentation with deck engine and navigation"
```

---

## Task 2: Slide 1 (Title) + Slide 9 (Closing)

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

Both bookend slides share the dark background and similar structure.

- [ ] **Step 1: Add per-slide CSS**

In the `<style>` block, add:

```css
/* Slide 1: Title */
.s1 { text-align: center; display: flex; flex-direction: column; align-items: center; gap: var(--space-lg); }
.s1-logo { width: 180px; height: auto; }
.s1-tagline { font-family: var(--font-display); font-size: var(--fs-h3); color: var(--rh-red); font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase; }
.s1-byline { font-family: var(--font-mono); font-size: var(--fs-small); color: var(--text-light-muted); letter-spacing: 0.1em; margin-top: var(--space-xl); }

/* Slide 9: Closing */
.s9 { text-align: center; display: flex; flex-direction: column; align-items: center; gap: var(--space-md); }
.s9-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-md); margin: var(--space-xl) 0; max-width: 800px; width: 100%; }
.s9-stat { background: rgba(255,255,255,0.05); border: 1px solid var(--border-subtle-dark); border-radius: 8px; padding: var(--space-lg) var(--space-md); text-align: center; }
.s9-stat-value { font-family: var(--font-display); font-size: var(--fs-h2); font-weight: 900; color: var(--text-light-primary); }
.s9-stat-label { font-size: var(--fs-small); color: var(--text-light-muted); margin-top: var(--space-xs); }
.s9-tagline { font-size: var(--fs-body-lead); color: var(--text-light-secondary); font-style: italic; max-width: 55ch; margin-top: var(--space-lg); }
.s9-logo { width: 120px; margin-top: var(--space-xl); }

@media (max-width: 768px) {
  .s9-grid { grid-template-columns: 1fr 1fr; }
}
```

- [ ] **Step 2: Add Slide 1 content**

Replace the `slide-inner` content for `data-slide="1"`:

```html
<div class="slide-inner s1" data-stagger>
  <svg class="s1-logo anim-fade-up" viewBox="0 0 613 145" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Red Hat logo">
    <!-- Inline Red Hat fedora logo SVG path. Use the official Red Hat logomark.
         The SVG should render the Red Hat hat icon in white (#ffffff) on the dark slide. -->
  </svg>
  <h1 class="display anim-fade-up" style="color:var(--text-light-primary);">PC Asset Hub</h1>
  <p class="s1-tagline anim-fade-up">Project Catalyst</p>
  <p class="lead anim-fade-up" style="color:var(--text-light-secondary);">A metadata-driven management system for AI assets on OpenShift</p>
  <p class="s1-byline anim-fade-up">Josh Salomon</p>
</div>
```

Note: For the Red Hat logo SVG, use the official Red Hat fedora/hat logomark. A simplified inline SVG version should be created that renders the hat shape in white. If a precise SVG path is not available, use a text-based "Red Hat" wordmark styled with Red Hat Display font as a fallback:
```html
<div class="s1-logo anim-fade-up" style="font-family:var(--font-display);font-size:2.5rem;font-weight:900;color:var(--rh-red);">Red&nbsp;Hat</div>
```

- [ ] **Step 3: Add Slide 9 content**

Replace the `slide-inner` content for `data-slide="9"`:

```html
<div class="slide-inner s9" data-stagger>
  <p class="slide-eyebrow anim-fade-up" style="color:var(--rh-red);">Under the Hood</p>
  <h2 class="h2 anim-fade-up" style="color:var(--text-light-primary);">Built for Enterprise</h2>
  <div class="s9-grid">
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value">Go</div>
      <div class="s9-stat-label">Echo backend</div>
    </div>
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value" style="font-size:1.6rem;">React</div>
      <div class="s9-stat-label">PatternFly 6 UI</div>
    </div>
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value" style="font-size:1.6rem;">PostgreSQL</div>
      <div class="s9-stat-label">Relational + EAV</div>
    </div>
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value">K8s</div>
      <div class="s9-stat-label">Operator managed</div>
    </div>
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value" style="color:var(--rh-red);">2,700+</div>
      <div class="s9-stat-label">Tests &bull; 97%+ coverage</div>
    </div>
    <div class="s9-stat anim-fade-up">
      <div class="s9-stat-value" style="font-size:1.3rem;">Schema-<br/>agnostic</div>
      <div class="s9-stat-label">No code changes</div>
    </div>
  </div>
  <p class="s9-tagline anim-fade-up">"Catalog MCP tools today. Models and prompts tomorrow."</p>
  <div class="s9-logo anim-fade-up" style="font-family:var(--font-display);font-size:1.8rem;font-weight:900;color:var(--rh-red);">Red&nbsp;Hat</div>
</div>
```

- [ ] **Step 4: Verify**

Open in browser. Check:
- Slide 1: Title centered, Red Hat branding, text animates in with stagger
- Slide 9: Stats grid displays, items fade up with stagger, tagline visible
- Dark backgrounds on both slides

- [ ] **Step 5: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add title and closing slides with Red Hat branding"
```

---

## Task 3: Slide 2 — The Gap in the MCP Ecosystem

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 2: The Gap */
.s2 { max-width: 900px; }
.s2-gaps { display: flex; flex-direction: column; gap: var(--space-md); margin-top: var(--space-xl); }
.s2-gap-card {
  display: flex; align-items: flex-start; gap: var(--space-lg);
  background: var(--rh-red-light); border-left: 4px solid var(--rh-red);
  padding: var(--space-lg) var(--space-xl); border-radius: 0 8px 8px 0;
}
.s2-gap-icon {
  font-size: 1.6rem; color: var(--rh-red); flex-shrink: 0;
  width: 40px; height: 40px; display: flex; align-items: center; justify-content: center;
  background: white; border-radius: 50%; border: 2px solid var(--rh-red);
}
.s2-gap-text h3 { color: var(--rh-red-dark); margin-bottom: var(--space-xs); }
.s2-gap-text p { color: var(--text-dark-secondary); font-size: var(--fs-body); margin: 0; }
```

- [ ] **Step 2: Add Slide 2 content**

```html
<div class="slide-inner s2" data-stagger>
  <p class="slide-eyebrow anim-fade-up">The Problem</p>
  <h2 class="display anim-fade-up" style="font-size:var(--fs-h2);">MCP gives us servers and tools.<br/><span style="color:var(--rh-red);">But the ecosystem has gaps.</span></h2>
  <p class="lead anim-fade-up">The protocol defines how tools communicate. It does not define how to manage them at scale.</p>
  <div class="s2-gaps">
    <div class="s2-gap-card anim-slide-right">
      <div class="s2-gap-icon">&times;</div>
      <div class="s2-gap-text">
        <h3 class="h3">No Registry</h3>
        <p>No way to track what MCP servers and tools are available across an organization.</p>
      </div>
    </div>
    <div class="s2-gap-card anim-slide-right">
      <div class="s2-gap-icon">&times;</div>
      <div class="s2-gap-text">
        <h3 class="h3">No Curation</h3>
        <p>No way to select which tools should be exposed for a given use case or audience.</p>
      </div>
    </div>
    <div class="s2-gap-card anim-slide-right">
      <div class="s2-gap-icon">&times;</div>
      <div class="s2-gap-text">
        <h3 class="h3">No Lifecycle</h3>
        <p>No governance over how tool configurations are validated, published, or updated in production.</p>
      </div>
    </div>
  </div>
</div>
```

- [ ] **Step 3: Verify**

Navigate to slide 2. Three red-accented gap cards should stagger in from the left, one at a time.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 2 - the gap in the MCP ecosystem"
```

---

## Task 4: Slide 3 — Asset Hub: Schema + Catalog

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 3: Schema + Catalog */
.s3 { max-width: 1000px; }
.s3-panels { display: grid; grid-template-columns: 1fr auto 1fr; gap: var(--space-xl); align-items: stretch; margin-top: var(--space-xl); }
.s3-panel {
  background: var(--bg-light-alt); border: 1px solid var(--border-diagram);
  border-radius: 12px; padding: var(--space-xl); border-top: 4px solid var(--teal-50);
}
.s3-panel-title { color: var(--teal-50); margin-bottom: var(--space-md); }
.s3-panel ul { list-style: none; padding: 0; margin: var(--space-md) 0 0; }
.s3-panel li {
  padding: var(--space-sm) 0; border-bottom: 1px solid var(--border-diagram);
  color: var(--text-dark-secondary); font-size: var(--fs-body);
}
.s3-panel li:last-child { border-bottom: none; }
.s3-arrow {
  display: flex; align-items: center; justify-content: center;
  font-size: 2.5rem; color: var(--teal-50);
}

@media (max-width: 768px) {
  .s3-panels { grid-template-columns: 1fr; }
  .s3-arrow { transform: rotate(90deg); }
}
```

- [ ] **Step 2: Add Slide 3 content**

```html
<div class="slide-inner s3" data-stagger>
  <p class="slide-eyebrow anim-fade-up" style="color:var(--teal-50);">The Solution</p>
  <h2 class="h2 anim-fade-up">Two parts. One system.</h2>
  <p class="lead anim-fade-up">Define your asset model once. Populate it with real data. Publish to the cluster.</p>
  <div class="s3-panels">
    <div class="s3-panel anim-slide-right">
      <h3 class="h3 s3-panel-title">Schema Management</h3>
      <p style="color:var(--text-dark-muted);font-size:var(--fs-small);">Define the model</p>
      <ul>
        <li>Entity types <span class="mono" style="color:var(--teal-50);font-size:var(--fs-small);">MCP Server, Tool, Virtual Server</span></li>
        <li>Attributes &amp; constraints</li>
        <li>Associations &amp; cardinality</li>
        <li>Reusable type definitions</li>
      </ul>
    </div>
    <div class="s3-arrow anim-fade-in">&rarr;</div>
    <div class="s3-panel anim-slide-left" style="border-top-color:var(--orange-40);">
      <h3 class="h3 s3-panel-title" style="color:var(--orange-40);">Catalog Management</h3>
      <p style="color:var(--text-dark-muted);font-size:var(--fs-small);">Populate with data</p>
      <ul>
        <li>Create catalogs from schema</li>
        <li>Add servers, tools, associations</li>
        <li>Validate completeness</li>
        <li>Publish to Kubernetes</li>
      </ul>
    </div>
  </div>
</div>
```

- [ ] **Step 3: Verify**

Navigate to slide 3. Left panel slides right, right panel slides left, arrow fades in between. Two-panel layout is visible and responsive.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 3 - schema and catalog overview"
```

---

## Task 5: Slide 4 — The Data Model

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

This slide uses the sequential animation system (`seq-step`) to build the UML diagram piece by piece.

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 4: Data Model */
.s4 { max-width: 1000px; }
.s4-diagram { position: relative; margin-top: var(--space-xl); min-height: 340px; }
.s4-entity {
  background: white; border: 2px solid var(--teal-40); border-radius: 8px;
  position: absolute; width: 260px; box-shadow: 0 2px 12px rgba(0,0,0,0.08);
}
.s4-entity-header {
  background: var(--teal-10); padding: var(--space-sm) var(--space-md);
  border-bottom: 1px solid var(--teal-40); border-radius: 6px 6px 0 0;
  font-family: var(--font-display); font-weight: 700; color: var(--teal-70);
  font-size: var(--fs-body);
}
.s4-entity-body { padding: var(--space-sm) var(--space-md); }
.s4-entity-body div {
  font-size: var(--fs-small); color: var(--text-dark-secondary);
  padding: 2px 0; display: flex; align-items: center; gap: var(--space-xs);
}
.s4-entity-body .attr-required { color: var(--rh-red); font-weight: 700; }
.s4-entity-body .attr-type { color: var(--text-dark-muted); font-family: var(--font-mono); font-size: 0.75rem; }
.s4-entity.s4-virtual { border-color: var(--purple-40); }
.s4-entity.s4-virtual .s4-entity-header { background: var(--purple-10); color: var(--purple-50); border-bottom-color: var(--purple-40); }
.s4-svg { position: absolute; inset: 0; width: 100%; height: 100%; pointer-events: none; }
.s4-caption { text-align: center; margin-top: var(--space-lg); color: var(--text-dark-muted); font-style: italic; font-size: var(--fs-body); }

@media (max-width: 768px) {
  .s4-diagram { min-height: auto; display: flex; flex-direction: column; gap: var(--space-md); }
  .s4-entity { position: relative; width: 100%; }
  .s4-svg { display: none; }
}
```

- [ ] **Step 2: Add Slide 4 content**

```html
<div class="slide-inner s4">
  <p class="slide-eyebrow anim-fade-up" style="color:var(--teal-50);">Data Model</p>
  <h2 class="h2 anim-fade-up">Entities and relationships</h2>
  <div class="s4-diagram">
    <!-- MCP Server entity -->
    <div class="s4-entity seq-step" data-seq-delay="200" style="top:0;left:0;">
      <div class="s4-entity-header">MCP Server</div>
      <div class="s4-entity-body">
        <div><span class="attr-required">*</span> endpoint <span class="attr-type">url</span></div>
        <div>containerized <span class="attr-type">boolean</span></div>
        <div>image URL <span class="attr-type">url</span></div>
        <div>exec command <span class="attr-type">string</span></div>
      </div>
    </div>

    <!-- MCP Tool entity -->
    <div class="s4-entity seq-step" data-seq-delay="800" style="top:0;right:0;">
      <div class="s4-entity-header">MCP Tool</div>
      <div class="s4-entity-body">
        <div>type <span class="attr-type">enum [read, write, readwrite]</span></div>
        <div>idempotent <span class="attr-type">boolean</span></div>
      </div>
    </div>

    <!-- Virtual Server entity -->
    <div class="s4-entity s4-virtual seq-step" data-seq-delay="1800" style="bottom:0;left:50%;transform:translateX(-50%);">
      <div class="s4-entity-header">Virtual Server</div>
      <div class="s4-entity-body">
        <div style="color:var(--text-dark-muted);font-style:italic;">name + description only</div>
      </div>
    </div>

    <!-- SVG arrows -->
    <svg class="s4-svg" viewBox="0 0 1000 340" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <!-- Containment: MCP Server → MCP Tool -->
      <g class="seq-step" data-seq-delay="1200">
        <line x1="270" y1="60" x2="720" y2="60" stroke="#63bdbd" stroke-width="2" class="draw-path" />
        <polygon points="720,55 735,60 720,65" fill="#63bdbd" />
        <rect x="420" y="45" width="120" height="22" rx="4" fill="white" stroke="#63bdbd" stroke-width="1" />
        <text x="480" y="60" text-anchor="middle" fill="#004d4d" font-family="Red Hat Mono, monospace" font-size="11">1 ◆──── 1..n</text>
      </g>
      <!-- Reference: Virtual Server → MCP Tool -->
      <g class="seq-step" data-seq-delay="2200">
        <line x1="560" y1="270" x2="790" y2="120" stroke="#876fd4" stroke-width="2" stroke-dasharray="6 4" class="draw-path" />
        <polygon points="786,115 800,118 790,128" fill="#876fd4" />
        <rect x="620" y="178" width="120" height="22" rx="4" fill="white" stroke="#876fd4" stroke-width="1" />
        <text x="680" y="193" text-anchor="middle" fill="#5e40be" font-family="Red Hat Mono, monospace" font-size="11">1..n ── 0..n</text>
      </g>
    </svg>
  </div>
  <p class="s4-caption seq-step" data-seq-delay="2600">All defined dynamically — no code changes needed.</p>
</div>
```

- [ ] **Step 3: Verify**

Navigate to slide 4. Entity boxes appear one by one with the containment arrow drawing after MCP Server and Tool are visible, then Virtual Server appears with its dashed reference arrow. Caption fades in last.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 4 - data model with animated UML diagram"
```

---

## Task 6: Slide 5 — MCP Tool Catalog

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 5: MCP Tool Catalog */
.s5 { max-width: 1100px; }
.s5-flow { display: grid; grid-template-columns: 1fr auto 1fr; gap: var(--space-xl); align-items: center; margin-top: var(--space-xl); }
.s5-pool { display: flex; flex-direction: column; gap: var(--space-md); }
.s5-server {
  background: white; border: 2px solid var(--border-diagram); border-radius: 8px;
  overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}
.s5-server-header {
  background: var(--bg-light-alt); padding: var(--space-sm) var(--space-md);
  font-family: var(--font-display); font-weight: 700; color: var(--text-dark-primary);
  border-bottom: 1px solid var(--border-diagram); font-size: var(--fs-body);
}
.s5-server-tools { padding: var(--space-sm) var(--space-md); display: flex; flex-wrap: wrap; gap: var(--space-xs); }
.s5-tool {
  font-family: var(--font-mono); font-size: 0.78rem; background: var(--orange-10);
  color: var(--orange-40); padding: 3px 10px; border-radius: 4px; border: 1px solid var(--orange-20);
}
.s5-arrow-col { display: flex; flex-direction: column; align-items: center; gap: var(--space-sm); color: var(--orange-40); }
.s5-arrow-col .arrow-text { font-size: var(--fs-small); font-weight: 700; letter-spacing: 0.05em; }
.s5-arrow-col .arrow-icon { font-size: 2.5rem; }
.s5-catalog {
  background: white; border: 3px solid var(--orange-40); border-radius: 12px;
  padding: var(--space-lg); box-shadow: 0 4px 24px rgba(245,146,27,0.15);
}
.s5-catalog-title {
  font-family: var(--font-display); font-weight: 700; color: var(--orange-40);
  font-size: var(--fs-h3); margin-bottom: var(--space-md);
  padding-bottom: var(--space-sm); border-bottom: 2px solid var(--orange-10);
}
.s5-caption { text-align: center; margin-top: var(--space-lg); color: var(--text-dark-muted); font-style: italic; }

@media (max-width: 768px) {
  .s5-flow { grid-template-columns: 1fr; }
  .s5-arrow-col .arrow-icon { transform: rotate(90deg); }
}
```

- [ ] **Step 2: Add Slide 5 content**

```html
<div class="slide-inner s5">
  <p class="slide-eyebrow anim-fade-up" style="color:var(--orange-40);">MCP Tool Catalog</p>
  <h2 class="h2 anim-fade-up">Curate what's available</h2>
  <div class="s5-flow">
    <!-- Pool of available servers -->
    <div class="s5-pool">
      <div class="s5-server seq-step" data-seq-delay="300">
        <div class="s5-server-header">Jira Server</div>
        <div class="s5-server-tools">
          <span class="s5-tool">get-story</span>
          <span class="s5-tool">add-watcher</span>
        </div>
      </div>
      <div class="s5-server seq-step" data-seq-delay="600">
        <div class="s5-server-header">GitHub Server</div>
        <div class="s5-server-tools">
          <span class="s5-tool">get-issue</span>
          <span class="s5-tool">create-PR</span>
        </div>
      </div>
    </div>

    <!-- Arrow -->
    <div class="s5-arrow-col seq-step" data-seq-delay="1000">
      <span class="arrow-text">Create Catalog</span>
      <span class="arrow-icon">&rarr;</span>
      <span class="arrow-text">Select Servers</span>
    </div>

    <!-- Catalog -->
    <div class="s5-catalog seq-step" data-seq-delay="1400">
      <div class="s5-catalog-title">Reporting Agent Catalog</div>
      <div class="s5-pool">
        <div class="s5-server" style="border-color:var(--orange-20);">
          <div class="s5-server-header" style="background:var(--orange-10);">Jira Server</div>
          <div class="s5-server-tools">
            <span class="s5-tool">get-story</span>
            <span class="s5-tool">add-watcher</span>
          </div>
        </div>
        <div class="s5-server" style="border-color:var(--orange-20);">
          <div class="s5-server-header" style="background:var(--orange-10);">GitHub Server</div>
          <div class="s5-server-tools">
            <span class="s5-tool">get-issue</span>
            <span class="s5-tool">create-PR</span>
          </div>
        </div>
      </div>
    </div>
  </div>
  <p class="s5-caption seq-step" data-seq-delay="1800">Your curated inventory of what's available in this deployment.</p>
</div>
```

- [ ] **Step 3: Verify**

Navigate to slide 5. Jira and GitHub server boxes appear on the left, arrow appears in center, then the catalog container with its contents appears on the right. Caption fades in last.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 5 - MCP tool catalog with curation flow"
```

---

## Task 7: Slide 6 — Virtual MCP Servers

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 6: Virtual MCP Servers */
.s6 { max-width: 1100px; }
.s6-diagram { margin-top: var(--space-xl); position: relative; }
.s6-catalog-strip {
  display: flex; justify-content: center; gap: var(--space-md); padding: var(--space-lg);
  background: var(--bg-light-alt); border: 2px solid var(--border-diagram); border-radius: 12px;
  margin-bottom: var(--space-2xl);
}
.s6-catalog-label {
  position: absolute; top: -12px; left: 50%; transform: translateX(-50%);
  background: white; padding: 0 var(--space-md);
  font-family: var(--font-display); font-weight: 700; color: var(--text-dark-muted);
  font-size: var(--fs-small); letter-spacing: 0.1em; text-transform: uppercase;
}
.s6-tool-chip {
  font-family: var(--font-mono); font-size: 0.82rem; padding: var(--space-sm) var(--space-md);
  border-radius: 6px; border: 2px solid var(--border-diagram); background: white;
  color: var(--text-dark-secondary); transition: border-color 400ms, background 400ms, color 400ms, box-shadow 400ms;
}
.s6-tool-chip.highlight-teal {
  border-color: var(--teal-50); background: var(--teal-10); color: var(--teal-70);
  box-shadow: 0 0 12px rgba(55,163,163,0.25);
}
.s6-tool-chip.highlight-purple {
  border-color: var(--purple-50); background: var(--purple-10); color: var(--purple-50);
  box-shadow: 0 0 12px rgba(94,64,190,0.25);
}
.s6-vs-row { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-xl); }
.s6-vs {
  border-radius: 12px; padding: var(--space-lg); text-align: center;
}
.s6-vs-teal { border: 3px solid var(--teal-50); background: rgba(218,242,242,0.3); }
.s6-vs-purple { border: 3px solid var(--purple-50); background: rgba(236,230,255,0.3); }
.s6-vs-title { font-family: var(--font-display); font-weight: 700; font-size: var(--fs-body); margin-bottom: var(--space-sm); }
.s6-vs-tools { font-family: var(--font-mono); font-size: 0.8rem; color: var(--text-dark-muted); }
.s6-vs-arrow { text-align: center; color: var(--text-dark-muted); font-size: 1.5rem; margin: var(--space-md) 0; }
.s6-caption { text-align: center; margin-top: var(--space-lg); color: var(--text-dark-muted); font-style: italic; }

@media (max-width: 768px) {
  .s6-vs-row { grid-template-columns: 1fr; }
  .s6-catalog-strip { flex-wrap: wrap; }
}
```

- [ ] **Step 2: Add Slide 6 content and JS highlight logic**

Slide content:

```html
<div class="slide-inner s6">
  <p class="slide-eyebrow anim-fade-up" style="color:var(--purple-50);">Virtual MCP Servers</p>
  <h2 class="h2 anim-fade-up">Compose purpose-built views</h2>
  <div class="s6-diagram">
    <!-- Catalog tool strip -->
    <div class="s6-catalog-strip seq-step" data-seq-delay="300" style="position:relative;">
      <span class="s6-catalog-label">Catalog Tools</span>
      <span class="s6-tool-chip" data-tool="get-story">get-story</span>
      <span class="s6-tool-chip" data-tool="add-watcher">add-watcher</span>
      <span class="s6-tool-chip" data-tool="get-issue">get-issue</span>
      <span class="s6-tool-chip" data-tool="create-pr">create-PR</span>
    </div>

    <div class="s6-vs-arrow seq-step" data-seq-delay="900">&darr;&ensp;select tools&ensp;&darr;</div>

    <!-- Virtual servers -->
    <div class="s6-vs-row">
      <div class="s6-vs s6-vs-teal seq-step" data-seq-delay="1300">
        <div class="s6-vs-title" style="color:var(--teal-70);">Read-Only Reporting</div>
        <div class="s6-vs-tools">get-issue &bull; get-story</div>
      </div>
      <div class="s6-vs s6-vs-purple seq-step" data-seq-delay="1700">
        <div class="s6-vs-title" style="color:var(--purple-50);">DevOps Actions</div>
        <div class="s6-vs-tools">create-PR &bull; add-watcher</div>
      </div>
    </div>
  </div>
  <p class="s6-caption seq-step" data-seq-delay="2100">Same catalog, different views, different consumers.</p>
</div>
```

Add tool highlight logic to the `fireSlideAnimations` function in `<script>`:

```javascript
// Inside fireSlideAnimations, after runSequentialSteps(cur):
if (parseInt(cur.getAttribute('data-slide'),10) === 6) {
  // Highlight tools matching each virtual server after VS appears
  setTimeout(function() {
    var chips = cur.querySelectorAll('.s6-tool-chip');
    chips.forEach(function(c) {
      var t = c.getAttribute('data-tool');
      if (t === 'get-story' || t === 'get-issue') c.classList.add('highlight-teal');
    });
  }, 1600);
  setTimeout(function() {
    var chips = cur.querySelectorAll('.s6-tool-chip');
    chips.forEach(function(c) {
      var t = c.getAttribute('data-tool');
      if (t === 'create-pr' || t === 'add-watcher') c.classList.add('highlight-purple');
    });
  }, 2000);
}
```

Also add cleanup when leaving the slide — in the `render()` function, after removing `.current` class from non-current slides, reset highlights:

```javascript
// In the forEach loop where slides lose .current:
// After s.classList.remove('current'):
Array.from(s.querySelectorAll('.s6-tool-chip')).forEach(function(c) {
  c.classList.remove('highlight-teal', 'highlight-purple');
});
Array.from(s.querySelectorAll('.seq-step')).forEach(function(step) {
  step.classList.remove('visible');
});
```

- [ ] **Step 3: Verify**

Navigate to slide 6. Catalog strip appears with 4 tool chips. Arrow appears. Virtual server boxes appear below. After each VS appears, the corresponding tools in the catalog strip highlight with matching colors (teal for read-only, purple for devops).

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 6 - virtual MCP servers with tool highlighting"
```

---

## Task 8: Slide 7 — Catalog Lifecycle

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 7: Catalog Lifecycle */
.s7 { max-width: 1000px; }
.s7-section-label { font-size: var(--fs-label); font-weight: 700; letter-spacing: 0.15em; text-transform: uppercase; color: var(--text-dark-muted); margin-bottom: var(--space-sm); margin-top: var(--space-xl); }
.s7-flow { display: flex; align-items: center; gap: 0; flex-wrap: wrap; }
.s7-state {
  padding: var(--space-sm) var(--space-lg); border-radius: 8px;
  font-family: var(--font-display); font-weight: 700; font-size: var(--fs-body);
  border: 2px solid var(--border-diagram); background: white; color: var(--text-dark-secondary);
  transition: background 400ms, border-color 400ms, color 400ms, box-shadow 400ms;
  white-space: nowrap;
}
.s7-state.active-teal {
  border-color: var(--teal-50); background: var(--teal-10); color: var(--teal-70);
  box-shadow: 0 0 16px rgba(55,163,163,0.3);
}
.s7-state.active-orange {
  border-color: var(--orange-40); background: var(--orange-10); color: #9e4a06;
  box-shadow: 0 0 16px rgba(245,146,27,0.3);
}
.s7-arrow { color: var(--text-dark-muted); font-size: 1.3rem; padding: 0 var(--space-sm); }
.s7-update-flow { margin-top: var(--space-lg); padding: var(--space-lg); background: var(--bg-light-alt); border-radius: 12px; border: 1px solid var(--border-diagram); }
.s7-update-label { font-size: var(--fs-small); color: var(--orange-40); font-weight: 700; letter-spacing: 0.1em; text-transform: uppercase; margin-bottom: var(--space-sm); }
.s7-note { font-size: var(--fs-small); color: var(--text-dark-muted); margin-top: var(--space-sm); font-style: italic; }
```

- [ ] **Step 2: Add Slide 7 content and sequential state lighting JS**

Slide content:

```html
<div class="slide-inner s7">
  <p class="slide-eyebrow anim-fade-up" style="color:var(--teal-50);">Lifecycle</p>
  <h2 class="h2 anim-fade-up">From draft to production</h2>

  <div class="s7-section-label seq-step" data-seq-delay="200">Promotion Flow</div>
  <div class="s7-flow seq-step" data-seq-delay="400">
    <span class="s7-state" data-lifecycle="draft">Draft</span>
    <span class="s7-arrow">&rarr;</span>
    <span class="s7-state" data-lifecycle="validate">Validate</span>
    <span class="s7-arrow">&rarr;</span>
    <span class="s7-state" data-lifecycle="testing">Testing</span>
    <span class="s7-arrow">&rarr;</span>
    <span class="s7-state" data-lifecycle="production">Production</span>
    <span class="s7-arrow">&rarr;</span>
    <span class="s7-state" data-lifecycle="publish">Publish</span>
  </div>

  <div class="s7-update-flow seq-step" data-seq-delay="2800">
    <div class="s7-update-label">Safe Update Path</div>
    <div class="s7-flow">
      <span class="s7-state" data-update="copy">Copy</span>
      <span class="s7-arrow">&rarr;</span>
      <span class="s7-state" data-update="edit">Edit</span>
      <span class="s7-arrow">&rarr;</span>
      <span class="s7-state" data-update="validate2">Validate</span>
      <span class="s7-arrow">&rarr;</span>
      <span class="s7-state" data-update="replace" style="border-width:3px;">Atomic Replace</span>
    </div>
    <p class="s7-note">Data version increments &bull; Old catalog archived for rollback</p>
  </div>
</div>
```

Add lifecycle state lighting to `fireSlideAnimations`:

```javascript
if (parseInt(cur.getAttribute('data-slide'),10) === 7) {
  var states = ['draft','validate','testing','production','publish'];
  states.forEach(function(name, i) {
    setTimeout(function() {
      var el = cur.querySelector('[data-lifecycle="' + name + '"]');
      if (el) el.classList.add('active-teal');
    }, 800 + i * 400);
  });
  var updates = ['copy','edit','validate2','replace'];
  updates.forEach(function(name, i) {
    setTimeout(function() {
      var el = cur.querySelector('[data-update="' + name + '"]');
      if (el) el.classList.add('active-orange');
    }, 3200 + i * 400);
  });
}
```

Add cleanup in `render()` for slide 7 states:

```javascript
// In the loop that removes .current from non-current slides:
Array.from(s.querySelectorAll('.s7-state')).forEach(function(st) {
  st.classList.remove('active-teal', 'active-orange');
});
```

- [ ] **Step 3: Verify**

Navigate to slide 7. Promotion flow appears, then states light up one by one in teal (Draft → Validate → Testing → Production → Publish). Then the update flow section appears below, and its states light up in orange.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 7 - catalog lifecycle with sequential state animations"
```

---

## Task 9: Slide 8 — Gateway Registration

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Add per-slide CSS**

```css
/* Slide 8: Gateway Registration */
.s8 { max-width: 1100px; }
.s8-arch { display: flex; align-items: flex-start; gap: 0; margin-top: var(--space-xl); justify-content: center; }
.s8-box {
  background: white; border: 2px solid var(--border-diagram); border-radius: 12px;
  padding: var(--space-lg); text-align: center; min-width: 180px;
  box-shadow: 0 2px 12px rgba(0,0,0,0.06);
}
.s8-box-title { font-family: var(--font-display); font-weight: 700; font-size: var(--fs-body); margin-bottom: var(--space-xs); }
.s8-box-sub { font-size: var(--fs-small); color: var(--text-dark-muted); }
.s8-box.s8-hub { border-color: var(--teal-50); border-top: 4px solid var(--teal-50); }
.s8-box.s8-hub .s8-box-title { color: var(--teal-70); }
.s8-box.s8-cr { border-color: var(--orange-40); border-top: 4px solid var(--orange-40); min-width: 240px; }
.s8-box.s8-cr .s8-box-title { color: #9e4a06; }
.s8-box.s8-gw { border-color: var(--purple-50); border-top: 4px solid var(--purple-50); }
.s8-box.s8-gw .s8-box-title { color: var(--purple-50); }
.s8-cr-fields { text-align: left; margin-top: var(--space-sm); font-family: var(--font-mono); font-size: 0.78rem; color: var(--text-dark-secondary); background: var(--bg-light-alt); padding: var(--space-sm) var(--space-md); border-radius: 4px; }
.s8-cr-fields div { padding: 2px 0; }
.s8-cr-fields .field-label { color: var(--text-dark-muted); }
.s8-cr-fields .field-value { color: var(--orange-40); font-weight: 500; }
.s8-arch-arrow {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  padding: 0 var(--space-md); color: var(--text-dark-muted); min-height: 100px;
}
.s8-arch-arrow .arrow-line { font-size: 1.8rem; }
.s8-arch-arrow .arrow-label { font-size: 0.72rem; letter-spacing: 0.08em; text-transform: uppercase; font-weight: 700; }
.s8-update-row {
  display: flex; align-items: center; justify-content: center; gap: var(--space-md);
  margin-top: var(--space-xl); padding: var(--space-lg);
  background: var(--bg-light-alt); border-radius: 12px; border: 1px solid var(--border-diagram);
}
.s8-update-step { font-family: var(--font-display); font-weight: 700; font-size: var(--fs-small); color: var(--text-dark-secondary); padding: var(--space-sm) var(--space-md); border-radius: 6px; border: 1px solid var(--border-diagram); background: white; }
.s8-update-arrow { color: var(--teal-50); font-size: 1.3rem; }
.s8-caption { text-align: center; margin-top: var(--space-lg); color: var(--text-dark-muted); font-style: italic; }

@media (max-width: 768px) {
  .s8-arch { flex-direction: column; align-items: center; gap: var(--space-md); }
  .s8-arch-arrow { flex-direction: row; padding: var(--space-sm) 0; min-height: auto; }
  .s8-arch-arrow .arrow-line { transform: rotate(90deg); }
  .s8-update-row { flex-direction: column; }
}
```

- [ ] **Step 2: Add Slide 8 content**

```html
<div class="slide-inner s8">
  <p class="slide-eyebrow anim-fade-up" style="color:var(--teal-70);">Gateway Registration</p>
  <h2 class="h2 anim-fade-up">Publish once. Discover everywhere.</h2>

  <div class="s8-arch">
    <!-- Asset Hub -->
    <div class="s8-box s8-hub seq-step" data-seq-delay="300">
      <div class="s8-box-title">Asset Hub</div>
      <div class="s8-box-sub">Publishes catalog</div>
    </div>

    <div class="s8-arch-arrow seq-step" data-seq-delay="700">
      <span class="arrow-label">creates</span>
      <span class="arrow-line">&rarr;</span>
    </div>

    <!-- Catalog CR -->
    <div class="s8-box s8-cr seq-step" data-seq-delay="1100">
      <div class="s8-box-title">Catalog CR</div>
      <div class="s8-cr-fields">
        <div><span class="field-label">endpoint:</span> <span class="field-value">/api/data/v1</span></div>
        <div><span class="field-label">catalog:</span> <span class="field-value">reporting-agent</span></div>
        <div><span class="field-label">dataVersion:</span> <span class="field-value">1</span></div>
      </div>
    </div>

    <div class="s8-arch-arrow seq-step" data-seq-delay="1500">
      <span class="arrow-label">watches</span>
      <span class="arrow-line">&rarr;</span>
    </div>

    <!-- Gateway -->
    <div class="s8-box s8-gw seq-step" data-seq-delay="1900">
      <div class="s8-box-title">MCP Gateway</div>
      <div class="s8-box-sub">Discovers &amp; routes</div>
    </div>
  </div>

  <!-- Update flow -->
  <div class="s8-update-row seq-step" data-seq-delay="2400">
    <span class="s8-update-step">Catalog updated</span>
    <span class="s8-update-arrow">&rarr;</span>
    <span class="s8-update-step" style="color:var(--orange-40);border-color:var(--orange-40);">dataVersion: 2</span>
    <span class="s8-update-arrow">&rarr;</span>
    <span class="s8-update-step">Gateway invalidates cache</span>
    <span class="s8-update-arrow">&rarr;</span>
    <span class="s8-update-step" style="color:var(--teal-50);border-color:var(--teal-50);">Re-fetches metadata</span>
  </div>

  <p class="s8-caption seq-step" data-seq-delay="2800">Zero-downtime catalog updates.</p>
</div>
```

- [ ] **Step 3: Verify**

Navigate to slide 8. Architecture builds left to right: Asset Hub → creates → Catalog CR (with fields) → watches → Gateway. Then the update flow row appears below showing the cache invalidation cycle.

- [ ] **Step 4: Commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: add slide 8 - gateway registration with architecture diagram"
```

---

## Task 10: Final Polish + Verification

**Files:**
- Modify: `docs/presentation/asset-hub-overview.html`

- [ ] **Step 1: Full walkthrough verification**

Open the file in a browser and navigate through all 9 slides sequentially. For each slide verify:

| Slide | Check |
|-------|-------|
| 1 | Title centered, Red Hat branding visible, stagger animation |
| 2 | Three gap cards animate in staggered, red accent visible |
| 3 | Two panels slide in from sides, arrow appears between |
| 4 | UML entities build up one by one, arrows draw, caption appears |
| 5 | Server pool → arrow → catalog container builds sequentially |
| 6 | Tool strip → virtual servers → tool chips highlight with colors |
| 7 | States light up sequentially (teal), then update flow lights (orange) |
| 8 | Architecture builds left→right, update flow appears below |
| 9 | Stats grid fades up, closing tagline visible |

- [ ] **Step 2: Test navigation edge cases**

- Press Home → should go to slide 1
- Press End → should go to slide 9
- Click dots randomly → should jump correctly
- Type "7" → should go to slide 7
- Prev on slide 1 → button should be disabled
- Next on slide 9 → button should be disabled
- Navigate away and back to an animated slide → animations should replay

- [ ] **Step 3: Test responsive layout**

Resize browser to 768px width. Verify:
- Font sizes reduce
- Diagrams stack vertically on mobile
- Controls bar stays usable
- No horizontal overflow

- [ ] **Step 4: Fix any issues found**

Apply fixes to any visual, animation, or navigation issues discovered in steps 1-3.

- [ ] **Step 5: Final commit**

```bash
git add docs/presentation/asset-hub-overview.html
git commit -m "feat: complete Asset Hub overview presentation - all 9 slides"
```
