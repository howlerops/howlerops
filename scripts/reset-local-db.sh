#!/bin/bash

# Reset local SQLite databases (WARNING: This will delete all local data!)
set -e

# Configuration
DATA_DIR="${HOME}/.howlerops"
LOCAL_DB="${DATA_DIR}/local.db"
VECTORS_DB="${DATA_DIR}/vectors.db"

# Confirmation
read -p "This will DELETE all local data in ${DATA_DIR}. Continue? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted"
    exit 1
fi

# Backup existing databases
if [ -f "${LOCAL_DB}" ] || [ -f "${VECTORS_DB}" ]; then
    BACKUP_DIR="${DATA_DIR}/backups"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    mkdir -p "${BACKUP_DIR}"
    
    echo "Creating backup..."
    [ -f "${LOCAL_DB}" ] && cp "${LOCAL_DB}" "${BACKUP_DIR}/local_${TIMESTAMP}.db"
    [ -f "${VECTORS_DB}" ] && cp "${VECTORS_DB}" "${BACKUP_DIR}/vectors_${TIMESTAMP}.db"
    echo "Backup saved to: ${BACKUP_DIR}"
fi

# Remove databases
echo "Removing databases..."
rm -f "${LOCAL_DB}" "${LOCAL_DB}-shm" "${LOCAL_DB}-wal"
rm -f "${VECTORS_DB}" "${VECTORS_DB}-shm" "${VECTORS_DB}-wal"

# Reinitialize
echo "Reinitializing databases..."
bash scripts/init-local-db.sh

echo ""
echo "✓ Databases reset successfully!"
echo ""

