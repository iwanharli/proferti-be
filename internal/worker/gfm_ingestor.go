package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"proferti-be/internal/repo"
	"github.com/jackc/pgx/v5/pgxpool"
)

type STACSearchRequest struct {
	Collections []string `json:"collections"`
	Datetime    string   `json:"datetime"`
	Limit       int      `json:"limit"`
	Intersects  any      `json:"intersects"`
}

type STACSearchResponse struct {
	Features []STACFeature `json:"features"`
}

type STACFeature struct {
	ID         string `json:"id"`
	BBox       []float64 `json:"bbox"`
	Geometry   any `json:"geometry"`
	Properties map[string]any `json:"properties"`
	Assets map[string]struct {
		Href string `json:"href"`
	} `json:"assets"`
}

// RunFullIngestionCycle runs the ingestion for all unique project locations with optional date range.
func RunFullIngestionCycle(ctx context.Context, pool *pgxpool.Pool, startDate, endDate string) error {
	rows, err := pool.Query(ctx, "SELECT DISTINCT longitude, latitude FROM t_project_locations WHERE longitude IS NOT NULL AND latitude IS NOT NULL")
	if err != nil {
		return fmt.Errorf("failed to query project locations: %v", err)
	}
	defer rows.Close()

	radiusDeg := 0.45 // ~50km (Lebih fokus dan cepat untuk pemantauan proyek)
	processedScenes := make(map[string]bool)
	var results []string

	for rows.Next() {
		var lng, lat float64
		if err := rows.Scan(&lng, &lat); err != nil {
			continue
		}

		bbox := [4]float64{
			lng - radiusDeg,
			lat - radiusDeg,
			lng + radiusDeg,
			lat + radiusDeg,
		}

		fmt.Printf("\n📡 Processing area around [%f, %f] (Radius 50km)...\n", lng, lat)

		sceneIDs, err := FetchLatestGFMScenes(ctx, pool, bbox, startDate, endDate)
		if err != nil {
			fmt.Printf("Error fetching scenes for location %f, %f: %v\n", lng, lat, err)
			continue
		}

		for _, sceneID := range sceneIDs {
			if processedScenes[sceneID] {
				fmt.Printf("   ➜ Skipping already processed scene (memory): %s\n", sceneID)
				continue
			}

			// Check if polygons already exist in DB to skip heavy processing
			var exists bool
			pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM gfm_flood_polygon WHERE scene_id = $1 LIMIT 1)", sceneID).Scan(&exists)
			if exists {
				fmt.Printf("   ➜ Skipping already processed scene (database): %s\n", sceneID)
				processedScenes[sceneID] = true
				continue
			}

			err = ProcessGFMScene(ctx, pool, sceneID, bbox)
			if err != nil {
				fmt.Printf("Processing failed for %s: %v\n", sceneID, err)
				results = append(results, fmt.Sprintf("Processing failed for %s: %v", sceneID, err))
			} else {
				results = append(results, "Success for "+sceneID)
				processedScenes[sceneID] = true
			}
		}
	}

	fmt.Printf("\n✅ Ingestion cycle completed. Results: %d success/fail items.\n", len(results))
	return nil
}

