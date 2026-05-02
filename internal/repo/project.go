package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectListFilters struct {
	City         string
	DeveloperID  string // Keep for compatibility
	DeveloperIDs []string
	Status       string
	MinPrice     *float64
	MaxPrice     *float64
	Search       string
	Type         string
	Sort         string
	Bedrooms     *int
	Bathrooms    *int
	Limit        int
	Skip         int
}

type DeveloperBrief struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Slug string  `json:"slug"`
	Logo *string `json:"logo,omitempty"`
}

type Region struct {
	ID   int    `json:"id"`
	Kode string `json:"kode"`
	Name string `json:"name"`
}

type Location struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Address   *string  `json:"address,omitempty"`
	RegionID  *int     `json:"regionId,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	City      string   `json:"city"`               // Deprecated: used for frontend compat
	Province  *string  `json:"province,omitempty"` // Deprecated: used for frontend compat
	Region    *Region  `json:"region,omitempty"`
}

type ProjectListRow struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Slug         string         `json:"slug"`
	Location     Location       `json:"location"`
	Description  *string        `json:"description,omitempty"`
	StartPrice   float64        `json:"startPrice"`
	Promo        *string        `json:"promo,omitempty"`
	Image        *string        `json:"image,omitempty"`
	Status       string         `json:"status"`
	Type         *string        `json:"type,omitempty"`
	CreatedAt    string         `json:"createdAt"`
	Developer    DeveloperBrief `json:"developer"`
	Polygon      any            `json:"polygon"`
	GalleryCount int            `json:"galleryCount"`
}

type ProjectGalleryRow struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type ProjectDetail struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Slug        string              `json:"slug"`
	Location    Location            `json:"location"`
	Description *string             `json:"description,omitempty"`
	StartPrice  float64             `json:"startPrice"`
	Promo       *string             `json:"promo,omitempty"`
	Image       *string             `json:"image,omitempty"`
	Status      string              `json:"status"`
	Type        *string             `json:"type,omitempty"`
	CreatedAt   string              `json:"createdAt"`
	Developer   struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Slug        string  `json:"slug"`
		Logo        *string `json:"logo,omitempty"`
		Description *string `json:"description,omitempty"`
		Website     *string `json:"website,omitempty"`
	} `json:"developer"`
	Gallery []ProjectGalleryRow `json:"gallery"`
	UnitTypes []UnitType          `json:"unitTypes,omitempty"`
	Polygon   any                 `json:"polygon"`
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
		where.WriteString(fmt.Sprintf(` AND (r.name ILIKE $%d OR l.name ILIKE $%d) `, n, n+1))
		args = append(args, "%"+f.City+"%", "%"+f.City+"%")
		n += 2
	}
	if f.DeveloperID != "" {
		where.WriteString(fmt.Sprintf(` AND p.developer_id = $%d `, n))
		args = append(args, f.DeveloperID)
		n++
	}
	if len(f.DeveloperIDs) > 0 {
		where.WriteString(fmt.Sprintf(` AND p.developer_id = ANY($%d) `, n))
		args = append(args, f.DeveloperIDs)
		n++
	}
	if f.Status != "" && f.Status != "active" {
		where.WriteString(fmt.Sprintf(` AND p.status::text = $%d `, n))
		args = append(args, f.Status)
		n++
	}
	if f.Type != "" {
		where.WriteString(fmt.Sprintf(` AND p.project_type = $%d `, n))
		args = append(args, f.Type)
		n++
	}
	if f.MinPrice != nil {
		where.WriteString(fmt.Sprintf(` AND p.starting_price >= $%d `, n))
		args = append(args, *f.MinPrice)
		n++
	}
	if f.MaxPrice != nil {
		where.WriteString(fmt.Sprintf(` AND p.starting_price <= $%d `, n))
		args = append(args, *f.MaxPrice)
		n++
	}
	if f.Bedrooms != nil {
		where.WriteString(fmt.Sprintf(` AND EXISTS (SELECT 1 FROM t_project_unit_types ut WHERE ut.project_id = p.id AND ut.bedroom >= $%d) `, n))
		args = append(args, *f.Bedrooms)
		n++
	}
	if f.Bathrooms != nil {
		where.WriteString(fmt.Sprintf(` AND EXISTS (SELECT 1 FROM t_project_unit_types ut WHERE ut.project_id = p.id AND ut.bathroom >= $%d) `, n))
		args = append(args, *f.Bathrooms)
		n++
	}
	if f.Search != "" {
		pat := "%" + f.Search + "%"
		where.WriteString(fmt.Sprintf(
			` AND (p.project_name ILIKE $%d OR l.name ILIKE $%d OR r.name ILIKE $%d OR d.company_name ILIKE $%d) `,
			n, n+1, n+2, n+3))
		args = append(args, pat, pat, pat, pat)
		n += 4
	}

	countSQL := `SELECT COUNT(*) FROM t_projects p 
	             INNER JOIN t_developers d ON p.developer_id = d.id
				 LEFT JOIN t_project_locations l ON p.location_id = l.id
				 LEFT JOIN regions r ON l.region_id = r.id` + where.String()

	var total int64
	if err := pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "p.created_at DESC"
	if f.Sort == "price_asc" {
		orderBy = "p.starting_price ASC"
	} else if f.Sort == "price_desc" {
		orderBy = "p.starting_price DESC"
	}

	listSQL := `
