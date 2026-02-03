---
title: Building Drift FM
---

# Building Drift FM

The opinions, trade-offs, and philosophy behind how Drift FM is built. For the technical architecture (packages, data model, request flow), see [Architecture](architecture).

---

## The Problem

Most music apps optimize for engagement. Algorithmic recommendations, social features, infinite scroll, notifications pulling you back in. The goal is retention, not listening.

Drift FM optimizes for mood. You pick a feeling — focus, calm, energize, late night — and the music plays. No recommendations, no feed, no decisions after the first one. You drift.

---

## Why Self-Hosted

Your music should live on your hardware. Not behind a subscription. Not gated by a service that might change its terms, raise its price, or shut down.

Self-hosting means:
- **Your library, your rules.** No licensing gaps. No region locks. No "this track is no longer available."
- **No tracking.** Zero analytics, zero cookies, zero third-party scripts. Listen events are stored locally in SQLite for playlist optimization only.
- **Runs anywhere.** A $5 VPS, a Raspberry Pi, your laptop. The binary is ~15 MB. Deploy it next to your files and forget about it.

---

## Why Go + SQLite + Vanilla JS

The stack is deliberately boring.

### Go

Go compiles to a single binary. No runtime, no JVM, no dependency tree. Cross-compile to any platform with one command. The standard library includes a production-grade HTTP server — no framework needed.

For a music server that handles a handful of concurrent users serving files from disk, Go is wildly overqualified. That's the point. The server will never be the bottleneck.

### SQLite

A music library of a few thousand tracks is a small dataset. SQLite handles it trivially. WAL mode gives concurrent reads. The database is a single file you can copy, back up, or inspect with the SQLite CLI.

Drift FM uses [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — a pure Go SQLite implementation. No CGO, no C compiler, clean cross-compilation. It's slightly slower than the C driver, but "slightly slower" on a workload this small is immeasurable.

No Postgres means no connection pooling, no migrations server, no Docker Compose for development. `make db-init` and you're done.

### Vanilla JS

The frontend is vanilla JavaScript using ES6 modules — the core player logic in `app.js` is about 1000 lines. No React, no Vue, no build step, no node_modules, no bundler.

This isn't a dogmatic stance. It's a scope decision. The player has one page, a few interactive elements, and a straightforward state model. A framework would add a build pipeline, a package manager, and a layer of abstraction — all to solve problems this app doesn't have.

CSS variables handle theming. The `<audio>` element handles playback. The browser is the framework.

---

## Shuffle with Memory

Random shuffle has a problem: true randomness feels repetitive. In a library of 20 tracks, hearing the same song twice in an hour doesn't feel random — it feels broken.

Drift FM's shuffle uses **recency avoidance**:

1. Fetch all tracks for the mood from SQLite
2. Partition into "not recently played" and "recently played" (last 3 track IDs)
3. Fisher-Yates shuffle only the non-recent tracks
4. Rebuild: shuffled non-recent first, recent appended at the end (unshuffled)
5. When a track plays, its ID is added to the recent list (FIFO, capped at 3)

This is simple and works well for small libraries. You won't hear a recently played track again until at least 3 others have played. With larger libraries the recency window is barely noticeable — but it still prevents the jarring back-to-back repeat.

The algorithm is stateful per mood. Switching moods resets the recency window. This is intentional: if you switch to calm and back to focus, you might hear a recently played focus track — but the mood change creates enough perceptual distance that it doesn't feel like a repeat.

---

## The .txt Convention

Drift FM needs to know if a track has vocals. Focus mood enforces instrumental-only — a vocal track in a focus playlist breaks concentration.

Rather than adding flags to the import command, Drift FM uses a file convention: if a `.txt` file with the same base name exists next to an `.mp3`, the track is marked as vocal. If the `.txt` has content, it's imported as displayable lyrics. If it's empty, the track is still vocal — just without lyrics.

See [Quickstart — Vocals and lyrics](quickstart#vocals-and-lyrics) for the full convention and examples.

This convention is zero-config. You don't need to remember import flags. Drop your files in a directory, add `.txt` files for vocal tracks, and batch import. The script figures it out.

---

## Single Binary, Deploy Anywhere

`make build` produces one binary. Copy it to a server along with the `web/` directory, your `config.yaml`, and your audio files. Run it.

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/server ./cmd/server
```

No container orchestration required. No process manager required (though systemd is sensible for production). No reverse proxy required (though Caddy or nginx in front for TLS is recommended).

The goal is the smallest operational footprint that still feels complete. One binary, one database file, one directory of audio files. Back it up by copying three things.

---

## What This Isn't

Drift FM is not a music discovery service. It doesn't fetch album art, it doesn't look up metadata from external APIs, it doesn't suggest tracks. It plays what you give it, in the mood you choose, with shuffle that respects your recent listening.

It's a player, not a platform. The value is in the simplicity: mood selection, continuous playback, and getting out of the way.
