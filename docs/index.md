---
title: Drift FM
---

Mood radio you host yourself.

Drop in your mp3s, tag them by mood, hit play. Continuous shuffled playback per mood. No accounts, no tracking, no frameworks.

`Go` `SQLite` `Vanilla JS`

---

## Quick Start

```bash
git clone https://github.com/1mb-dev/driftfm.git && cd driftfm
make db-init && make import-batch ARGS="/path/to/music"
make run
# → http://localhost:8080
```

---

## Documentation

[**Quickstart**](quickstart) — Step-by-step setup with your own music library. Prerequisites, import, configuration, troubleshooting.

[**Architecture**](architecture) — System internals. Backend packages, data model, frontend structure, request flow.

[**Building Drift FM**](building-driftfm) — Why these choices. The philosophy behind Go + SQLite + vanilla JS, shuffle with memory, and the .txt convention.

[**Design Language**](design) — Frontend design system. Color tokens, theming, glass morphism, motion, responsive patterns, accessibility.

---

## Links

- [GitHub](https://github.com/1mb-dev/driftfm)
- [Issues](https://github.com/1mb-dev/driftfm/issues)
- [Contributing](https://github.com/1mb-dev/driftfm/blob/main/.github/CONTRIBUTING.md)
- [License (MIT)](https://github.com/1mb-dev/driftfm/blob/main/LICENSE)
