#!/bin/sh
set -e

DB_PATH="${DB_PATH:-./bardbase.db}"
REPO="${GITHUB_REPO:-scottdkey/bardbase}"

if [ ! -f "$DB_PATH" ]; then
  echo "Downloading bardbase.db from latest release..."
  curl -fsSL \
    -H "Accept: application/octet-stream" \
    "https://github.com/$REPO/releases/latest/download/bardbase.db" \
    -o "$DB_PATH"
  echo "Downloaded $(du -h "$DB_PATH" | cut -f1)"
fi

exec ./api