// FetchLatestGFMScenes searches for GFM items and saves metadata to DB. Returns a slice of scene IDs.
func FetchLatestGFMScenes(ctx context.Context, pool *pgxpool.Pool, bbox [4]float64, startDate, endDate string) ([]string, error) {
	// Jakarta BBox: [106.60, -6.40, 107.10, -5.90]
	intersects := map[string]any{
		"type": "Polygon",
		"coordinates": [][][]float64{{
			{bbox[0], bbox[1]},
			{bbox[2], bbox[1]},
			{bbox[2], bbox[3]},
			{bbox[0], bbox[3]},
			{bbox[0], bbox[1]},
		}},
	}

	// Default date range: last 60 days if not provided (2 months)
	dateRange := fmt.Sprintf("%s/%s", time.Now().AddDate(0, 0, -60).Format(time.RFC3339), time.Now().Format(time.RFC3339))
	
	if startDate != "" && endDate != "" {
		// Normalize YYYY-MM-DD to RFC3339 if needed
		s := startDate
		if len(s) == 10 {
			s = s + "T00:00:00Z"
		}
		e := endDate
		if len(e) == 10 {
			e = e + "T23:59:59Z"
		}
		dateRange = fmt.Sprintf("%s/%s", s, e)
	}

	reqBody := STACSearchRequest{
		Collections: []string{"GFM"},
		Datetime:    dateRange,
		Limit:       10,
		Intersects:  intersects,
	}

	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post("https://stac.eodc.eu/api/v1/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stacResp STACSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&stacResp); err != nil {
		return nil, err
	}

	var sceneIDs []string
	for _, feat := range stacResp.Features {
		// Extract specific fields from properties map
		acqTimeStr, _ := feat.Properties["datetime"].(string)
		acqTime, _ := time.Parse(time.RFC3339, acqTimeStr)
		
		platform, _ := feat.Properties["platform"].(string)
		if platform == "" {
			platform, _ = feat.Properties["constellation"].(string)
		}
		
		// If platform is still sentinel-1, try to get specific S1A/S1B from parent
		parent, _ := feat.Properties["parent"].(string)
		if parent != "" && len(parent) > 3 {
			platform = parent[:3] // e.g. "S1A" or "S1B"
		}
		
		orbitDir, _ := feat.Properties["sat:orbit_state"].(string)
		if orbitDir == "" {
			// Smart Inference based on Jakarta Time
			loc, _ := time.LoadLocation("Asia/Jakarta")
			acqLocal := acqTime.In(loc)
			if acqLocal.Hour() < 12 {
				orbitDir = "descending"
			} else {
				orbitDir = "ascending"
			}
		}
		relOrbit, _ := feat.Properties["sat:relative_orbit"].(float64)

		// Convert Geometry to JSON string for footprint
		geomJSON, _ := json.Marshal(feat.Geometry)
		rawMetaJSON, _ := json.Marshal(feat.Properties)

		scene := repo.GFMScene{
			Source:          "copernicus_gfm",
			STACItemID:      feat.ID,
			AcquisitionTime: acqTime,
			ProductTime:     time.Now(), // Placeholder or from metadata
			Platform:        platform,
			OrbitDirection:  orbitDir,
			RelativeOrbit:   int(relOrbit),
			BBox:            feat.BBox,
			Footprint:       string(geomJSON),
			RawMetadata:     string(rawMetaJSON),
		}

		sceneID, err := repo.InsertGFMScene(ctx, pool, scene)
		if err != nil {
			fmt.Printf("Error inserting scene %s: %v\n", feat.ID, err)
			continue
		}

		sceneIDs = append(sceneIDs, sceneID)

		// Save assets
		for band, asset := range feat.Assets {
			// We only care about some bands for now
			if band == "ensemble_flood_extent" || band == "ensemble_likelihood" || band == "exclusion_mask" {
				_, err := pool.Exec(ctx, `
					INSERT INTO gfm_asset (scene_id, band_name, asset_href)
					VALUES ($1, $2, $3)
					ON CONFLICT (scene_id, band_name) DO NOTHING
				`, sceneID, band, asset.Href)
				if err != nil {
					fmt.Printf("Error inserting asset %s for scene %s: %v\n", band, feat.ID, err)
				}
			}
		}
	}

	return sceneIDs, nil
}

