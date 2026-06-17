#!/bin/bash
# Daily PostgreSQL backup — run via cron or manually
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="./backups"
mkdir -p $BACKUP_DIR
docker exec fintech-db pg_dump -U fintech_user -d fintech -F c -f /tmp/fintech_$DATE.dump
docker cp fintech-db:/tmp/fintech_$DATE.dump $BACKUP_DIR/
find $BACKUP_DIR -name "*.dump" -mtime +7 -delete
echo "Backup complete: $BACKUP_DIR/fintech_$DATE.dump"