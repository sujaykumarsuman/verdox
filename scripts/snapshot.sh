#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# Verdox Dev Snapshot — create and restore database snapshots
# ============================================================
#
# Usage:
#   scripts/snapshot.sh create [tag]       Create a snapshot (tag defaults to timestamp)
#   scripts/snapshot.sh restore <tag>      Restore a snapshot into the running dev cluster
#   scripts/snapshot.sh list               List available snapshots
#
# Snapshots are stored in snapshots/<tag>/ and contain:
#   - dump.pgdump   PostgreSQL custom-format dump
#   - metadata.json  Git branch, commit, timestamp, migration version

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SNAPSHOTS_DIR="$PROJECT_ROOT/snapshots"

COMPOSE="docker compose --env-file .env.dev -f docker-compose.yml -f docker-compose.dev.yml"
DB_USER="verdox"
DB_NAME="verdox"

# ── Helpers ──────────────────────────────────────────────────

red()   { printf '\033[0;31m%s\033[0m\n' "$*"; }
green() { printf '\033[0;32m%s\033[0m\n' "$*"; }
dim()   { printf '\033[0;90m%s\033[0m\n' "$*"; }

die() { red "Error: $*" >&2; exit 1; }

check_postgres() {
  cd "$PROJECT_ROOT"
  local status
  status=$($COMPOSE ps postgres --format '{{.Health}}' 2>/dev/null || echo "")
  if [[ "$status" != *"healthy"* ]]; then
    return 1
  fi
  return 0
}

pg_exec() {
  cd "$PROJECT_ROOT"
  $COMPOSE exec -T postgres "$@"
}

# ── Create ───────────────────────────────────────────────────