// ProcessGFMScene downloads, clips, and polygonizes a GFM raster
func ProcessGFMScene(ctx context.Context, pool *pgxpool.Pool, sceneID string, bbox [4]float64) error {
	var href string
	var acqTime time.Time
	err := pool.QueryRow(ctx, "SELECT asset_href, acquisition_time FROM gfm_asset a JOIN gfm_scene s ON a.scene_id = s.id WHERE a.scene_id = $1 AND a.band_name = 'ensemble_flood_extent'", sceneID).Scan(&href, &acqTime)
	if err != nil {
		return err
	}
	dateStr := acqTime.Format("2006-01-02")

	tempDir := "./data/gfm/" + sceneID
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	gdalBin := os.Getenv("GDAL_BIN_PATH")
	gdalExecutable := "gdalwarp"
	rasterExecutable := "raster2pgsql"
	if gdalBin != "" {
		gdalExecutable = filepath.Join(gdalBin, "gdalwarp")
		rasterExecutable = filepath.Join(gdalBin, "raster2pgsql")
	}

	// 1. Download and clip 'ensemble_flood_extent'
	tifPath := filepath.Join(tempDir, "clip_extent.tif")
	warpCmd := exec.Command(gdalExecutable,
		"-overwrite", "-t_srs", "EPSG:4326", "-te_srs", "EPSG:4326",
		"-te", fmt.Sprintf("%f", bbox[0]), fmt.Sprintf("%f", bbox[1]), fmt.Sprintf("%f", bbox[2]), fmt.Sprintf("%f", bbox[3]),
		"-of", "GTiff", "/vsicurl/"+href, tifPath,
	)
	if out, err := warpCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gdalwarp extent failed: %v, output: %s", err, string(out))
	}

	// 1.1 Download and clip 'ensemble_likelihood'
	var likelihoodHref string
	err = pool.QueryRow(ctx, "SELECT asset_href FROM gfm_asset WHERE scene_id = $1 AND band_name = 'ensemble_likelihood'", sceneID).Scan(&likelihoodHref)
	likelihoodTifPath := filepath.Join(tempDir, "clip_likelihood.tif")
	if err == nil {
		warpLikelihood := exec.Command(gdalExecutable,
			"-overwrite", "-t_srs", "EPSG:4326", "-te_srs", "EPSG:4326",
			"-te", fmt.Sprintf("%f", bbox[0]), fmt.Sprintf("%f", bbox[1]), fmt.Sprintf("%f", bbox[2]), fmt.Sprintf("%f", bbox[3]),
			"-of", "GTiff", "/vsicurl/"+likelihoodHref, likelihoodTifPath,
		)
		warpLikelihood.Run() // If it fails, we just don't get confidence data
	}

	// 1.2 Download and clip 'exclusion_mask'
	var exclusionHref string
	err = pool.QueryRow(ctx, "SELECT asset_href FROM gfm_asset WHERE scene_id = $1 AND band_name = 'exclusion_mask'", sceneID).Scan(&exclusionHref)
	exclusionTifPath := filepath.Join(tempDir, "clip_exclusion.tif")
	if err == nil {
		warpExclusion := exec.Command(gdalExecutable,
			"-overwrite", "-t_srs", "EPSG:4326", "-te_srs", "EPSG:4326",
			"-te", fmt.Sprintf("%f", bbox[0]), fmt.Sprintf("%f", bbox[1]), fmt.Sprintf("%f", bbox[2]), fmt.Sprintf("%f", bbox[3]),
			"-of", "GTiff", "/vsicurl/"+exclusionHref, exclusionTifPath,
		)
		warpExclusion.Run()
	}

	// 2. raster2pgsql for EXTENT
	fmt.Printf("   ➜ [%s] Converting raster to polygons...\n", dateStr)
	rastCmd := exec.Command(rasterExecutable,
		"-s", "4326", "-F", tifPath, "temp_flood_raster_"+sceneID,
	)
	sqlOut, err := rastCmd.Output()
	if err != nil {
		return fmt.Errorf("raster2pgsql failed: %v", err)
	}

	// 3. Execute the generated SQL
	tempTable := fmt.Sprintf("\"temp_flood_raster_%s\"", sceneID)
	_, err = pool.Exec(ctx, "DROP TABLE IF EXISTS "+tempTable)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, string(sqlOut))
	if err != nil {
		return fmt.Errorf("failed to execute raster SQL: %v", err)
	}

	// 3.5 Add spatial index to speed up DumpAsPolygons
	_, err = pool.Exec(ctx, fmt.Sprintf("CREATE INDEX ON %s USING gist (ST_ConvexHull(rast))", tempTable))
	if err != nil {
		fmt.Printf("Warning: failed to create spatial index on temp table: %v\n", err)
	}

	// 3.6 Ingest Likelihood raster to temp table
	if _, err := os.Stat(likelihoodTifPath); err == nil {
		likelihoodRastCmd := exec.Command(filepath.Join(gdalBin, "raster2pgsql"),
			"-s", "4326", "-F", likelihoodTifPath, "temp_likelihood_raster_"+sceneID,
		)
		if sqlLikelihood, err := likelihoodRastCmd.Output(); err == nil {
			pool.Exec(ctx, "DROP TABLE IF EXISTS \"temp_likelihood_raster_"+sceneID+"\"")
			pool.Exec(ctx, string(sqlLikelihood))
			// Add spatial index to speed up confidence calculation
			pool.Exec(ctx, fmt.Sprintf("CREATE INDEX ON \"temp_likelihood_raster_%s\" USING gist (ST_ConvexHull(rast))", sceneID))
		}
	}

	// 3.7 Ingest Exclusion raster to temp table
	if _, err := os.Stat(exclusionTifPath); err == nil {
		exclusionRastCmd := exec.Command(filepath.Join(gdalBin, "raster2pgsql"),
			"-s", "4326", "-F", exclusionTifPath, "temp_exclusion_raster_"+sceneID,
		)
		if sqlExclusion, err := exclusionRastCmd.Output(); err == nil {
			pool.Exec(ctx, "DROP TABLE IF EXISTS \"temp_exclusion_raster_"+sceneID+"\"")
			pool.Exec(ctx, string(sqlExclusion))
			pool.Exec(ctx, fmt.Sprintf("CREATE INDEX ON \"temp_exclusion_raster_%s\" USING gist (ST_ConvexHull(rast))", sceneID))
		}
	}

	// 4. Ingest from raster tables to final polygon tables
	err = IngestFromRasterTable(ctx, pool, sceneID)
	if err == nil {
		IngestExclusionFromRasterTable(ctx, pool, sceneID)
		fmt.Printf("   ✔ [%s] Area processing complete.\n", dateStr)
	}
	
	// 5. Cleanup (Database & Files)
	pool.Exec(ctx, "DROP TABLE IF EXISTS "+tempTable)
	pool.Exec(ctx, "DROP TABLE IF EXISTS \"temp_likelihood_raster_"+sceneID+"\"")
	pool.Exec(ctx, "DROP TABLE IF EXISTS \"temp_exclusion_raster_"+sceneID+"\"")
	
	return err
}

