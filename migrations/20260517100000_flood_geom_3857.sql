-- +goose Up

-- Pre-project flood geometries to EPSG:3857 so MVT queries avoid per-row ST_Transform.
ALTER TABLE gfm_flood_polygon
    ADD COLUMN IF NOT EXISTS geom_3857 geometry(MultiPolygon, 3857);

UPDATE gfm_flood_polygon
    SET geom_3857 = ST_Transform(geom, 3857)
    WHERE geom_3857 IS NULL;

CREATE INDEX IF NOT EXISTS idx_gfm_flood_polygon_geom_3857
    ON gfm_flood_polygon USING GIST (geom_3857);

-- +goose Down
DROP INDEX IF EXISTS idx_gfm_flood_polygon_geom_3857;
ALTER TABLE gfm_flood_polygon DROP COLUMN IF EXISTS geom_3857;
