#!/bin/bash

# Initialize local SQLite databases for HowlerOps
set -e

# Configuration
DATA_DIR="${HOME}/.howlerops"
LOCAL_DB="${DATA_DIR}/local.db"
VECTORS_DB="${DATA_DIR}/vectors.db"

# Create data directory
echo "Creating data directory: ${DATA_DIR}"
mkdir -p "${DATA_DIR}"
mkdir -p "${DATA_DIR}/extensions"
mkdir -p "${DATA_DIR}/backups"

# Initialize main local database
echo "Initializing local database: ${LOCAL_DB}"
if [ -f "${LOCAL_DB}" ]; then
    echo "Warning: ${LOCAL_DB} already exists. Skipping main DB initialization."
else
    sqlite3 "${LOCAL_DB}" < backend-go/pkg/storage/migrations/001_init_local_storage.sql
    echo "Local database initialized successfully"
fi

# Initialize vectors database
echo "Initializing vectors database: ${VECTORS_DB}"
if [ -f "${VECTORS_DB}" ]; then
    echo "Warning: ${VECTORS_DB} already exists. Skipping vectors DB initialization."
else
    sqlite3 "${VECTORS_DB}" < backend-go/internal/rag/migrations/001_init_sqlite_vectors.sql
    echo "Vectors database initialized successfully"
fi

# Set proper permissions
chmod 700 "${DATA_DIR}"
chmod 600 "${LOCAL_DB}" 2>/dev/null || true
chmod 600 "${VECTORS_DB}" 2>/dev/null || true

echo ""
echo "✓ Database initialization complete!"
echo "  Local DB: ${LOCAL_DB}"
echo "  Vectors DB: ${VECTORS_DB}"
echo ""
echo "Next steps:"
echo "  1. Run 'make dev' to start the development server"
echo "  2. Configure AI providers in the UI Settings"
echo ""

