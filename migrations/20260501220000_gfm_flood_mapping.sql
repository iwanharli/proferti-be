-- +goose Up
-- Aktifkan PostGIS & pgcrypto jika belum
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Tabel scene / produk GFM (Metadata Citra)
CREATE TABLE gfm_scene (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL DEFAULT 'copernicus_gfm',
    stac_item_id TEXT NOT NULL UNIQUE,
    acquisition_time TIMESTAMPTZ,
    product_time TIMESTAMPTZ,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    platform TEXT,
    orbit_direction TEXT,
    relative_orbit INTEGER,
    bbox geometry(POLYGON, 4326),
    footprint geometry(MULTIPOLYGON, 4326),
    raw_metadata JSONB
);
CREATE INDEX idx_gfm_scene_acquisition_time ON gfm_scene (acquisition_time);
CREATE INDEX idx_gfm_scene_footprint ON gfm_scene USING GIST (footprint);

-- 2. Tabel asset raster (File Layer)
CREATE TABLE gfm_asset (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES gfm_scene(id) ON DELETE CASCADE,
    band_name TEXT NOT NULL,
    asset_href TEXT NOT NULL,
    local_path TEXT,
    mime_type TEXT,
    file_size_bytes BIGINT,
    checksum TEXT,
    downloaded_at TIMESTAMPTZ,
    UNIQUE(scene_id, band_name)
);

-- 3. Tabel polygon banjir (Hasil Deteksi Spasial)
CREATE TABLE gfm_flood_polygon (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES gfm_scene(id) ON DELETE CASCADE,
    acquisition_time TIMESTAMPTZ NOT NULL,
    detected_date DATE GENERATED ALWAYS AS ((acquisition_time AT TIME ZONE 'UTC')::date) STORED,
    geom geometry(MULTIPOLYGON, 4326) NOT NULL,
    centroid geometry(POINT, 4326),
    area_m2 DOUBLE PRECISION,
    confidence_mean DOUBLE PRECISION,
    confidence_min DOUBLE PRECISION,
    confidence_max DOUBLE PRECISION,
    pixel_count INTEGER,
    admin_province_code TEXT,
    admin_city_code TEXT,
    admin_district_code TEXT,
    admin_village_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_gfm_flood_polygon_time ON gfm_flood_polygon (acquisition_time);
CREATE INDEX idx_gfm_flood_polygon_date ON gfm_flood_polygon (detected_date);
CREATE INDEX idx_gfm_flood_polygon_geom ON gfm_flood_polygon USING GIST (geom);
CREATE INDEX idx_gfm_flood_polygon_admin ON gfm_flood_polygon (admin_province_code, admin_city_code, admin_district_code, detected_date);

-- 4. Tabel agregasi harian per wilayah (Ringkasan Statistik)
CREATE TABLE gfm_admin_daily_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source TEXT NOT NULL DEFAULT 'copernicus_gfm',
    admin_level TEXT NOT NULL,
    admin_code TEXT NOT NULL,
    admin_name TEXT,
    date DATE NOT NULL,
    flood_polygon_count INTEGER NOT NULL DEFAULT 0,
    total_flood_area_m2 DOUBLE PRECISION NOT NULL DEFAULT 0,
    max_flood_area_m2 DOUBLE PRECISION NOT NULL DEFAULT 0,
    admin_area_m2 DOUBLE PRECISION,
    flood_percentage DOUBLE PRECISION,
    first_detected_at TIMESTAMPTZ,
    last_detected_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(source, admin_level, admin_code, date)
);

-- 5. Tabel Skor Risiko Historis (Analisis Jangka Panjang)
CREATE TABLE gfm_admin_risk_score (
    admin_level TEXT NOT NULL,
    admin_code TEXT NOT NULL,
    admin_name TEXT,
    total_detections INTEGER DEFAULT 0,
    flood_occurrence_count INTEGER DEFAULT 0,
    risk_score DOUBLE PRECISION DEFAULT 0,
    last_updated_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (admin_level, admin_code)
);

