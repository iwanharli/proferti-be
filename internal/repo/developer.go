package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Developer maps to t_developers.
// In Go BE schema, the 1:1 link is stored on t_users.developer_id (not on t_developers).
// UserID is resolved via JOIN when fetching by user.
type Developer struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	OwnerName   *string `json:"ownerName,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Email       *string `json:"email,omitempty"`
	Logo        *string `json:"logo,omitempty"`
	Description *string `json:"description,omitempty"`
	Website     *string `json:"website,omitempty"`
	RegionID    *int    `json:"regionId,omitempty"`
	UserID      string  `json:"userId"`
	CreatedAt   string  `json:"createdAt"`
}

// ErrAlreadyDeveloper is returned when the user already has a developer profile.
var ErrAlreadyDeveloper = errors.New("user already has a developer profile")

// GetDeveloperByUserID returns the developer profile associated with a given user.
// Uses JOIN because the link is stored on t_users.developer_id.
func GetDeveloperByUserID(ctx context.Context, pool *pgxpool.Pool, userID string) (*Developer, error) {
	const q = `
		SELECT d.id::text, d.company_name, d.slug, d.owner_name, d.phone, d.email,
		       d.logo, d.description, d.website, d.region_id, u.id::text, d.created_at::text
		FROM t_developers d
		JOIN t_users u ON u.developer_id = d.id
		WHERE u.id = $1::uuid
	`
	var d Developer
	err := pool.QueryRow(ctx, q, userID).Scan(
		&d.ID, &d.Name, &d.Slug, &d.OwnerName, &d.Phone, &d.Email,
		&d.Logo, &d.Description, &d.Website, &d.RegionID, &d.UserID, &d.CreatedAt,
	)
	if err != nil {
		fmt.Printf("DEBUG: GetDeveloperByUserID error for userID %s: %v\n", userID, err)
		return nil, err
	}
	return &d, nil
}

// CreateDeveloperForUser creates a developer profile inside a transaction:
//  1. Checks the user doesn't already have a developer_id.
//  2. Inserts into t_developers.
//  3. Updates t_users.developer_id + role = 'developer'.
func CreateDeveloperForUser(
	ctx context.Context, pool *pgxpool.Pool,
	userID, name, slug string,
	description, website, logo *string,
) (*Developer, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Guard: check user doesn't already own a developer profile
	var existingDevID *string
	err = tx.QueryRow(ctx,
		`SELECT developer_id::text FROM t_users WHERE id = $1`,
		userID,
	).Scan(&existingDevID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if existingDevID != nil {
		return nil, ErrAlreadyDeveloper
	}

	// 2. Insert developer
	const insQ = `
		INSERT INTO t_developers (company_name, slug, description, website, logo, region_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, company_name, slug, description, website, logo, region_id, created_at::text
	`
	var d Developer
	err = tx.QueryRow(ctx, insQ, name, slug, description, website, logo, nil).Scan( // region_id nil for now on create
		&d.ID, &d.Name, &d.Slug, &d.Description, &d.Website, &d.Logo, &d.RegionID, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.UserID = userID

	// 3. Link user → developer, elevate role
	_, err = tx.Exec(ctx,
		`UPDATE t_users
		 SET developer_id = $1, role = 'developer', updated_at = now()
		 WHERE id = $2`,
		d.ID, userID,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &d, nil
}

// GetDeveloperByID returns a developer profile by its primary key.
func GetDeveloperByID(ctx context.Context, pool *pgxpool.Pool, id string) (*Developer, error) {
	const q = `
		SELECT id::text, company_name, slug, owner_name, phone, email,
		       logo, description, website, region_id, created_at::text
		FROM t_developers
		WHERE id = $1
	`
	var d Developer
	err := pool.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.Name, &d.Slug, &d.OwnerName, &d.Phone, &d.Email,
		&d.Logo, &d.Description, &d.Website, &d.RegionID, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// UpdateDeveloper updates an existing developer profile.
func UpdateDeveloper(
	ctx context.Context, pool *pgxpool.Pool,
	id string, name string, slug string,
	description, website, logo *string,
) error {
	const q = `
		UPDATE t_developers
		SET company_name = $1, slug = $2, description = $3, website = $4, logo = $5, updated_at = now()
		WHERE id = $6
	`
	_, err := pool.Exec(ctx, q, name, slug, description, website, logo, id)
	return err
}

// GetDeveloperBySlug returns a developer profile by its slug.
func GetDeveloperBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*Developer, error) {
	const q = `
		SELECT id::text, company_name, slug, owner_name, phone, email,
		       logo, description, website, region_id, created_at::text
		FROM t_developers
		WHERE slug = $1
	`
	var d Developer
	err := pool.QueryRow(ctx, q, slug).Scan(
		&d.ID, &d.Name, &d.Slug, &d.OwnerName, &d.Phone, &d.Email,
		&d.Logo, &d.Description, &d.Website, &d.RegionID, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
