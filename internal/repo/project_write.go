package repo

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateProject inserts a new project for a developer.
func CreateProject(
	ctx context.Context, pool *pgxpool.Pool,
	developerID, name, slug, locationID string,
	description, coverImage, promo *string,
	startPrice float64, status string,
) (*ProjectListRow, error) {
	if status == "" {
		status = "AVAILABLE"
	}
	const q = `
		INSERT INTO t_projects
			(developer_id, project_name, slug, location_id, description, cover_image, promo_text, status, starting_price, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8::"ProjectStatus",$9, now())
		RETURNING id, project_name, description, starting_price, promo_text, cover_image, status::text, created_at::text
	`
	var p ProjectListRow
	// Note: returning p.Location.ID etc might be tricky without a JOIN, 
	// but we usually just need the basic info back or fetch again.
	// For now we scan the basics.
	err := pool.QueryRow(ctx, q,
		developerID, name, slug, locationID,
		description, coverImage, promo, strings.ToUpper(status), startPrice,
	).Scan(
		&p.ID, &p.Name, &p.Description,
		&p.StartPrice, &p.Promo, &p.Image, &p.Status, &p.CreatedAt,
	)
	if err == nil {
		p.Location.ID = locationID
	}
	return &p, err
}

// UpdateProject updates a project, guarded by developerID ownership.
func UpdateProject(
	ctx context.Context, pool *pgxpool.Pool,
	projectID, developerID, name, slug, locationID string,
	description, coverImage, promo *string,
	startPrice float64, status string,
) error {
	const q = `
		UPDATE t_projects
		SET project_name=$1, slug=$2, location_id=$3, description=$4,
		    cover_image=$5, promo_text=$6, status=$7::"ProjectStatus",
		    starting_price=$8, updated_at=now()
		WHERE id=$9 AND developer_id=$10
	`
	cmd, err := pool.Exec(ctx, q,
		name, slug, locationID, description, coverImage, promo, strings.ToUpper(status),
		startPrice, projectID, developerID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("project not found or access denied")
	}
	return nil
}


// DeleteProject removes a project, guarded by developerID ownership.
func DeleteProject(ctx context.Context, pool *pgxpool.Pool, projectID, developerID string) error {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM t_projects WHERE id=$1 AND developer_id=$2`,
		projectID, developerID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("project not found or access denied")
	}
	return nil
}
