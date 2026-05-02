package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UnitType maps to t_project_unit_types.
type UnitType struct {
	ID           string  `json:"id"`
	ProjectID    string  `json:"projectId"`
	TypeName     string  `json:"typeName"`
	Slug         string  `json:"slug"`
	LandSize     *string `json:"landSize,omitempty"`
	BuildingSize *string `json:"buildingSize,omitempty"`
	Bedroom      *int16  `json:"bedroom,omitempty"`
	Bathroom     *int16  `json:"bathroom,omitempty"`
	Garage       *int16  `json:"garage,omitempty"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
}

// GetUnitTypeByID returns a single unit type.
func GetUnitTypeByID(ctx context.Context, pool *pgxpool.Pool, id string) (*UnitType, error) {
	const q = `
		SELECT id, project_id, type_name, slug, land_size, building_size,
		       bedroom, bathroom, garage, price, stock
		FROM t_project_unit_types WHERE id=$1
	`
	var u UnitType
	err := pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.ProjectID, &u.TypeName, &u.Slug, &u.LandSize, &u.BuildingSize,
		&u.Bedroom, &u.Bathroom, &u.Garage, &u.Price, &u.Stock,
	)
	return &u, err
}

// GetUnitTypeBySlug returns a single unit type by its slug.
func GetUnitTypeBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*UnitType, error) {
	const q = `
		SELECT id, project_id, type_name, slug, land_size, building_size,
		       bedroom, bathroom, garage, price, stock
		FROM t_project_unit_types WHERE slug=$1
	`
	var u UnitType
	err := pool.QueryRow(ctx, q, slug).Scan(
		&u.ID, &u.ProjectID, &u.TypeName, &u.Slug, &u.LandSize, &u.BuildingSize,
		&u.Bedroom, &u.Bathroom, &u.Garage, &u.Price, &u.Stock,
	)
	return &u, err
}

// ListUnitTypes returns all unit types for a project.
func ListUnitTypes(ctx context.Context, pool *pgxpool.Pool, projectID string) ([]UnitType, error) {
	const q = `
		SELECT id, project_id, type_name, slug, land_size, building_size,
		       bedroom, bathroom, garage, price, stock
		FROM t_project_unit_types WHERE project_id=$1 ORDER BY created_at ASC
	`
	rows, err := pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UnitType
	for rows.Next() {
		var u UnitType
		if err := rows.Scan(
			&u.ID, &u.ProjectID, &u.TypeName, &u.Slug, &u.LandSize, &u.BuildingSize,
			&u.Bedroom, &u.Bathroom, &u.Garage, &u.Price, &u.Stock,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateUnitType inserts a new unit type for a project.
func CreateUnitType(
	ctx context.Context, pool *pgxpool.Pool,
	projectID, typeName, slug string,
	landSize, buildingSize *string,
	bedroom, bathroom, garage *int16,
	price float64, stock int,
) (*UnitType, error) {
	const q = `
		INSERT INTO t_project_unit_types
			(project_id, type_name, slug, land_size, building_size, bedroom, bathroom, garage, price, stock)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, project_id, type_name, slug, land_size, building_size, bedroom, bathroom, garage, price, stock
	`
	var u UnitType
	err := pool.QueryRow(ctx, q,
		projectID, typeName, slug, landSize, buildingSize,
		bedroom, bathroom, garage, price, stock,
	).Scan(
		&u.ID, &u.ProjectID, &u.TypeName, &u.Slug, &u.LandSize, &u.BuildingSize,
		&u.Bedroom, &u.Bathroom, &u.Garage, &u.Price, &u.Stock,
	)
	return &u, err
}

// UpdateUnitType updates a unit type, guarded by projectID.
func UpdateUnitType(
	ctx context.Context, pool *pgxpool.Pool,
	id, projectID, typeName, slug string,
	landSize, buildingSize *string,
	bedroom, bathroom, garage *int16,
	price float64, stock int,
) error {
	const q = `
		UPDATE t_project_unit_types
		SET type_name=$1, slug=$2, land_size=$3, building_size=$4,
		    bedroom=$5, bathroom=$6, garage=$7, price=$8, stock=$9, updated_at=now()
		WHERE id=$10 AND project_id=$11
	`
	cmd, err := pool.Exec(ctx, q,
		typeName, slug, landSize, buildingSize, bedroom, bathroom, garage,
		price, stock, id, projectID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("unit type not found or access denied")
	}
	return nil
}

// DeleteUnitType removes a unit type, guarded by projectID.
func DeleteUnitType(ctx context.Context, pool *pgxpool.Pool, id, projectID string) error {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM t_project_unit_types WHERE id=$1 AND project_id=$2`, id, projectID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("unit type not found or access denied")
	}
	return nil
}