SELECT
  p.id, p.project_name, p.slug, p.description, p.starting_price, p.promo_text, p.cover_image, p.status::text,
  p.created_at::text, p.project_type,
  l.id, l.name, l.address, l.region_id, l.latitude, l.longitude,
  r.id, r.kode, r.name,
  d.id, d.company_name, d.slug, d.logo,
  p.polygon_coordinates,
  (SELECT COUNT(*)::int FROM t_project_galleries pi WHERE pi.project_id = p.id)
FROM t_projects p
INNER JOIN t_developers d ON p.developer_id = d.id
LEFT JOIN t_project_locations l ON p.location_id = l.id
LEFT JOIN regions r ON l.region_id = r.id
` + where.String() + fmt.Sprintf(` ORDER BY %s LIMIT $%d OFFSET $%d`, orderBy, n, n+1)

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
		var regID *int
		var regKode, regName *string
		if err := rows.Scan(
			&r.ID, &r.Name, &r.Slug, &r.Description, &r.StartPrice, &r.Promo, &r.Image, &r.Status,
			&r.CreatedAt, &r.Type,
			&r.Location.ID, &r.Location.Name, &r.Location.Address, &r.Location.RegionID,
			&r.Location.Latitude, &r.Location.Longitude,
			&regID, &regKode, &regName,
			&r.Developer.ID, &r.Developer.Name, &r.Developer.Slug, &devLogo,
			&r.Polygon,
			&r.GalleryCount,
		); err != nil {
			return nil, 0, err
		}
		if regID != nil {
			r.Location.Region = &Region{
				ID:   *regID,
				Kode: *regKode,
				Name: *regName,
			}
			r.Location.City = *regName // Simplification for frontend
		}
		r.Developer.Logo = devLogo
		out = append(out, r)
	}
	return out, total, rows.Err()
}

func GetProjectByID(ctx context.Context, pool *pgxpool.Pool, id string) (*ProjectDetail, error) {
	return getProject(ctx, pool, "p.id", id)
}

func GetProjectBySlug(ctx context.Context, pool *pgxpool.Pool, slug string) (*ProjectDetail, error) {
	return getProject(ctx, pool, "p.slug", slug)
}

func getProject(ctx context.Context, pool *pgxpool.Pool, field, val string) (*ProjectDetail, error) {
	q := fmt.Sprintf(`
SELECT
  p.id, p.project_name, p.slug, p.description, p.starting_price, p.promo_text, p.cover_image, p.status::text, p.created_at::text,
  l.id, l.name, l.address, l.region_id, l.latitude, l.longitude,
  r.id, r.kode, r.name,
  d.id, d.company_name, d.slug, d.logo, d.description, d.website,
  p.polygon_coordinates