func IngestExclusionFromRasterTable(ctx context.Context, pool *pgxpool.Pool, sceneID string) {
	tempTable := fmt.Sprintf("\"temp_exclusion_raster_%s\"", sceneID)
	// Only ingest Urban (1) and Radar Shadow (5) for property safety context
	query := fmt.Sprintf(`
		WITH dumped AS (
			SELECT (ST_DumpAsPolygons(rast)).*
			FROM %s
		)
		INSERT INTO gfm_exclusion_polygon (scene_id, exclusion_type, geom, area_m2)
		SELECT 
			$1, val::int,
			ST_Multi(geom),
			ST_Area(ST_Transform(geom, 3857))
		FROM dumped
		WHERE val IN (1, 5)
	`, tempTable)

	_, err := pool.Exec(ctx, query, sceneID)
	if err != nil {
		fmt.Printf("Warning: failed to ingest exclusion polygons: %v\n", err)
	}
}

// IngestFromRasterTable converts raster pixels to polygons in DB
func IngestFromRasterTable(ctx context.Context, pool *pgxpool.Pool, sceneID string) error {
	var acqTime time.Time
	err := pool.QueryRow(ctx, "SELECT acquisition_time FROM gfm_scene WHERE id = $1", sceneID).Scan(&acqTime)
	if err != nil {
		return err
	}

	tempTable := fmt.Sprintf("\"temp_flood_raster_%s\"", sceneID)
	query := fmt.Sprintf(`
		WITH dumped AS (
			SELECT (ST_DumpAsPolygons(rast)).*
			FROM %s
		)
		INSERT INTO gfm_flood_polygon (scene_id, acquisition_time, geom, centroid, area_m2)
		SELECT 
			$1, $2,
			ST_Multi(geom),
			ST_Centroid(geom),
			ST_Area(ST_Transform(geom, 3857))
		FROM dumped
		WHERE val = 1
	`, tempTable)

	// Check unique values in the raster
	rows, err := pool.Query(ctx, fmt.Sprintf("SELECT val, count(*) FROM (SELECT (ST_DumpAsPolygons(rast)).val FROM %s) s GROUP BY val ORDER BY count(*) DESC", tempTable))
	if err != nil {
		return err
	}
	for rows.Next() {
		var val int
		var count int
		rows.Scan(&val, &count)
		if val == 1 {
			fmt.Printf("   ➜ Detected %d flooded pixels.\n", count)
		}
	}
	rows.Close()

	_, err = pool.Exec(ctx, query, sceneID, acqTime)
	if err != nil {
		return err
	}

	// 5. Enrich with admin codes (includes Cleanup)
	err = EnrichGFMPolygons(ctx, pool, sceneID)
	if err != nil {
		return err
	}

	// 5.5 Populate Confidence Stats from Likelihood Raster
	likelihoodTable := fmt.Sprintf("\"temp_likelihood_raster_%s\"", sceneID)
	// Check if table exists
	var exists bool
	pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = $1)", "temp_likelihood_raster_"+sceneID).Scan(&exists)
	
	if exists {
		dateStr := ""
		pool.QueryRow(ctx, "SELECT acquisition_time::date::text FROM gfm_scene WHERE id = $1", sceneID).Scan(&dateStr)
		fmt.Printf("   ➜ [%s] Calculating confidence metrics...\n", dateStr)
		updateQuery := fmt.Sprintf(`
			UPDATE gfm_flood_polygon f
			SET 
				confidence_mean = (s.stats).mean,
				confidence_min = (s.stats).min,
				confidence_max = (s.stats).max,
				pixel_count = (s.stats).count::int
			FROM (
				SELECT f.id, ST_SummaryStats(ST_Clip(l.rast, f.geom)) as stats
				FROM %s l
				JOIN gfm_flood_polygon f ON ST_Intersects(l.rast, f.geom)
				WHERE f.scene_id = $1
			) s
			WHERE f.id = s.id
		`, likelihoodTable)
		_, err = pool.Exec(ctx, updateQuery, sceneID)
		if err != nil {
			fmt.Printf("Warning: failed to populate confidence stats: %v\n", err)
		}
	}

	// 6. Aggregate stats
	return AggregateGFMStats(ctx, pool, sceneID)
}

