-- +goose Up

-- 1. Create t_project_locations (Universal Location Metadata)
CREATE TABLE IF NOT EXISTS t_project_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    address TEXT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    region_id INTEGER REFERENCES regions(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 2. Link t_projects to t_project_locations
ALTER TABLE t_projects ADD COLUMN location_id UUID REFERENCES t_project_locations(id) ON DELETE SET NULL;
ALTER TABLE t_projects ADD COLUMN polygon_coordinates JSONB;

-- 3. Link t_developers to regions
ALTER TABLE t_developers ADD COLUMN region_id INTEGER REFERENCES regions(id) ON DELETE SET NULL;

-- 4. Link t_leads to regions
ALTER TABLE t_leads ADD COLUMN region_id INTEGER REFERENCES regions(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE t_leads DROP COLUMN IF EXISTS region_id;
ALTER TABLE t_developers DROP COLUMN IF EXISTS region_id;
ALTER TABLE t_projects DROP COLUMN IF EXISTS polygon_coordinates;
ALTER TABLE t_projects DROP COLUMN IF EXISTS location_id;
DROP TABLE IF EXISTS t_project_locations;
