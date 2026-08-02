#!/bin/bash
# Repo bootstrap -- run once per clone/machine. Idempotent.
#
# This file is duplicated across repos on purpose: a fresh clone has to bootstrap itself
# with no sibling checkout present, so a shared copy would defeat the point.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

echo "==> Git hooks (core.hooksPath -> scripts/hooks)"
git config core.hooksPath scripts/hooks
chmod +x scripts/hooks/*
echo "    active: $(ls scripts/hooks | tr '\n' ' ')"

# Anything left in .git/hooks/ is now dead weight that still looks alive. Editing it is a
# silent no-op -- report it rather than let the next edit vanish into it.
shadowed=$(ls .git/hooks 2>/dev/null | grep -v '\.sample$' || true)
if [ -n "$shadowed" ]; then
    echo "    [warn] shadowed by hooksPath, git ignores these: $(echo "$shadowed" | tr '\n' ' ')"
    echo "           port any real checks into scripts/hooks/, then: rm .git/hooks/<name>"
fi

echo "==> Setup complete."