// EnrichGFMPolygons updates flood polygons with administrative codes using spatial join
func EnrichGFMPolygons(ctx context.Context, pool *pgxpool.Pool, sceneID string) error {
	// 1. Update Province Code (length 2)
	// We use ST_DWithin with a small tolerance (0.001 degrees ~ 110m) to catch edge cases
	_, err := pool.Exec(ctx, `
		UPDATE gfm_flood_polygon f
		SET admin_province_code = (
			SELECT r.kode 
			FROM regions r 
			WHERE ST_DWithin(f.centroid, r.geom, 0.00135) 
			  AND LENGTH(r.kode) = 2
			ORDER BY ST_Distance(f.centroid, r.geom) ASC
			LIMIT 1
		)
		WHERE f.scene_id = $1
		  AND f.admin_province_code IS NULL
	`, sceneID)
	if err != nil {
		return fmt.Errorf("failed to enrich province: %v", err)
	}

	// 2. Update City Code (length 5)
	_, err = pool.Exec(ctx, `
		UPDATE gfm_flood_polygon f
		SET admin_city_code = (
			SELECT r.kode 
			FROM regions r 
			WHERE ST_DWithin(f.centroid, r.geom, 0.00135) 
			  AND LENGTH(r.kode) = 5
			ORDER BY ST_Distance(f.centroid, r.geom) ASC
			LIMIT 1
		)
		WHERE f.scene_id = $1
		  AND f.admin_city_code IS NULL
	`, sceneID)
	if err != nil {
		return fmt.Errorf("failed to enrich city: %v", err)
	}

	// 2.1 Update District Code (length 8)
	_, err = pool.Exec(ctx, `
		UPDATE gfm_flood_polygon f
		SET admin_district_code = (
			SELECT r.kode 
			FROM regions r 
			WHERE ST_DWithin(f.centroid, r.geom, 0.00135) 
			  AND LENGTH(r.kode) = 8
			ORDER BY ST_Distance(f.centroid, r.geom) ASC
			LIMIT 1
		)
		WHERE f.scene_id = $1
		  AND f.admin_district_code IS NULL
	`, sceneID)
	if err != nil {
		fmt.Printf("Warning: failed to enrich district: %v\n", err)
	}

	// 2.2 Update Village Code (length 13)
	_, err = pool.Exec(ctx, `
		UPDATE gfm_flood_polygon f
		SET admin_village_code = (
			SELECT r.kode 
			FROM regions r 
			WHERE ST_DWithin(f.centroid, r.geom, 0.00135) 
			  AND LENGTH(r.kode) = 13
			ORDER BY ST_Distance(f.centroid, r.geom) ASC
			LIMIT 1
		)
		WHERE f.scene_id = $1
		  AND f.admin_village_code IS NULL
	`, sceneID)
	if err != nil {
		fmt.Printf("Warning: failed to enrich village: %v\n", err)
	}

	// 2.3 Hierarchical Backfill: If we have a child, we must have its parents
	_, _ = pool.Exec(ctx, `
		-- Backfill District from Village
		UPDATE gfm_flood_polygon 
		SET admin_district_code = LEFT(admin_village_code, 8)
		WHERE scene_id = $1 AND admin_district_code IS NULL AND admin_village_code IS NOT NULL;
		
		-- Backfill City from District
		UPDATE gfm_flood_polygon 
		SET admin_city_code = LEFT(admin_district_code, 5)
		WHERE scene_id = $1 AND admin_city_code IS NULL AND admin_district_code IS NOT NULL;
		
		-- Backfill Province from City
		UPDATE gfm_flood_polygon 
		SET admin_province_code = LEFT(admin_city_code, 2)
		WHERE scene_id = $1 AND admin_province_code IS NULL AND admin_city_code IS NOT NULL;
	`, sceneID)

	// 3. Cleanup: Delete polygons that are still NULL (outside business coverage)
	_, err = pool.Exec(ctx, "DELETE FROM gfm_flood_polygon WHERE scene_id = $1 AND (admin_province_code IS NULL OR admin_city_code IS NULL)", sceneID)
	if err != nil {
		fmt.Printf("Warning: failed to cleanup NULL polygons: %v\n", err)
	}

	return nil
}