FROM t_projects p
INNER JOIN t_developers d ON p.developer_id = d.id
LEFT JOIN t_project_locations l ON p.location_id = l.id
LEFT JOIN regions r ON l.region_id = r.id
WHERE %s = $1
`, field)

	var d ProjectDetail
	var devLogo, devDesc, devWeb *string
	var regID *int
	var regKode, regName *string
	err := pool.QueryRow(ctx, q, val).Scan(
		&d.ID, &d.Name, &d.Slug, &d.Description, &d.StartPrice, &d.Promo, &d.Image, &d.Status, &d.CreatedAt,
		&d.Location.ID, &d.Location.Name, &d.Location.Address, &d.Location.RegionID,
		&d.Location.Latitude, &d.Location.Longitude,
		&regID, &regKode, &regName,
		&d.Developer.ID, &d.Developer.Name, &d.Developer.Slug, &devLogo, &devDesc, &devWeb,
		&d.Polygon,
	)
	if err != nil {
		return nil, err
	}
	if regID != nil {
		d.Location.Region = &Region{
			ID:   *regID,
			Kode: *regKode,
			Name: *regName,
		}
		d.Location.City = *regName // Compat
	}
	d.Developer.Logo = devLogo
	d.Developer.Description = devDesc
	d.Developer.Website = devWeb

	// Fetch gallery and unit types using d.ID (the internal UUID)
	gq := `SELECT id, image FROM t_project_galleries WHERE project_id = $1 ORDER BY id ASC LIMIT 20`
	grows, err := pool.Query(ctx, gq, d.ID)
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

	uq := `SELECT id, project_id, type_name, slug, land_size, building_size, bedroom, bathroom, garage, price, stock FROM t_project_unit_types WHERE project_id = $1 ORDER BY created_at ASC`
	urows, err := pool.Query(ctx, uq, d.ID)
	if err != nil {
		return nil, err
	}
	defer urows.Close()
	for urows.Next() {
		var u UnitType
		if err := urows.Scan(&u.ID, &u.ProjectID, &u.TypeName, &u.Slug, &u.LandSize, &u.BuildingSize, &u.Bedroom, &u.Bathroom, &u.Garage, &u.Price, &u.Stock); err != nil {
			return nil, err
		}
		d.UnitTypes = append(d.UnitTypes, u)
	}

	return &d, urows.Err()
}

func ListLocations(ctx context.Context, pool *pgxpool.Pool) ([]Location, error) {
	rows, err := pool.Query(ctx, `
		SELECT DISTINCT ON (l.id) l.id, l.name, l.address, l.region_id, l.latitude, l.longitude, r.id, r.kode, r.name
		FROM t_project_locations l
		INNER JOIN t_projects p ON p.location_id = l.id
		LEFT JOIN regions r ON l.region_id = r.id
		ORDER BY l.id, r.name ASC, l.name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Location
	for rows.Next() {
		var l Location
		var regID *int
		var regKode, regName *string
		if err := rows.Scan(&l.ID, &l.Name, &l.Address, &l.RegionID, &l.Latitude, &l.Longitude, &regID, &regKode, &regName); err != nil {
			return nil, err
		}
		if regID != nil {
			l.Region = &Region{
				ID:   *regID,
				Kode: *regKode,
				Name: *regName,
			}
			l.City = *regName // Compat
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func GetProjectsMeta(ctx context.Context, pool *pgxpool.Pool) (map[string]any, error) {
	var types []string
	var minPrice, maxPrice float64

	// Get unique types
	rows, err := pool.Query(ctx, "SELECT DISTINCT project_type FROM t_projects WHERE project_type IS NOT NULL AND project_type != '' ORDER BY project_type ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err == nil {
			types = append(types, t)
		}
	}

	// Get price range
	err = pool.QueryRow(ctx, "SELECT COALESCE(MIN(starting_price), 0), COALESCE(MAX(starting_price), 0) FROM t_projects").Scan(&minPrice, &maxPrice)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"types":    types,
		"minPrice": minPrice,
		"maxPrice": maxPrice,
	}, nil
}

func GetRegionsGeoJSON(ctx context.Context, pool *pgxpool.Pool, parentKode string) (map[string]any, error) {
	var whereClause string
	if parentKode != "" {
		// Fetch cities for a specific province
		whereClause = fmt.Sprintf("WHERE kode LIKE '%s.%%' AND LENGTH(kode) > 2", parentKode)
	} else {
		// Only fetch provinces (kode length = 2) for performance and clarity
		whereClause = "WHERE LENGTH(kode) = 2"
	}

	query := fmt.Sprintf(`
		SELECT json_build_object(
			'type', 'FeatureCollection',
			'features', COALESCE(json_agg(
				json_build_object(
					'type', 'Feature',
					'id', id,
					'geometry', ST_AsGeoJSON(geom)::json,
					'properties', json_build_object(
						'id', id,
						'kode', kode,
						'name', name
					)
				)
			), '[]'::json)
		)
		FROM (
			SELECT id, kode, name, geom 
			FROM regions 
			%s
		) AS t
	`, whereClause)

	var result map[string]any
	err := pool.QueryRow(ctx, query).Scan(&result)
	return result, err
}

func GetRegionByPoint(ctx context.Context, pool *pgxpool.Pool, lat, lng float64) (string, error) {
	// Find the most specific region (longest kode) containing the point
	query := `
		SELECT name 
		FROM regions 
		WHERE ST_Contains(geom, ST_SetSRID(ST_Point($1, $2), 4326))
		  AND LENGTH(kode) = 5
		LIMIT 1
	`
	var name string
	err := pool.QueryRow(ctx, query, lng, lat).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}
