package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LeadNote struct {
	ID             string     `json:"id"`
	LeadID         string     `json:"leadId"`
	UserID         string     `json:"userId"`
	UserName       string     `json:"userName"`
	Note           string     `json:"note"`
	NextFollowupAt *time.Time `json:"nextFollowupAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

func AddLeadNote(
	ctx context.Context, pool *pgxpool.Pool,
	leadID, userID, note string,
	nextFollowupAt *time.Time,
) (*LeadNote, error) {
	const q = `
		INSERT INTO t_lead_notes (lead_id, user_id, note, next_followup_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	var n LeadNote
	err := pool.QueryRow(ctx, q, leadID, userID, note, nextFollowupAt).Scan(&n.ID, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	n.LeadID = leadID
	n.UserID = userID
	n.Note = note
	n.NextFollowupAt = nextFollowupAt
	return &n, nil
}

func ListLeadNotes(ctx context.Context, pool *pgxpool.Pool, leadID string) ([]LeadNote, error) {
	const q = `
		SELECT n.id, n.lead_id, n.user_id, u.name, n.note, n.next_followup_at, n.created_at
		FROM t_lead_notes n
		INNER JOIN t_users u ON n.user_id = u.id
		WHERE n.lead_id = $1
		ORDER BY n.created_at DESC
	`
	rows, err := pool.Query(ctx, q, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []LeadNote
	for rows.Next() {
		var n LeadNote
		if err := rows.Scan(&n.ID, &n.LeadID, &n.UserID, &n.UserName, &n.Note, &n.NextFollowupAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