// AggregateGFMStats calculates flood area per administrative region (Prov, City, Dist, Vill)
func AggregateGFMStats(ctx context.Context, pool *pgxpool.Pool, sceneID string) error {
	levels := []struct {
		Level string
		Col   string
	}{
		{"province", "admin_province_code"},
		{"city", "admin_city_code"},
		{"district", "admin_district_code"},
		{"village", "admin_village_code"},
	}

	for _, l := range levels {
		fmt.Printf("   ➜ Aggregating stats for level: %s...\n", l.Level)
		query := fmt.Sprintf(`
			INSERT INTO gfm_admin_daily_summary (
				source, admin_level, admin_code, admin_name, date, 
				flood_polygon_count, total_flood_area_m2, max_flood_area_m2, admin_area_m2, flood_percentage,
				first_detected_at, last_detected_at
			)
			SELECT 
				'copernicus_gfm',
				$1,
				f.%s,
				r.name,
				(f.acquisition_time AT TIME ZONE 'Asia/Jakarta')::date,
				COUNT(*),
				SUM(f.area_m2),
				MAX(f.area_m2),
				ST_Area(r.geom::geography),
				SUM(f.area_m2) / NULLIF(ST_Area(r.geom::geography), 0) * 100,
				MIN(f.acquisition_time),
				MAX(f.acquisition_time)
			FROM gfm_flood_polygon f
			JOIN regions r ON f.%s = r.kode
			WHERE f.scene_id = $2 AND f.%s IS NOT NULL
			GROUP BY f.%s, r.name, r.geom, (f.acquisition_time AT TIME ZONE 'Asia/Jakarta')::date
			ON CONFLICT (source, admin_level, admin_code, date) DO UPDATE
			SET 
				flood_polygon_count = EXCLUDED.flood_polygon_count,
				total_flood_area_m2 = EXCLUDED.total_flood_area_m2,
				max_flood_area_m2 = EXCLUDED.max_flood_area_m2,
				admin_area_m2 = EXCLUDED.admin_area_m2,
				flood_percentage = EXCLUDED.flood_percentage,
				first_detected_at = EXCLUDED.first_detected_at,
				last_detected_at = EXCLUDED.last_detected_at,
				admin_name = EXCLUDED.admin_name,
				updated_at = now()
		`, l.Col, l.Col, l.Col, l.Col)

		_, err := pool.Exec(ctx, query, l.Level, sceneID)
		if err != nil {
			fmt.Printf("Warning: failed to aggregate stats for %s: %v\n", l.Level, err)
		}
	}

	// 7. Update Historical Risk Scores for all regions
	fmt.Printf("   ➜ Updating historical risk scores...\n")
	return UpdateHistoricalRiskScores(ctx, pool)
}

// UpdateHistoricalRiskScores calculates long-term risk metrics based on all ingested data
func UpdateHistoricalRiskScores(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		INSERT INTO gfm_admin_risk_score (admin_level, admin_code, admin_name, total_detections, flood_occurrence_count, risk_score, last_updated_at)
		SELECT 
			admin_level,
			admin_code,
			MAX(admin_name),
			COUNT(*), 
			COUNT(*) FILTER (WHERE flood_percentage > 0), 
			AVG(flood_percentage), 
			NOW()
		FROM gfm_admin_daily_summary
		GROUP BY admin_level, admin_code
		ON CONFLICT (admin_level, admin_code) DO UPDATE
		SET 
			total_detections = EXCLUDED.total_detections,
			flood_occurrence_count = EXCLUDED.flood_occurrence_count,
			risk_score = EXCLUDED.risk_score,
			last_updated_at = NOW()
	`
	_, err := pool.Exec(ctx, query)
	return err
}

