#!/bin/bash

# Configuration
DB_NAME="db_proferti"
while true; do
    clear
    echo "===================================================="
    echo "🛰️  PROFERTI GFM WORKER MONITORING (Auto-refresh: 2s)"
    echo "===================================================="

    # 1. Check if ingest process is running
    INGEST_PID=$(pgrep -f "go run cmd/ingest/main.go" || pgrep -f "cmd/ingest/main")
    if [ -z "$INGEST_PID" ]; then
        echo "⭕ Worker Status: IDLE (No active ingestion process found)"
    else
        echo "🟢 Worker Status: RUNNING (PID: $INGEST_PID)"
    fi

    echo "----------------------------------------------------"
    echo "📊 DATABASE STATISTICS (GFM)"

    # 2. Query Statistics from Database
    psql -d $DB_NAME -c "
    SELECT 
        (SELECT COUNT(*) FROM gfm_scene) as total_scenes,
        (SELECT COUNT(*) FROM gfm_flood_polygon) as total_polygons,
        (SELECT MAX(acquisition_time) FROM gfm_scene) as latest_satellite_pass,
        (SELECT COUNT(*) FROM gfm_admin_daily_summary) as daily_summaries
    "

    echo "----------------------------------------------------"
    echo "🕒 RECENT ACTIVITY (Last 5 Ingested Scenes)"
    psql -d $DB_NAME -c "
    SELECT stac_item_id, platform, orbit_direction, acquisition_time, ingested_at 
    FROM gfm_scene 
    ORDER BY ingested_at DESC 
    LIMIT 5
    "

    echo "===================================================="
    echo "Tekan [CTRL+C] untuk berhenti."
    sleep 2
done
