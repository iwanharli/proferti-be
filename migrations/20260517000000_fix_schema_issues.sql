-- +goose Up

-- Fix 1: password nullable — OAuth users tidak punya password
ALTER TABLE t_users ALTER COLUMN password DROP NOT NULL;

-- Fix 2: slug unit type default kosong — aman untuk baris yang sudah ada
ALTER TABLE t_project_unit_types ALTER COLUMN slug SET DEFAULT '';

-- +goose Down
ALTER TABLE t_project_unit_types ALTER COLUMN slug DROP DEFAULT;
ALTER TABLE t_users ALTER COLUMN password SET NOT NULL;
