#!/usr/bin/env bash
#
# Restores a School CBT Node's Postgres database from a backup made by
# scripts/backup-school-node.sh. DESTRUCTIVE: this overwrites the node's
# current database with the backup's contents.
#
# Usage:
#   scripts/restore-school-node.sh path/to/cbt-node-<db>-<timestamp>.sql.gz
#
# Requires an interactive "yes" confirmation unless CBT_RESTORE_CONFIRM=yes
# is set in the environment (for scripted/tested use - see this script's
# own test run recorded in README-SCHOOL-NODE.md).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

BACKUP_FILE="${1:?Usage: scripts/restore-school-node.sh path/to/backup.sql.gz}"
ENV_FILE=".env.school-node"
COMPOSE_FILE="docker-compose.school-node.yml"

if [ ! -f "$BACKUP_FILE" ]; then
  echo "ERROR: backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: $ENV_FILE not found. Run this from the School CBT Node's" >&2
  echo "deployment directory (the one containing $COMPOSE_FILE)." >&2
  exit 1
fi

# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a
DB_NAME="${DB_NAME:-cbt_db}"
DB_USER="${DB_USER:-cbt_node}"

if ! docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" ps postgres 2>/dev/null | grep -q "Up\|running"; then
  echo "ERROR: the postgres service isn't running. Start the node first:" >&2
  echo "  docker compose -f $COMPOSE_FILE --env-file $ENV_FILE up -d" >&2
  exit 1
fi

echo "================================================================"
echo " WARNING: this will REPLACE the current contents of database"
echo " '$DB_NAME' with the contents of:"
echo "   $BACKUP_FILE"
echo " Every exam, student record, and result created since that"
echo " backup was taken will be permanently lost."
echo "================================================================"

if [ "${CBT_RESTORE_CONFIRM:-}" != "yes" ]; then
  read -r -p "Type 'yes' to continue: " CONFIRM
  if [ "$CONFIRM" != "yes" ]; then
    echo "Aborted. No changes were made."
    exit 1
  fi
fi

echo "Stopping cbt-api so no request is served against a database being restored ..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" stop cbt-api

echo "Restoring $BACKUP_FILE into '$DB_NAME' ..."
# The backup was taken with pg_dump --clean --if-exists, so it drops and
# recreates every object itself - this restores cleanly over an existing,
# non-empty database without a separate manual wipe step.
if ! gunzip -c "$BACKUP_FILE" | docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
    psql -U "$DB_USER" -d "$DB_NAME" --set ON_ERROR_STOP=on; then
  echo "ERROR: restore failed partway through. The database may be in an" >&2
  echo "inconsistent state - restore again from a known-good backup before" >&2
  echo "restarting cbt-api." >&2
  exit 1
fi

echo "Restore complete. Starting cbt-api back up ..."
docker compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" start cbt-api

echo "Done."
