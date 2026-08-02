#!/usr/bin/env bash
set -euo pipefail
cd /srv/sonostalgia

git add src/memories src/wip-memories src/assets

if git diff --cached --quiet; then
  exit 0
fi

git commit -m "auto: sync content $(date -u +%Y-%m-%dT%H:%M:%SZ)"
git push origin HEAD:vps-sync
