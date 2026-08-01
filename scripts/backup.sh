#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL is required}"
: "${BACKUP_BUCKET:?BACKUP_BUCKET is required}"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
target="${BACKUP_DIR:-./backups}/loreline-${stamp}.dump"
mkdir -p "$(dirname "$target")"
pg_dump --format=custom --no-owner --no-acl --dbname="$DATABASE_URL" --file="$target"
sha256sum "$target" > "${target}.sha256"
aws s3 cp "$target" "${BACKUP_BUCKET%/}/database/$(basename "$target")" --sse AES256
aws s3 cp "${target}.sha256" "${BACKUP_BUCKET%/}/database/$(basename "${target}.sha256")" --sse AES256
echo "$target"
