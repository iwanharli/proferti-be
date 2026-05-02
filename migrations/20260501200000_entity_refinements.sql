-- +goose Up
ALTER TABLE t_project_unit_types ADD COLUMN IF NOT EXISTS slug VARCHAR(255) NOT NULL;
ALTER TABLE t_projects ADD COLUMN IF NOT EXISTS project_type VARCHAR(50);

-- +goose Down
ALTER TABLE t_projects DROP COLUMN IF EXISTS project_type;
ALTER TABLE t_project_unit_types DROP COLUMN IF EXISTS slug;
