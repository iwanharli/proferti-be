package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)


type Unit struct {
	ID         string  `json:"id"`
	ProjectID  string  `json:"projectId"`
	UnitTypeID string  `json:"unitTypeId"`
	TypeName   string  `json:"typeName"`
	Block      *string `json:"block,omitempty"`
	Number     *string `json:"number,omitempty"`
	Facing     *string `json:"facing,omitempty"`
	Price      float64 `json:"price"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"createdAt"`
}

func CreateUnit(
	ctx context.Context, pool *pgxpool.Pool,
	projectID, unitTypeID string,
	block, number, facing *string,
	price float64, status string,
) (*Unit, error) {
	if status == "" {
		status = "available"
	}
	const q = `
		INSERT INTO t_project_units (project_id, unit_type_id, block, number, facing, price, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at::text
	`
	var u Unit
	err := pool.QueryRow(ctx, q, projectID, unitTypeID, block, number, facing, price, status).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	u.ProjectID = projectID
	u.UnitTypeID = unitTypeID
	u.Block = block
	u.Number = number
	u.Facing = facing
	u.Price = price
	u.Status = status
	return &u, nil
}

func ListUnitsByProject(ctx context.Context, pool *pgxpool.Pool, projectID string) ([]Unit, error) {
	const q = `
		SELECT u.id, u.project_id, u.unit_type_id, t.type_name, u.block, u.number, u.facing, u.price, u.status, u.created_at::text
		FROM t_project_units u
		INNER JOIN t_project_unit_types t ON u.unit_type_id = t.id
		WHERE u.project_id = $1
		ORDER BY t.type_name ASC, u.block ASC, u.number ASC
	`
	rows, err := pool.Query(ctx, q, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Unit
	for rows.Next() {
		var u Unit
		if err := rows.Scan(&u.ID, &u.ProjectID, &u.UnitTypeID, &u.TypeName, &u.Block, &u.Number, &u.Facing, &u.Price, &u.Status, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func UpdateUnitStatus(ctx context.Context, pool *pgxpool.Pool, id, status string) error {
	const q = `UPDATE t_project_units SET status = $1 WHERE id = $2`
	_, err := pool.Exec(ctx, q, status, id)
	return err
}