cmd_create() {
  local tag="${1:-$(date +%Y%m%d-%H%M%S)}"

  # Validate tag (alphanumeric, hyphens, underscores)
  if [[ ! "$tag" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    die "Invalid tag '$tag'. Use alphanumeric characters, hyphens, underscores, or dots."
  fi

  local snap_dir="$SNAPSHOTS_DIR/$tag"
  if [[ -d "$snap_dir" ]]; then
    die "Snapshot '$tag' already exists. Choose a different tag or delete snapshots/$tag/"
  fi

  check_postgres || die "PostgreSQL is not running. Start the dev cluster with 'make dev' first."

  mkdir -p "$snap_dir"

  # Dump database
  echo "Dumping database..."
  pg_exec pg_dump -U "$DB_USER" -Fc "$DB_NAME" > "$snap_dir/dump.pgdump"

  local dump_size
  dump_size=$(du -sh "$snap_dir/dump.pgdump" | cut -f1)

  # Get migration version
  local migration_version
  migration_version=$(pg_exec psql -U "$DB_USER" -d "$DB_NAME" -tAc \
    "SELECT COALESCE(MAX(version), 0) FROM schema_migrations" 2>/dev/null || echo "unknown")

  # Get git info
  local git_branch git_commit
  git_branch=$(git -C "$PROJECT_ROOT" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
  git_commit=$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || echo "unknown")

  # Write metadata
  cat > "$snap_dir/metadata.json" <<EOF
{
  "tag": "$tag",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "git_branch": "$git_branch",
  "git_commit": "$git_commit",
  "migration_version": $migration_version,
  "dump_size": "$dump_size"
}
EOF

  echo ""
  green "Snapshot created: snapshots/$tag/"
  dim "  Dump size:  $dump_size"
  dim "  Migration:  v$migration_version"
  dim "  Branch:     $git_branch ($git_commit)"
}

# ── Restore ──────────────────────────────────────────────────

cmd_restore() {
  local tag="${1:-}"
  if [[ -z "$tag" ]]; then
    die "Usage: scripts/snapshot.sh restore <tag>\n       Run 'scripts/snapshot.sh list' to see available snapshots."
  fi

  local snap_dir="$SNAPSHOTS_DIR/$tag"
  local dump_file="$snap_dir/dump.pgdump"

  [[ -d "$snap_dir" ]]  || die "Snapshot '$tag' not found. Run 'scripts/snapshot.sh list' to see available snapshots."
  [[ -f "$dump_file" ]] || die "Dump file missing in snapshots/$tag/. Snapshot may be corrupted."

  check_postgres || die "Dev cluster is not running. Start with 'make dev' first."

  echo "Restoring snapshot '$tag'..."

  # Terminate existing connections
  dim "  Terminating active connections..."
  pg_exec psql -U "$DB_USER" -d postgres -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='$DB_NAME' AND pid <> pg_backend_pid();" \
    > /dev/null 2>&1 || true

  # Drop and recreate database
  dim "  Dropping and recreating database..."
  pg_exec psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" > /dev/null
  pg_exec psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" > /dev/null

  # Restore dump
  dim "  Restoring dump..."
  cat "$dump_file" | pg_exec pg_restore -U "$DB_USER" -d "$DB_NAME" --no-owner --no-privileges 2>/dev/null || true

  # Flush Redis (stale sessions, job queues)
  dim "  Flushing Redis..."
  cd "$PROJECT_ROOT"
  $COMPOSE exec -T redis redis-cli FLUSHALL > /dev/null 2>&1 || true

  # Show what was restored
  if [[ -f "$snap_dir/metadata.json" ]]; then
    local branch commit migration
    branch=$(grep -o '"git_branch": *"[^"]*"' "$snap_dir/metadata.json" | cut -d'"' -f4)
    commit=$(grep -o '"git_commit": *"[^"]*"' "$snap_dir/metadata.json" | cut -d'"' -f4)
    migration=$(grep -o '"migration_version": *[0-9]*' "$snap_dir/metadata.json" | grep -o '[0-9]*$')
    echo ""
    green "Snapshot '$tag' restored successfully."
    dim "  Source:     $branch ($commit)"
    dim "  Migration:  v$migration"
  else
    echo ""
    green "Snapshot '$tag' restored successfully."
  fi

  dim ""
  dim "  Backend will auto-reconnect. If it doesn't, restart with: make dev"
}

# ── List ─────────────────────────────────────────────────────

cmd_list() {
  if [[ ! -d "$SNAPSHOTS_DIR" ]] || [[ -z "$(ls -A "$SNAPSHOTS_DIR" 2>/dev/null)" ]]; then
    dim "No snapshots found. Create one with: make snapshot TAG=my-tag"
    return
  fi

  printf "%-24s %-22s %-20s %-8s %s\n" "TAG" "TIMESTAMP" "BRANCH" "MIGRATE" "SIZE"
  printf "%-24s %-22s %-20s %-8s %s\n" "---" "---------" "------" "-------" "----"

  for meta in "$SNAPSHOTS_DIR"/*/metadata.json; do
    [[ -f "$meta" ]] || continue
    local dir
    dir=$(dirname "$meta")
    local tag timestamp branch migration size
    tag=$(grep -o '"tag": *"[^"]*"' "$meta" | cut -d'"' -f4)
    timestamp=$(grep -o '"timestamp": *"[^"]*"' "$meta" | cut -d'"' -f4)
    branch=$(grep -o '"git_branch": *"[^"]*"' "$meta" | cut -d'"' -f4)
    migration=$(grep -o '"migration_version": *[0-9]*' "$meta" | grep -o '[0-9]*$')
    size=$(grep -o '"dump_size": *"[^"]*"' "$meta" | cut -d'"' -f4)
    printf "%-24s %-22s %-20s v%-7s %s\n" "$tag" "$timestamp" "$branch" "$migration" "$size"
  done
}

# ── Main ─────────────────────────────────────────────────────

case "${1:-}" in
  create)  shift; cmd_create "$@" ;;
  restore) shift; cmd_restore "$@" ;;
  list)    cmd_list ;;
  *)
    echo "Verdox Dev Snapshot"
    echo ""
    echo "Usage:"
    echo "  scripts/snapshot.sh create [tag]    Create a snapshot"
    echo "  scripts/snapshot.sh restore <tag>   Restore a snapshot"
    echo "  scripts/snapshot.sh list            List available snapshots"
    exit 1
    ;;
esac
