#!/usr/bin/env bash
#
# Backs up a School CBT Node's Postgres database to a local, compressed
# dump file. Run manually, or on a schedule via cron (see
# README-SCHOOL-NODE.md's "Backups" section for the exact crontab line).
#
# This never requires internet access - it runs entirely against the
# node's own local `postgres` container via `docker compose exec`, the
# same way the node itself talks to its database.
#
# Usage:
#   scripts/backup-school-node.sh [backup-dir] [retention-count]
#
#   backup-dir        Where dumps are written. Default: ./backups
#   retention-count   How many recent backups to keep (older ones are
#                     deleted). Default: 14. Pass 0 to keep all.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

BACKUP_DIR="${1:-./backups}"
RETENTION="${2:-14}"
ENV_FILE=".env.school-node"
COMPOSE_FILE="docker-compose.school-node.yml"

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: $ENV_FILE not found. Run this from the School CBT Node's" >&2
  echo "deployment directory (the one containing $COMPOSE_FILE)." >&2
  exit 1
fi

# Read DB_NAME/DB_USER the same way docker-compose.school-node.yml does
# (with the same defaults), so this script never needs its own copy of
# those values to drift out of sync with the actual running container.
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a
DB_NAME="${DB_NAME:-cbt_db}"
DB_USER="${DB_USER:-cbt_node}"

if ! docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps postgres 2>/dev/null | grep -q "Up\|running"; then
  echo "ERROR: the postgres service isn't running. Start the node first:" >&2
  echo "  docker compose -f $COMPOSE_FILE --env-file $ENV_FILE up -d" >&2
  exit 1
fi

mkdir -p "$BACKUP_DIR"
# Backups contain every student's and teacher's data for this school -
# restrict access immediately, before any data is written, not after.
chmod 700 "$BACKUP_DIR"

TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DUMP_FILE="$BACKUP_DIR/cbt-node-${DB_NAME}-${TIMESTAMP}.sql.gz"
TMP_FILE="${DUMP_FILE}.partial"

echo "Backing up database '$DB_NAME' to $DUMP_FILE ..."

# --clean --if-exists: the restore script below runs this dump against a
# freshly-started empty database normally, but making the dump itself
# self-contained (drops existing objects before recreating them) means
# it's also safe to restore over a non-empty database without a separate
# manual cleanup step first.
if ! docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
    pg_dump -U "$DB_USER" -d "$DB_NAME" --clean --if-exists \
    | gzip > "$TMP_FILE"; then
  echo "ERROR: pg_dump failed. No backup file was left in place." >&2
  rm -f "$TMP_FILE"
  exit 1
fi

# Only now, after pg_dump AND gzip both succeeded, does the backup
# become visible under its real name - an interrupted or failed backup
# must never be mistaken for a complete, restorable one.
mv "$TMP_FILE" "$DUMP_FILE"
chmod 600 "$DUMP_FILE"

SIZE="$(du -h "$DUMP_FILE" | cut -f1)"
echo "Backup complete: $DUMP_FILE ($SIZE)"

if [ "$RETENTION" != "0" ]; then
  # Filenames are script-generated (timestamp, no spaces), so plain
  # newline-separated `ls -t` (newest first by mtime) is safe here.
  COUNT=0
  while IFS= read -r old; do
    COUNT=$((COUNT + 1))
    if [ "$COUNT" -gt "$RETENTION" ]; then
      echo "Removing old backup (retention=$RETENTION): $old"
      rm -f "$old"
    fi
  done < <(ls -t "$BACKUP_DIR"/cbt-node-*.sql.gz 2>/dev/null)
fi

echo "Done. $(find "$BACKUP_DIR" -maxdepth 1 -name 'cbt-node-*.sql.gz' | wc -l) backup(s) retained in $BACKUP_DIR."
