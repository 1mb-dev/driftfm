---
title: Design Language
---

Frontend design system reference. Token values live in `web/tokens.css` — this page covers the intent and patterns behind them.

For backend architecture, see [Architecture](architecture). For the philosophy, see [Building Drift FM](building-driftfm).

---

## Philosophy

- **Mood-driven.** UI adapts to the active mood — accent colors, backgrounds, breathing rhythms all shift.
- **Dark-first.** Dark default, light supported, auto follows OS preference.
- **Accessible.** Focus-visible, 44px touch targets, `prefers-reduced-motion`, ARIA live regions.
- **No frameworks.** Vanilla JS, CSS custom properties, ES6 modules. No build step.

---

## Color System

Semantic tokens in `:root`, mood accents remapped via `body[data-mood]`.

| Mood | Base | Dim |
|------|------|-----|
| Focus | `#3b82f6` (blue) | `#1e40af` |
| Calm | `#8b5cf6` (purple) | `#5b21b6` |
| Late Night | `#f59e0b` (amber) | `#b45309` |
| Energize | `#ef4444` (red) | `#b91c1c` |

Components reference `--color-accent` — never a specific mood color. Adding a mood means one new `body[data-mood]` rule.

---

## Typography

System font stack, no web fonts. Rem scale from `--text-xs` (0.75rem) to `--text-hero` (2.5rem). See `tokens.css` for the full scale.

---

## Theming

Three modes: `dark`, `light`, `auto`. Stored in localStorage, applied via `[data-theme]` on `<html>`.

A synchronous script (`init-theme.js`) reads the preference before first paint to prevent flash. If `auto`, resolves via `prefers-color-scheme`. Updates `<meta name="theme-color">` for browser chrome.

`[data-theme="light"]` overrides all semantic tokens — backgrounds, surfaces, glass, shadows.

---

## Glass Morphism

Translucent surfaces with backdrop blur. Three blur levels: subtle (8px), medium (16px), heavy (24px). Glass surfaces, scrims, and text-shadow tokens for legibility are all in `tokens.css`.

Fallback: browsers without `backdrop-filter` get solid `--color-surface` backgrounds.

---

## Motion

Duration tokens from `--duration-instant` (100ms) to `--duration-ambient` (4s). Three easing curves: `--ease-out`, `--ease-in`, `--ease-in-out`.

### Co-Prime Drift

Orb drift uses co-prime durations so cycles never visually repeat: 23s, 29s, 31s, 37s, 41s, 43s. Background gradients use a separate set: 7s, 11s, 13s, 17s, 19s — producing 17+ minutes of variation.

### Breathing

Mood orbs pulse at rates matched to energy: focus 4s, late night 5s, calm 6s, energize 3s.

### Reduced Motion

All animations disabled when `prefers-reduced-motion: reduce` is active. No loss of functionality.

---

## Responsive

Mobile-first. Three breakpoints.

| Breakpoint | Width | Key Change |
|------------|-------|------------|
| Base | < 641px | Two-row player bar (108px), mobile mood sizes |
| Tablet | 641px / 768px | Single-row player bar (72px), larger orbs |
| Desktop | 1024px | Full-size orbs, wider panels |

Safe area insets for notched devices on player bar and bottom panels.

---

## Z-Index Layers

| Token | Value | Element |
|-------|-------|---------|
| `--z-header` | 10 | Top nav |
| `--z-panel` | 50 | Side panels |
| `--z-drawer` | 60 | Bottom drawers |
| `--z-player` | 100 | Player bar (always on top) |
| `--z-toast` | 400 | Toast notifications |
| `--z-modal` | 500 | Modal overlays |

---

## Components

**Mood Orbs** — Round buttons in elliptical galaxy layout (JS-positioned, CSS-drifted). Gradient fill, breathing pulse, orbital drift. Hover pauses animation, shows glow + play icon. Selection expands 5x and fades, dimming other orbs. Falls back to CSS grid.

**Player Bar** — Fixed bottom, glass morphism. Track info, play/pause, skip, progress rail (keyboard-navigable). Mood pill taps back to selector.

**Drawers** — Bottom sheets for settings, about, lyrics. Slide up behind player bar. Glass background with scrim.

**Toasts** — Moodlet discovery and progressive skip friction. Auto-dismiss 10s. Above player bar.

**Lyrics Panel** — Plain text display when available. One line per lyric, blank lines between stanzas.

---

## Accessibility

- `:focus-visible` outlines on all interactive elements
- 44×44px minimum touch targets (`--touch-target`)
- `env(safe-area-inset-bottom)` for notched devices
- ARIA live regions for player state changes
- Keyboard navigation on progress rail and mood orbs
- `prefers-reduced-motion` disables all animation
- WCAG AA contrast in both themes
