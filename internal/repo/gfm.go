package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GFMScene maps to gfm_scene table
type GFMScene struct {
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	STACItemID      string    `json:"stac_item_id"`
	AcquisitionTime time.Time `json:"acquisition_time"`
	ProductTime     time.Time `json:"product_time"`
	IngestedAt      time.Time `json:"ingested_at"`
	Platform        string    `json:"platform"`
	OrbitDirection  string    `json:"orbit_direction"`
	RelativeOrbit   int       `json:"relative_orbit"`
	BBox            []float64 `json:"bbox"`
	Footprint       string    `json:"footprint"`
	RawMetadata     string    `json:"raw_metadata"`
}

// GFMFloodPolygon maps to gfm_flood_polygon table
type GFMFloodPolygon struct {
	ID              string    `json:"id"`
	SceneID         string    `json:"scene_id"`
	AcquisitionTime time.Time `json:"acquisition_time"`
	AreaM2          float64   `json:"area_m2"`
	Confidence      float64   `json:"confidence_mean"`
	AdminCityCode   string    `json:"admin_city_code"`
}

// InsertGFMScene saves a new STAC item metadata
func InsertGFMScene(ctx context.Context, pool *pgxpool.Pool, s GFMScene) (string, error) {
	query := `
		INSERT INTO gfm_scene (stac_item_id, source, acquisition_time, product_time, platform, orbit_direction, relative_orbit, bbox, footprint, raw_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, ST_MakeEnvelope($8, $9, $10, $11, 4326), ST_GeomFromGeoJSON($12), $13)
		ON CONFLICT (stac_item_id) DO UPDATE 
		SET platform = EXCLUDED.platform,
		    orbit_direction = EXCLUDED.orbit_direction,
		    relative_orbit = EXCLUDED.relative_orbit,
		    bbox = EXCLUDED.bbox,
		    footprint = EXCLUDED.footprint,
		    raw_metadata = EXCLUDED.raw_metadata,
		    ingested_at = now()
		RETURNING id
	`
	var id string
	err := pool.QueryRow(ctx, query, 
		s.STACItemID, s.Source, s.AcquisitionTime, s.ProductTime, s.Platform,
		s.OrbitDirection, s.RelativeOrbit,
		s.BBox[0], s.BBox[1], s.BBox[2], s.BBox[3],
		s.Footprint, s.RawMetadata,
	).Scan(&id)
	return id, err
}

// GetGFMScenes fetches recently ingested scenes
func GetGFMScenes(ctx context.Context, pool *pgxpool.Pool, limit int) ([]GFMScene, error) {
	query := `
		SELECT id, stac_item_id, source, acquisition_time, product_time, platform, ingested_at
		FROM gfm_scene
		ORDER BY acquisition_time DESC
		LIMIT $1
	`
	rows, err := pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenes []GFMScene
	for rows.Next() {
		var s GFMScene
		err := rows.Scan(&s.ID, &s.STACItemID, &s.Source, &s.AcquisitionTime, &s.ProductTime, &s.Platform, &s.IngestedAt)
		if err != nil {
			return nil, err
		}
		scenes = append(scenes, s)
	}
	return scenes, nil
}

