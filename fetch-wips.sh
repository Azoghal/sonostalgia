#!/usr/bin/bash

set -e -o pipefail

for file in src/wip-memories/*; do
    [ -f "$file" ] || continue
    build/songfetcher $(xargs -a "$file")
done