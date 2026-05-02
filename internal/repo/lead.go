package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Lead maps to t_leads with a joined project name.
type Lead struct {
	ID          string   `json:"id"`
	DeveloperID string   `json:"developerId"`
	ProjectID   string   `json:"projectId"`
	ProjectName string   `json:"projectName"`
	Name        string   `json:"name"`
	Phone       *string  `json:"phone,omitempty"`
	Email       *string  `json:"email,omitempty"`
	Budget      *float64 `json:"budget,omitempty"`
	Message     *string  `json:"message,omitempty"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"createdAt"`
}

// CreateLead inserts a lead, resolving developer_id from the project.
func CreateLead(
	ctx context.Context, pool *pgxpool.Pool,
	projectID, name string,
	phone, email, message *string,
	budget *float64,
) (*Lead, error) {
	// INSERT ... SELECT so developer_id is pulled from t_projects automatically.
	const q = `
		INSERT INTO t_leads (developer_id, project_id, name, phone, email, message, budget, status)
		SELECT developer_id, id, $2, $3, $4, $5, $6, 'new'
		FROM t_projects WHERE id = $1
		RETURNING id, developer_id, project_id, name, phone, email, budget, message, status, created_at::text
	`
	var l Lead
	err := pool.QueryRow(ctx, q, projectID, name, phone, email, message, budget).Scan(
		&l.ID, &l.DeveloperID, &l.ProjectID,
		&l.Name, &l.Phone, &l.Email, &l.Budget, &l.Message,
		&l.Status, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	l.ProjectName = projectID // will be enriched in handler if needed
	return &l, nil
}

// ListLeadsByDeveloper returns paginated leads for a developer with project name.
func ListLeadsByDeveloper(ctx context.Context, pool *pgxpool.Pool, developerID string, limit, skip int) ([]Lead, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	var total int64
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM t_leads WHERE developer_id=$1`, developerID,
	).Scan(&total); err != nil {
		fmt.Printf("DEBUG: ListLeadsByDeveloper count error: %v\n", err)
		return nil, 0, err
	}

	const q = `
		SELECT l.id::text, l.developer_id::text, l.project_id::text, p.project_name,
		       l.name, l.phone, l.email, l.budget, l.message, l.status, l.created_at::text
		FROM t_leads l
		JOIN t_projects p ON l.project_id = p.id
		WHERE l.developer_id = $1::uuid
		ORDER BY l.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := pool.Query(ctx, q, developerID, limit, skip)
	if err != nil {
		fmt.Printf("DEBUG: ListLeadsByDeveloper query error: %v\n", err)
		return nil, 0, err
	}
	defer rows.Close()

	var out []Lead
	for rows.Next() {
		var l Lead
		if err := rows.Scan(
			&l.ID, &l.DeveloperID, &l.ProjectID, &l.ProjectName,
			&l.Name, &l.Phone, &l.Email, &l.Budget, &l.Message, &l.Status, &l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, l)
	}
	return out, total, rows.Err()
}

// UpdateLeadStatus changes a lead's status, guarded by developerID.
func UpdateLeadStatus(ctx context.Context, pool *pgxpool.Pool, leadID, developerID, status string) error {
	cmd, err := pool.Exec(ctx,
		`UPDATE t_leads SET status=$1, updated_at=now() WHERE id=$2 AND developer_id=$3`,
		status, leadID, developerID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("lead not found or access denied")
	}
	return nil
}
