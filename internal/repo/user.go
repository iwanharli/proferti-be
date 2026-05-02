package repo

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// User maps to t_users
type User struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	Password    *string `json:"-"`
	Role        string  `json:"role"`
	DeveloperID *string `json:"developerId,omitempty"`
	Image       *string `json:"image,omitempty"`
}

// RoleUpper returns the role in uppercase for compatibility with Nuxt middleware
// (middleware expects 'ADMIN' / 'DEVELOPER', DB stores 'admin' / 'developer')
func (u *User) RoleUpper() string {
	return strings.ToUpper(u.Role)
}

// GetUserByEmail fetches a user by email, including the hashed password for verification.
func GetUserByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (*User, error) {
	const q = `
		SELECT id, COALESCE(name,'') AS name, email, password, role::text,
		       developer_id::text, image
		FROM t_users
		WHERE email = $1
	`
	var u User
	err := pool.QueryRow(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.Password, &u.Role, &u.DeveloperID, &u.Image,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID fetches a user by primary key (no password returned).
func GetUserByID(ctx context.Context, pool *pgxpool.Pool, id string) (*User, error) {
	const q = `
		SELECT id, COALESCE(name,'') AS name, email, role::text,
		       developer_id::text, image
		FROM t_users
		WHERE id = $1
	`
	var u User
	err := pool.QueryRow(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &u.DeveloperID, &u.Image,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser inserts a new user with a pre-hashed password and default role 'developer'.
func CreateUser(ctx context.Context, pool *pgxpool.Pool, name, email, hashedPassword string) (*User, error) {
	const q = `
		INSERT INTO t_users (name, email, password, role)
		VALUES ($1, $2, $3, 'developer')
		RETURNING id, COALESCE(name,'') AS name, email, role::text
	`
	var u User
	err := pool.QueryRow(ctx, q, name, email, hashedPassword).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UpsertUserByEmail creates a user if they don't exist (keyed by email).
// Used for OAuth providers (GitHub) so user data is persisted to DB without Prisma adapter.
func UpsertUserByEmail(ctx context.Context, pool *pgxpool.Pool, email, name string, image *string) (*User, error) {
	const q = `
		INSERT INTO t_users (name, email, image, role)
		VALUES ($1, $2, $3, 'developer')
		ON CONFLICT (email) DO UPDATE
			SET name        = EXCLUDED.name,
			    image       = COALESCE(EXCLUDED.image, t_users.image),
			    updated_at  = now()
		RETURNING id, COALESCE(name,'') AS name, email, role::text, developer_id::text, image
	`
	var u User
	err := pool.QueryRow(ctx, q, name, email, image).Scan(
		&u.ID, &u.Name, &u.Email, &u.Role, &u.DeveloperID, &u.Image,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
