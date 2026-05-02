#!/bin/bash

echo "===================================================="
echo "🔍 PROFERTI SYSTEM HEALTH CHECK"
echo "===================================================="

# 1. Check .env file
if [ ! -f ".env" ]; then
    echo "❌ ERROR: .env file not found!"
    exit 1
else
    echo "✅ .env file found."
fi

# Load .env safely
set -a
[ -f .env ] && . .env
set +a

STATUS=0

# 2. Check Database Connection
echo "📡 Checking Database Connection..."
if psql $DATABASE_URL -c "SELECT 1" > /dev/null 2>&1; then
    echo "✅ Database connection successful."
else
    echo "❌ ERROR: Cannot connect to database. Check DATABASE_URL in .env"
    STATUS=1
fi

# 3. Check GDAL Dependencies
echo "🗺️  Checking GDAL Dependencies..."

# Check GDAL_BIN_PATH from .env or default
GDAL_PATH=${GDAL_BIN_PATH:-"/usr/local/bin"}

if [ -f "$GDAL_PATH/gdalwarp" ]; then
    echo "✅ gdalwarp found at $GDAL_PATH"
elif command -v gdalwarp > /dev/null 2>&1; then
    echo "✅ gdalwarp found in system PATH."
else
    echo "❌ ERROR: gdalwarp not found! Please install GDAL or set GDAL_BIN_PATH."
    STATUS=1
fi

if [ -f "$GDAL_PATH/raster2pgsql" ]; then
    echo "✅ raster2pgsql found at $GDAL_PATH"
elif command -v raster2pgsql > /dev/null 2>&1; then
    echo "✅ raster2pgsql found in system PATH."
else
    echo "❌ ERROR: raster2pgsql not found! Usually comes with PostGIS client tools."
    STATUS=1
fi

# 4. Check Data Directory
echo "📁 Checking Data Directory..."
DATA_DIR="./data/gfm"
mkdir -p $DATA_DIR
if [ -w "$DATA_DIR" ]; then
    echo "✅ Data directory $DATA_DIR is writable."
else
    echo "❌ ERROR: Data directory $DATA_DIR is NOT writable!"
    STATUS=1
fi

echo "===================================================="
if [ $STATUS -eq 0 ]; then
    echo "🚀 Everything looks good! You are ready to ingest."
else
    echo "⚠️  SOME CHECKS FAILED. Please fix the errors above before running ingest."
fi
echo "===================================================="

