---
title: Design Language
---

Frontend design system reference for Drift FM contributors. This covers the token system, theming, motion, and component patterns used across the UI.

For the broader architecture (backend, data model, packages), see [Architecture](architecture). For the philosophy behind the tech choices, see [Building Drift FM](building-driftfm).

---

## Philosophy

- **Mood-driven.** The UI adapts to the active mood — accent colors, ambient backgrounds, and breathing rhythms all shift.
- **Dark-first.** The default theme is dark. Light theme is fully supported. Auto mode follows OS preference.
- **Accessible.** Focus-visible outlines, 44px touch targets, `prefers-reduced-motion` support, ARIA live regions.
- **No frameworks.** Vanilla JS, CSS custom properties, ES6 modules. No build step.

---

## Color System

Colors are defined as semantic tokens in `web/tokens.css`. Mood-specific accents are remapped dynamically via `body[data-mood]`.

### Semantic Tokens (Dark Default)

| Token | Value | Purpose |
|-------|-------|---------|
| `--color-bg` | `#0a0a0a` | Page background |
| `--color-surface` | `#141414` | Card/panel backgrounds |
| `--color-surface-elevated` | `#1a1a1a` | Elevated surfaces |
| `--color-text` | `#fafafa` | Primary text |
| `--color-text-muted` | `#888888` | Secondary text |
| `--color-text-subtle` | `#838383` | Tertiary text |
| `--color-border` | `#2a2a2a` | Borders and dividers |

### Mood Colors

| Mood | Base | Dim | Accent Use |
|------|------|-----|------------|
| Focus | `#3b82f6` (blue) | `#1e40af` | Default accent |
| Calm | `#8b5cf6` (purple) | `#5b21b6` | |
| Late Night | `#f59e0b` (amber) | `#b45309` | |
| Energize | `#ef4444` (red) | `#b91c1c` | |

### Dynamic Accent

The active mood sets `--color-accent` and `--color-accent-dim` via data attributes:

```css
body[data-mood="focus"]  { --color-accent: var(--color-focus); }
body[data-mood="calm"]   { --color-accent: var(--color-calm); }
```

All accent-aware components reference `--color-accent` — never a specific mood color. This means adding a new mood requires only a new `body[data-mood]` rule.

---

## Typography

System font stack. No web fonts, no loading delay.

```css
--font-sans: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
```

### Scale

| Token | Size | Usage |
|-------|------|-------|
| `--text-xs` | 0.75rem | Labels, metadata |
| `--text-sm` | 0.875rem | Secondary text, mood orb labels |
| `--text-base` | 1rem | Body text |
| `--text-lg` | 1.25rem | Tablet mood labels |
| `--text-xl` | 1.5rem | Section headings |
| `--text-2xl` | 2rem | Page headings |
| `--text-hero` | 2.5rem | Hero display text |

---

## Theming

Three modes: `dark`, `light`, `auto`. Stored in localStorage, applied via `[data-theme]` on `<html>`.

### How It Works

1. A synchronous external script (`init-theme.js`) loaded in `index.html` before other scripts reads the stored preference before first paint — prevents flash of wrong theme.
2. If `auto`, resolves via `window.matchMedia('(prefers-color-scheme: dark)')`.
3. Sets `document.documentElement.dataset.theme` to `'dark'` or `'light'`.
4. Updates `<meta name="theme-color">` for browser chrome.

### Light Theme Overrides

The `[data-theme="light"]` selector overrides semantic tokens:

| Token | Dark | Light |
|-------|------|-------|
| `--color-bg` | `#0a0a0a` | `#fafafa` |
| `--color-surface` | `#141414` | `#ffffff` |
| `--color-text` | `#fafafa` | `#1a1a1a` |
| `--surface-glass` | `rgba(10,10,10,0.8)` | `rgba(255,255,255,0.75)` |

Shadows and text-shadows also reduce in intensity for light theme.

---

## Glass Morphism

Layered translucent surfaces with backdrop blur. Used for the player bar, panels, drawers, and toasts.

### Blur Levels

| Token | Value | Usage |
|-------|-------|-------|
| `--blur-subtle` | 8px | Background hints |
| `--blur-medium` | 16px | Panels, drawers |
| `--blur-heavy` | 24px | Modal overlays |

### Glass Surfaces

```css
--surface-glass: rgba(10, 10, 10, 0.8);           /* Dark */
--surface-glass-elevated: rgba(26, 26, 26, 0.85);  /* Dark elevated */
--surface-scrim: rgba(0, 0, 0, 0.4);               /* Overlay dimming */
```

### Fallbacks

Browsers that don't support `backdrop-filter` get solid `--color-surface` backgrounds. The design is legible either way.

### Text Legibility

Text over glass surfaces uses text-shadow tokens:

```css
--text-shadow-soft: 0 1px 2px rgba(0, 0, 0, 0.3);    /* Default */
--text-shadow-strong: 0 2px 4px rgba(0, 0, 0, 0.5);   /* Over busy backgrounds */
```

