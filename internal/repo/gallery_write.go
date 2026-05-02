package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AddGalleryImage inserts a gallery image URL for a project.
func AddGalleryImage(ctx context.Context, pool *pgxpool.Pool, projectID, imageURL string, title *string) (*ProjectGalleryRow, error) {
	const q = `
		INSERT INTO t_project_galleries (project_id, image, title)
		VALUES ($1, $2, $3)
		RETURNING id, image
	`
	var g ProjectGalleryRow
	err := pool.QueryRow(ctx, q, projectID, imageURL, title).Scan(&g.ID, &g.URL)
	return &g, err
}

// DeleteGalleryImage removes a gallery image, guarded by projectID.
func DeleteGalleryImage(ctx context.Context, pool *pgxpool.Pool, galleryID, projectID string) error {
	cmd, err := pool.Exec(ctx,
		`DELETE FROM t_project_galleries WHERE id=$1 AND project_id=$2`,
		galleryID, projectID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("image not found or access denied")
	}
	return nil
}
