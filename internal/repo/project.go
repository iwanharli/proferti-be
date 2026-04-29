package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectListFilters struct {
	City         string
	DeveloperID  string
	Status       string
	MinPrice     *float64
	MaxPrice     *float64
	Search       string
	Limit        int
	Skip         int
}

type DeveloperBrief struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Logo *string `json:"logo,omitempty"`
}

type ProjectListRow struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Location     string         `json:"location"`
	Description  *string        `json:"description,omitempty"`
	StartPrice   float64        `json:"startPrice"`
	Promo        *string        `json:"promo,omitempty"`
	Image        *string        `json:"image,omitempty"`
	Status       string         `json:"status"`
	CreatedAt    string         `json:"createdAt"`
	Developer    DeveloperBrief `json:"developer"`
	GalleryCount int            `json:"galleryCount"`
}

type ProjectGalleryRow struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type ProjectDetail struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Location    string              `json:"location"`
	Description *string             `json:"description,omitempty"`
	StartPrice  float64             `json:"startPrice"`
	Promo       *string             `json:"promo,omitempty"`
	Image       *string             `json:"image,omitempty"`
	Status      string              `json:"status"`
	CreatedAt   string              `json:"createdAt"`
	Developer   struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Logo        *string `json:"logo,omitempty"`
		Description *string `json:"description,omitempty"`
		Website     *string `json:"website,omitempty"`
	} `json:"developer"`
	Gallery []ProjectGalleryRow `json:"gallery"`
}

func ListProjects(ctx context.Context, pool *pgxpool.Pool, f ProjectListFilters) ([]ProjectListRow, int64, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Skip < 0 {
		f.Skip = 0
	}

	var where strings.Builder
	var args []any
	n := 1

	where.WriteString(` WHERE 1=1 `)

	if f.City != "" {
		where.WriteString(fmt.Sprintf(` AND p.location ILIKE $%d `, n))
		args = append(args, "%"+f.City+"%")
		n++
	}
	if f.DeveloperID != "" {
		where.WriteString(fmt.Sprintf(` AND p."developerId" = $%d `, n))
		args = append(args, f.DeveloperID)
		n++
	}
	if f.Status != "" {
		where.WriteString(fmt.Sprintf(` AND p.status::text = $%d `, n))
		args = append(args, f.Status)
		n++
	}
	if f.MinPrice != nil {
		where.WriteString(fmt.Sprintf(` AND p."startPrice" >= $%d `, n))
		args = append(args, *f.MinPrice)
		n++
	}
	if f.MaxPrice != nil {
		where.WriteString(fmt.Sprintf(` AND p."startPrice" <= $%d `, n))
		args = append(args, *f.MaxPrice)
		n++
	}
	if f.Search != "" {
		pat := "%" + f.Search + "%"
		where.WriteString(fmt.Sprintf(
			` AND (p.name ILIKE $%d OR p.location ILIKE $%d OR d.name ILIKE $%d) `,
			n, n+1, n+2))
		args = append(args, pat, pat, pat)
		n += 3
	}

	countSQL := `SELECT COUNT(*) FROM "Project" p INNER JOIN "Developer" d ON p."developerId" = d.id` + where.String()

	var total int64
	if err := pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL := `
SELECT
  p.id, p.name, p.location, p.description, p."startPrice", p.promo, p.image, p.status::text,
  p."createdAt"::text,
  d.id, d.name, d.logo,
  (SELECT COUNT(*)::int FROM "ProjectImage" pi WHERE pi."projectId" = p.id)
FROM "Project" p
INNER JOIN "Developer" d ON p."developerId" = d.id
` + where.String() + fmt.Sprintf(` ORDER BY p."createdAt" DESC LIMIT $%d OFFSET $%d`, n, n+1)

	args = append(args, f.Limit, f.Skip)

	rows, err := pool.Query(ctx, listSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []ProjectListRow
	for rows.Next() {
		var r ProjectListRow
		var devLogo *string
		err := rows.Scan(
			&r.ID, &r.Name, &r.Location, &r.Description, &r.StartPrice, &r.Promo, &r.Image, &r.Status,
			&r.CreatedAt,
			&r.Developer.ID, &r.Developer.Name, &devLogo,
			&r.GalleryCount,
		)
		if err != nil {
			return nil, 0, err
		}
		r.Developer.Logo = devLogo
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func GetProjectByID(ctx context.Context, pool *pgxpool.Pool, id string) (*ProjectDetail, error) {
	const q = `
SELECT
  p.id, p.name, p.location, p.description, p."startPrice", p.promo, p.image, p.status::text, p."createdAt"::text,
  d.id, d.name, d.logo, d.description, d.website
FROM "Project" p
INNER JOIN "Developer" d ON p."developerId" = d.id
WHERE p.id = $1
`

	var d ProjectDetail
	var devLogo, devDesc, devWeb *string
	err := pool.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.Name, &d.Location, &d.Description, &d.StartPrice, &d.Promo, &d.Image, &d.Status, &d.CreatedAt,
		&d.Developer.ID, &d.Developer.Name, &devLogo, &devDesc, &devWeb,
	)
	if err != nil {
		return nil, err
	}
	d.Developer.Logo = devLogo
	d.Developer.Description = devDesc
	d.Developer.Website = devWeb

	gq := `SELECT id, url FROM "ProjectImage" WHERE "projectId" = $1 ORDER BY id ASC LIMIT 20`
	grows, err := pool.Query(ctx, gq, id)
	if err != nil {
		return nil, err
	}
	defer grows.Close()
	for grows.Next() {
		var g ProjectGalleryRow
		if err := grows.Scan(&g.ID, &g.URL); err != nil {
			return nil, err
		}
		d.Gallery = append(d.Gallery, g)
	}
	return &d, grows.Err()
}
