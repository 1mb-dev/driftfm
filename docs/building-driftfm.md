---
title: Building Drift FM
---

The story behind Drift FM — why it exists, why it's built the way it is.

For the technical details, see [Architecture](architecture). For the frontend design system, see [Design Language](design).

---

## The Problem

Every music app wants your attention. Recommendations, social feeds, notifications — they optimize for engagement, not listening. I wanted something that plays music for a mood and gets out of the way.

Drift FM has one interaction: pick a mood. After that, music plays. No decisions, no feed, no algorithm nudging you toward something else.

---

## Why Self-Hosted

Your files, your server. No subscription, no licensing gaps, no "this track is no longer available." No analytics, no cookies, no third-party scripts.

It runs on a $5 VPS, a Raspberry Pi, or localhost. The binary is ~15 MB.

---

## The Stack

Go + SQLite + vanilla JS. Boring on purpose.

**Go** compiles to a single binary. Cross-compile to any platform. The stdlib HTTP server is more than enough for serving files to a handful of users.

**SQLite** because a few thousand tracks is a tiny dataset. WAL mode for concurrent reads. One file to back up. No Postgres, no connection pooling, no Docker Compose. We use [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — pure Go, no CGO, clean cross-compilation.

**Vanilla JS** because the player is one page with a few interactive elements. A framework would add a build pipeline and a package manager to solve problems this app doesn't have. CSS variables handle theming. The `<audio>` element handles playback.

---

## Shuffle with Memory

True randomness feels repetitive. Hearing the same track twice in an hour feels broken, not random.

The shuffle uses recency avoidance:

1. Partition tracks into "not recently played" and "recently played" (last 3)
2. Fisher-Yates shuffle only the non-recent tracks
3. Append recent tracks at the end, unshuffled
4. When a track plays, add it to the recent list (FIFO, capped at 3)

Simple, works well for small libraries. Stateful per mood — switching moods resets the window.

---

## The .txt Convention

Focus mood enforces instrumental-only. Rather than import flags, we use a file convention: if a `.txt` file exists next to an `.mp3` with the same name, the track is vocal. Content becomes displayable lyrics. Empty file still marks it as vocal.

Zero-config. Drop files, batch import, the script figures it out. See [Quickstart — Vocals and lyrics](quickstart#vocals-and-lyrics) for examples.

---

## Deploy

One binary, one database file, one directory of audio files. Copy three things, run it. See the [README](https://github.com/1mb-dev/driftfm#deploy) for the full deploy guide.
