-- +goose Up
CREATE TABLE gfm_exclusion_polygon (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES gfm_scene(id) ON DELETE CASCADE,
    exclusion_type INTEGER NOT NULL, -- e.g., 1=Urban, 5=Radar Shadow
    geom geometry(MULTIPOLYGON, 4326) NOT NULL,
    area_m2 DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_gfm_exclusion_polygon_geom ON gfm_exclusion_polygon USING GIST (geom);
CREATE INDEX idx_gfm_exclusion_polygon_scene_id ON gfm_exclusion_polygon (scene_id);

-- +goose Down
DROP TABLE IF EXISTS gfm_exclusion_polygon;
