#!/usr/bin/env bash
set -euo pipefail
: "${DATABASE_URL:?DATABASE_URL is required}"
: "${LORELINE_CONFIRM_RESTORE:?Set LORELINE_CONFIRM_RESTORE=restore to acknowledge replacement of the target database}"
[[ "$LORELINE_CONFIRM_RESTORE" == "restore" ]] || { echo "restore confirmation did not match" >&2; exit 2; }
archive="${1:?usage: restore.sh BACKUP.dump}"
sha256sum --check "${archive}.sha256"
pg_restore --clean --if-exists --no-owner --no-acl --exit-on-error --dbname="$DATABASE_URL" "$archive"
echo "restore completed; run the smoke test before admitting traffic"
