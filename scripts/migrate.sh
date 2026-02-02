#!/bin/bash
#
# Run database migrations
# Usage: ./scripts/migrate.sh [migration_number]
#
# If migration_number is provided, runs that specific migration.
# Otherwise, runs all migrations that haven't been applied yet.

set -e

DB="data/inventory.db"
MIGRATIONS_DIR="scripts/migrations"

if [ ! -f "$DB" ]; then
    echo "Error: Database not found. Run 'make db-init' first."
    exit 1
fi

# Create migrations tracking table if not exists
sqlite3 "$DB" <<EOF
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
EOF

# If specific migration requested
if [ -n "$1" ]; then
    MIGRATION_FILE="$MIGRATIONS_DIR/$1"

    # Try with .sql extension if not provided
    if [ ! -f "$MIGRATION_FILE" ] && [ -f "${MIGRATION_FILE}.sql" ]; then
        MIGRATION_FILE="${MIGRATION_FILE}.sql"
    fi

    if [ ! -f "$MIGRATION_FILE" ]; then
        echo "Error: Migration file not found: $MIGRATION_FILE"
        exit 1
    fi

    VERSION=$(basename "$MIGRATION_FILE" .sql)

    # Check if already applied
    APPLIED=$(sqlite3 "$DB" "SELECT version FROM schema_migrations WHERE version='$VERSION';")
    if [ -n "$APPLIED" ]; then
        echo "Migration $VERSION already applied"
        exit 0
    fi

    echo "Applying migration: $VERSION"
    sqlite3 "$DB" < "$MIGRATION_FILE"
    sqlite3 "$DB" "INSERT INTO schema_migrations (version) VALUES ('$VERSION');"
    echo "Done: $VERSION"
    exit 0
fi

# Run all pending migrations
echo "Checking for pending migrations..."

for MIGRATION_FILE in "$MIGRATIONS_DIR"/*.sql; do
    [ -f "$MIGRATION_FILE" ] || continue

    VERSION=$(basename "$MIGRATION_FILE" .sql)

    # Check if already applied
    APPLIED=$(sqlite3 "$DB" "SELECT version FROM schema_migrations WHERE version='$VERSION';")
    if [ -n "$APPLIED" ]; then
        echo "  [SKIP] $VERSION (already applied)"
        continue
    fi

    echo "  [RUN]  $VERSION"
    sqlite3 "$DB" < "$MIGRATION_FILE"
    sqlite3 "$DB" "INSERT INTO schema_migrations (version) VALUES ('$VERSION');"
done

echo ""
echo "All migrations complete."

# Show current state
echo ""
echo "Applied migrations:"
sqlite3 "$DB" "SELECT version, applied_at FROM schema_migrations ORDER BY version;"