---

## Motion

### Duration Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--duration-instant` | 100ms | Immediate feedback |
| `--duration-fast` | 150ms | Hover, focus transitions |
| `--duration-normal` | 300ms | Panel open/close |
| `--duration-slow` | 500ms | Page-level transitions |
| `--duration-ambient` | 4000ms | Breathing animation base |

### Easing

| Token | Curve | Usage |
|-------|-------|-------|
| `--ease-out` | `cubic-bezier(0, 0, 0.2, 1)` | Elements entering |
| `--ease-in` | `cubic-bezier(0.4, 0, 1, 1)` | Elements exiting |
| `--ease-in-out` | `cubic-bezier(0.4, 0, 0.2, 1)` | Continuous motion |

### Co-Prime Drift

Mood orb orbital drift uses co-prime durations so animation cycles never visually repeat:

```
Orb 1: 23s    Orb 2: 29s    Orb 3: 31s
Orb 4: 37s    Orb 5: 41s    Orb 6: 43s
```

Ambient background gradients use a separate co-prime set: 7s, 11s, 13s, 17s, 19s — producing a variation cycle over 17 minutes.

### Breathing

Mood orbs pulse with a subtle scale animation. Base period is `--duration-ambient` (4s). Energy level adjusts:

| Mood | Period | Rationale |
|------|--------|-----------|
| Focus | 4s | Default, neutral |
| Calm | 6s | Slower = calmer |
| Late Night | 5s | Between calm and default |
| Energize | 3s | Faster = more energy |

### Reduced Motion

All animations are disabled when `prefers-reduced-motion: reduce` is active. Mood orbs become static. Backgrounds don't animate. Expansion transitions are instant.

---

## Responsive

Mobile-first. Three breakpoints.

| Breakpoint | Width | Layout Changes |
|------------|-------|----------------|
| Base | < 641px | Two-row player bar, mobile mood sizes |
| Tablet | 641px / 768px | Single-row player bar, larger mood orbs |
| Desktop | 1024px | Full-size mood orbs, wider panels |

### Player Bar

The player bar is fixed to the bottom. On mobile, it uses a two-row layout (108px) — track info on top, controls below. At tablet width, it collapses to a single row (72px).

```css
--player-height: 108px;  /* Mobile */
/* Overridden to 72px at tablet breakpoint */
```

### Safe Areas

Notched devices are handled with `env(safe-area-inset-*)` on the player bar and bottom panels.

---

## Z-Index Layers

Explicit layering system. Panels slide up behind the player — the player stays on top for continuous control.

| Token | Value | Element |
|-------|-------|---------|
| `--z-header` | 10 | Top nav |
| `--z-panel` | 50 | Side panels |
| `--z-drawer` | 60 | Bottom drawers |
| `--z-player` | 100 | Player bar |
| `--z-toast` | 400 | Toast notifications |
| `--z-modal` | 500 | Modal overlays |

---

## Components

### Mood Orbs

Round buttons displayed in an elliptical galaxy layout (positioned by JS, drifted by CSS). Each orb:
- Gradient background from mood base to dim color
- Breathing pulse animation (scale 1 → 1.03)
- Orbital drift via `translate` property (GPU-accelerated)
- Hover: glow shadow, paused animation, scale lift, play icon reveal
- Selection: expansion animation (scale 5x, fade out), other orbs dim to 30%
- Fallback: CSS grid layout when galaxy positioning isn't available

### Player Bar

Fixed bottom bar with glass morphism background. Contains:
- Track info (title, artist, mood indicator pill)
- Playback controls (play/pause, skip)
- Progress rail with keyboard navigation (arrow keys, Home, End)
- Mood indicator pill: tap to return to mood selector

### Drawers

Bottom sheets for settings, about, and lyrics. Slide up from bottom, sit behind player bar in z-order. Glass morphism background, scrim overlay on the content behind.

### Toasts

Notification system for moodlet discovery (suggests intensity adjustments after several tracks) and progressive skip friction (visual nudge on 3rd skip, philosophy toast on 4th+ skip). Auto-dismiss after 10 seconds. Positioned above the player bar.

### Lyrics Panel

Displays lyrics when available for the current track. Plain text, one line per lyric line, blank lines between stanzas. Only visible when track has lyrics and user hasn't disabled the feature.

---

## Accessibility

- **Focus-visible:** All interactive elements have visible focus outlines via `:focus-visible`.
- **Touch targets:** Minimum 44×44px (`--touch-target`) for all tappable elements.
- **Safe area insets:** Player bar and bottom panels respect `env(safe-area-inset-bottom)` for notched devices.
- **ARIA live regions:** Player state changes (track title, mood) announced to screen readers.
- **Keyboard navigation:** Progress rail supports arrow keys, Home, End. Mood orbs are focusable buttons.
- **Reduced motion:** All animations disabled. No loss of functionality.
- **Color contrast:** Semantic text tokens maintain WCAG AA contrast in both themes.
