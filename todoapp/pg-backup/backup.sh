#!/bin/bash
set -e  # Exit on error

# Validate required environment variables
if [ -z "$DATABASE_URL" ]; then
    echo "ERROR: DATABASE_URL is not set"
    exit 1
fi

if [ -z "$BUCKET" ]; then
    echo "ERROR: BUCKET is not set"
    exit 1
fi

# Generate backup filename with timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="backup_${TIMESTAMP}.sql.gz"

echo "Starting backup at $(date)"
echo "Backup file: ${BACKUP_FILE}"

# Perform backup
pg_dump "$DATABASE_URL" | gzip > "/tmp/${BACKUP_FILE}"

# Upload to GCS
echo "Uploading to ${BUCKET}/${BACKUP_FILE}"
gsutil cp "/tmp/${BACKUP_FILE}" "${BUCKET}/"

# Cleanup local file
rm "/tmp/${BACKUP_FILE}"

echo "Backup completed successfully at $(date)"
exit 0