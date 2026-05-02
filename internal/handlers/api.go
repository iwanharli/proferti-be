package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"proferti-be/internal/repo"
	"proferti-be/internal/worker"
)

type API struct {
	Pool *pgxpool.Pool
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func errJSON(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeBody(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func parsePositiveInt(qs string, def int) int {
	if qs == "" {
		return def
	}
	n, err := strconv.Atoi(qs)
	if err != nil || n < 0 || n > math.MaxInt32 {
		return def
	}
	return n
}

func parseFloat(qs string) *float64 {
	if qs == "" {
		return nil
	}
	f, err := strconv.ParseFloat(qs, 64)
	if err != nil {
		return nil
	}
	return &f
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func toSlug(s string) string {
	slug := slugRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "developer"
	}
	return slug
}

// ─── Auth endpoints ───────────────────────────────────────────────────────────

// POST /api/auth/login — validate email/password, return user data (no session created here)
func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeBody(r, &body); err != nil || body.Email == "" || body.Password == "" {
		errJSON(w, http.StatusBadRequest, "email and password are required")
		return
	}

	user, err := repo.GetUserByEmail(r.Context(), a.Pool, body.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errJSON(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}
	if user.Password == nil {
		errJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(body.Password)); err != nil {
		errJSON(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.RoleUpper(),
		},
	})
}

// POST /api/auth/register — create a new user with email/password
func (a *API) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" || body.Password == "" || body.Name == "" {
		errJSON(w, http.StatusBadRequest, "name, email, and password are required")
		return
	}
	if len(body.Password) < 8 {
		errJSON(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}

	user, err := repo.CreateUser(r.Context(), a.Pool, body.Name, body.Email, string(hashed))
	if err != nil {
		// Unique constraint on email
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			errJSON(w, http.StatusConflict, "email already registered")
			return
		}
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.RoleUpper(),
		},
	})
}

// POST /api/auth/oauth-sync — upsert a user from OAuth provider (GitHub etc.)
// Called from Nuxt server-side JWT callback; not directly from browser.
func (a *API) OAuthSync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email string  `json:"email"`
		Name  string  `json:"name"`
		Image *string `json:"image,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil || body.Email == "" {
		errJSON(w, http.StatusBadRequest, "email is required")
		return
	}
	if body.Name == "" {
		body.Name = body.Email
	}

	user, err := repo.UpsertUserByEmail(r.Context(), a.Pool, body.Email, body.Name, body.Image)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.RoleUpper(),
		},
	})
}

// ─── Developer endpoints ──────────────────────────────────────────────────────

// POST /api/developers/register — create developer profile for authenticated user
func (a *API) RegisterDeveloper(w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID      string  `json:"userId"`
		Name        string  `json:"name"`
		Description *string `json:"description,omitempty"`
		Website     *string `json:"website,omitempty"`
		Logo        *string `json:"logo,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.UserID == "" || body.Name == "" {
		errJSON(w, http.StatusBadRequest, "userId and name are required")
		return
	}

	slug := toSlug(body.Name)

	dev, err := repo.CreateDeveloperForUser(
		r.Context(), a.Pool,
		body.UserID, body.Name, slug,
		body.Description, body.Website, body.Logo,
	)
	if err != nil {
		if errors.Is(err, repo.ErrAlreadyDeveloper) {
			// Jika sudah terdaftar, lakukan UPDATE
			existingDev, err2 := repo.GetDeveloperByUserID(r.Context(), a.Pool, body.UserID)
			if err2 == nil {
				err3 := repo.UpdateDeveloper(
					r.Context(), a.Pool,
					existingDev.ID, body.Name, slug,
					body.Description, body.Website, body.Logo,
				)
				if err3 != nil {
					errJSON(w, http.StatusInternalServerError, "failed to update developer")
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{
					"ok":      true,
					"message": "profil developer berhasil diperbarui",
				})
				return
			}
			errJSON(w, http.StatusConflict, "akun ini sudah terdaftar")
			return
		}
		if strings.Contains(err.Error(), "user not found") {
			errJSON(w, http.StatusNotFound, "user not found")
			return
		}
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			errJSON(w, http.StatusConflict, "nama developer sudah digunakan, coba nama lain")
			return
		}
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":        true,
		"message":   "profil developer berhasil dibuat",
		"developer": dev,
	})
}

// GET /api/developers/me?userId={uuid} — fetch developer profile for a given user
func (a *API) GetMyDeveloper(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		errJSON(w, http.StatusBadRequest, "userId query param is required")
		return
	}

	dev, err := repo.GetDeveloperByUserID(r.Context(), a.Pool, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errJSON(w, http.StatusNotFound, "developer profile not found")
			return
		}
		log.Printf("Error GetDeveloperByUserID: %v", err)
		errJSON(w, http.StatusInternalServerError, "server error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"developer": dev})
}

// ─── Project endpoints (existing) ────────────────────────────────────────────

func parseInt(qs string) *int {
	if qs == "" {
		return nil
	}
	n, err := strconv.Atoi(qs)
	if err != nil {
		return nil
	}
	return &n
}

// GET /api/projects
func (a *API) ListProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := parsePositiveInt(q.Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	skip := parsePositiveInt(q.Get("skip"), 0)

	f := repo.ProjectListFilters{
		City:         q.Get("city"),
		DeveloperID:  q.Get("developerId"),
		DeveloperIDs: q["developerIds"], // pgx handles []string with ANY($n)
		Status:       q.Get("status"),
		Search:       q.Get("q"),
		Type:         q.Get("type"),
		Sort:         q.Get("sort"),
		Bedrooms:     parseInt(q.Get("bedrooms")),
		Bathrooms:    parseInt(q.Get("bathrooms")),
		Limit:        limit,
		Skip:         skip,
	}
	f.MinPrice = parseFloat(q.Get("minPrice"))
	f.MaxPrice = parseFloat(q.Get("maxPrice"))

	projects, total, err := repo.ListProjects(r.Context(), a.Pool, f)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
		"pagination": map[string]int64{
			"total": total,
			"limit": int64(limit),
			"skip":  int64(skip),
		},
	})
}

// GET /api/projects/{idOrSlug}
func (a *API) GetProject(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if idOrSlug == "" {
		errJSON(w, http.StatusBadRequest, "ID atau Slug proyek diperlukan")
		return
	}

	// Try slug first
	p, err := repo.GetProjectBySlug(r.Context(), a.Pool, idOrSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// If slug fails, try ID (UUID)
			p, err = repo.GetProjectByID(r.Context(), a.Pool, idOrSlug)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					errJSON(w, http.StatusNotFound, "proyek tidak ditemukan")
					return
				}
				errJSON(w, http.StatusInternalServerError, "query failed")
				return
			}
		} else {
			errJSON(w, http.StatusInternalServerError, "query failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": p})
}

// GET /api/unit-types/{idOrSlug}
func (a *API) GetUnitType(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if idOrSlug == "" {
		errJSON(w, http.StatusBadRequest, "ID atau Slug tipe unit diperlukan")
		return
	}

	// Try slug first
	u, err := repo.GetUnitTypeBySlug(r.Context(), a.Pool, idOrSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// If slug fails, try ID (UUID)
			u, err = repo.GetUnitTypeByID(r.Context(), a.Pool, idOrSlug)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					errJSON(w, http.StatusNotFound, "tipe unit tidak ditemukan")
					return
				}
				errJSON(w, http.StatusInternalServerError, "query failed")
				return
			}
		} else {
			errJSON(w, http.StatusInternalServerError, "query failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"unitType": u})
}



// ─── Developer list endpoint (existing) ──────────────────────────────────────

type developerRow struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Logo         *string `json:"logo,omitempty"`
	Description  *string `json:"description,omitempty"`
	ProjectCount int     `json:"projectCount"`
}

// GET /api/developers
func (a *API) ListDevelopers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Pool.Query(r.Context(),
		`SELECT d.id, d.company_name AS name, d.slug, d.logo, d.description, 
		        (SELECT COUNT(*) FROM t_projects p WHERE p.developer_id = d.id) AS project_count
		 FROM t_developers d 
		 ORDER BY project_count DESC, d.company_name ASC 
		 LIMIT 200`)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed: "+err.Error())
		return
	}
	defer rows.Close()

	var list []developerRow
	for rows.Next() {
		var d developerRow
		if err := rows.Scan(&d.ID, &d.Name, &d.Slug, &d.Logo, &d.Description, &d.ProjectCount); err != nil {
			errJSON(w, http.StatusInternalServerError, "scan failed: "+err.Error())
			return
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"developers": list})
}

// GET /api/developers/{id}
func (a *API) GetDeveloper(w http.ResponseWriter, r *http.Request) {
	idOrSlug := chi.URLParam(r, "id")
	if idOrSlug == "" {
		errJSON(w, http.StatusBadRequest, "ID developer diperlukan")
		return
	}
	
	// Try lookup by slug first
	dev, err := repo.GetDeveloperBySlug(r.Context(), a.Pool, idOrSlug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Try lookup by ID
			dev, err = repo.GetDeveloperByID(r.Context(), a.Pool, idOrSlug)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					errJSON(w, http.StatusNotFound, "developer tidak ditemukan")
					return
				}
				errJSON(w, http.StatusInternalServerError, "query ID failed: "+err.Error())
				return
			}
		} else {
			errJSON(w, http.StatusInternalServerError, "query slug failed: "+err.Error())
			return
		}
	}
	
	writeJSON(w, http.StatusOK, map[string]any{"developer": dev})
}

// GET /api/locations
func (a *API) ListLocations(w http.ResponseWriter, r *http.Request) {
	locs, err := repo.ListLocations(r.Context(), a.Pool)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"locations": locs})
}

// GET /api/projects/meta
func (a *API) GetProjectsMeta(w http.ResponseWriter, r *http.Request) {
	meta, err := repo.GetProjectsMeta(r.Context(), a.Pool)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

// GET /api/regions/geojson
func (a *API) GetRegionsGeoJSON(w http.ResponseWriter, r *http.Request) {
	parent := r.URL.Query().Get("parent")
	data, err := repo.GetRegionsGeoJSON(r.Context(), a.Pool, parent)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// GET /api/regions/detect?lat=...&lng=...
func (a *API) DetectRegion(w http.ResponseWriter, r *http.Request) {
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")

	lat, _ := strconv.ParseFloat(latStr, 64)
	lng, _ := strconv.ParseFloat(lngStr, 64)

	if lat == 0 || lng == 0 {
		errJSON(w, http.StatusBadRequest, "invalid coordinates")
		return
	}

	name, err := repo.GetRegionByPoint(r.Context(), a.Pool, lat, lng)
	if err != nil {
		errJSON(w, http.StatusNotFound, "region not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

// GET /api/gfm/scenes
func (a *API) ListGFMScenes(w http.ResponseWriter, r *http.Request) {
	scenes, err := repo.GetGFMScenes(r.Context(), a.Pool, 20)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "failed to list scenes: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scenes)
}


// POST /api/gfm/ingest
func (a *API) TriggerGFMIngestion(w http.ResponseWriter, r *http.Request) {
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	go func() {
		err := worker.RunFullIngestionCycle(context.Background(), a.Pool, start, end)
		if err != nil {
			fmt.Printf("Manual ingestion cycle failed (start=%s, end=%s): %v\n", start, end, err)
		}
	}()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ingestion cycle started in background",
	})
}

// GET /api/flood-mvt/{z}/{x}/{y}.pbf
func (a *API) GetFloodMVT(w http.ResponseWriter, r *http.Request) {
	// 1. Get Z, X, Y from path
	// Assuming URL pattern /api/flood-mvt/{z}/{x}/{y}.pbf
	// For simplicity in this demo, we'll parse from URL path manually
	z, _ := strconv.Atoi(chi.URLParam(r, "z"))
	x, _ := strconv.Atoi(chi.URLParam(r, "x"))
	yStr := chi.URLParam(r, "y")
	yStr = strings.Replace(yStr, ".pbf", "", 1)
	y, _ := strconv.Atoi(yStr)

	// 1. Get query parameters for date filtering
	startDate := r.URL.Query().Get("start") // format: YYYY-MM-DD
	endDate := r.URL.Query().Get("end")     // format: YYYY-MM-DD

	// 2. Generate Multi-layer MVT using PostGIS
	query := `
		WITH flood_mvt_geom AS (
			SELECT 
				id,
				acquisition_time,
				area_m2,
				ST_AsMVTGeom(ST_Transform(geom, 3857), ST_TileEnvelope($1, $2, $3), 4096, 256, true) AS geom
			FROM gfm_flood_polygon
			WHERE geom && ST_Transform(ST_TileEnvelope($1, $2, $3), 4326)
			AND ($4 = '' OR acquisition_time >= NULLIF($4, '')::timestamp)
			AND ($5 = '' OR acquisition_time <= NULLIF($5, '')::timestamp + interval '1 day')
		),
		exclusion_mvt_geom AS (
			SELECT 
				id,
				exclusion_type,
				ST_AsMVTGeom(ST_Transform(geom, 3857), ST_TileEnvelope($1, $2, $3), 4096, 256, true) AS geom
			FROM gfm_exclusion_polygon
			WHERE geom && ST_Transform(ST_TileEnvelope($1, $2, $3), 4326)
		),
		mvt_flood AS (
			SELECT ST_AsMVT(flood_mvt_geom.*, 'flood_layer') as tile FROM flood_mvt_geom
		),
		mvt_exclusion AS (
			SELECT ST_AsMVT(exclusion_mvt_geom.*, 'exclusion_layer') as tile FROM exclusion_mvt_geom
		)
		SELECT COALESCE((SELECT tile FROM mvt_flood), ''::bytea) || COALESCE((SELECT tile FROM mvt_exclusion), ''::bytea);
	`

	var mvt []byte
	err := a.Pool.QueryRow(r.Context(), query, z, x, y, startDate, endDate).Scan(&mvt)
	if (err != nil) {
		log.Printf("❌ MVT Error (z=%d, x=%d, y=%d): %v", z, x, y, err)
		errJSON(w, http.StatusInternalServerError, "failed to generate mvt: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/vnd.mapbox-vector-tile")
	w.Write(mvt)
}
func (a *API) GetGFMScenesGeoJSON(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT jsonb_build_object(
			'type', 'FeatureCollection',
			'features', jsonb_agg(features.feature)
		)
		FROM (
			SELECT jsonb_build_object(
				'type', 'Feature',
				'geometry', ST_AsGeoJSON(COALESCE(footprint, bbox))::jsonb,
				'properties', jsonb_build_object(
					'stac_item_id', stac_item_id,
					'platform', platform,
					'orbit_direction', orbit_direction,
					'acquisition_time', acquisition_time,
					'date', (acquisition_time AT TIME ZONE 'Asia/Jakarta')::date
				)
			) AS feature
			FROM gfm_scene
			WHERE bbox IS NOT NULL
			ORDER BY acquisition_time DESC
		) features;
	`

	var geojson []byte
	err := a.Pool.QueryRow(r.Context(), query).Scan(&geojson)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "failed to generate scenes geojson: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(geojson)
}

// GET /api/gfm/summary?lat=...&lng=...
func (a *API) GetGFMSummary(w http.ResponseWriter, r *http.Request) {
	lat := r.URL.Query().Get("lat")
	lng := r.URL.Query().Get("lng")

	if lat == "" || lng == "" {
		errJSON(w, http.StatusBadRequest, "missing lat/lng parameters")
		return
	}

	// Query ALL scenes in the past 7 days covering this point, not just the latest one.
	// This prevents false-negatives when the latest scene happens to not detect flooding
	// but a scene 2 days ago DID detect it.
	query := `
		WITH covering_scenes AS (
			SELECT id, platform, acquisition_time
			FROM gfm_scene
			WHERE ST_Intersects(footprint, ST_SetSRID(ST_Point($1, $2), 4326))
			  AND acquisition_time >= NOW() - INTERVAL '7 days'
			ORDER BY acquisition_time DESC
		),
		latest_scene AS (
			SELECT * FROM covering_scenes LIMIT 1
		),
		nearby_flood AS (
			SELECT 
				SUM(fp.area_m2) as total_area,
				AVG(fp.confidence_mean) as avg_confidence,
				MAX(fp.admin_city_code) as region
			FROM gfm_flood_polygon fp
			WHERE fp.scene_id IN (SELECT id FROM covering_scenes)
			  AND ST_DWithin(fp.geom, ST_SetSRID(ST_Point($1, $2), 4326), 0.05)
		)
		SELECT 
			COALESCE(ls.platform, 'N/A') as platform,
			COALESCE(ls.acquisition_time, NOW()) as latest_time,
			COALESCE(nf.total_area, 0) as total_area,
			COALESCE(nf.avg_confidence, 0) as avg_confidence,
			COALESCE(nf.region, 'Unknown Region') as region
		FROM latest_scene ls
		LEFT JOIN nearby_flood nf ON true
	`
	var platform string
	var latestTime time.Time
	var totalArea float64
	var avgConfidence float64
	var region string

	err := a.Pool.QueryRow(r.Context(), query, lng, lat).Scan(&platform, &latestTime, &totalArea, &avgConfidence, &region)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "no_data"})
		return
	}

	status := "no_flood"
	if totalArea > 0 {
		status = "flood_detected"
	}

	// Confidence from DB is stored in 0-100 range (mean of likelihood raster).
	// Clamp to 0-100 to prevent display issues.
	confidenceVal := avgConfidence
	if confidenceVal < 0 {
		confidenceVal = 0
	} else if confidenceVal > 100 {
		confidenceVal = 100
	}
	confidenceLabel := fmt.Sprintf("%.0f%%", confidenceVal)

	resp := map[string]any{
		"status":         status,
		"source":         "Copernicus GFM",
		"platform":       platform,
		"satellite_time": latestTime.Format("2 Jan 2006, 15:04 MST"),
		"raw_time":       latestTime,
		"region":         region,
		"area_ha":        totalArea / 10000,
		"confidence":     confidenceLabel,
		"type":           "Observed Flood Extent",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /api/gfm/risk-summary
func (a *API) GetGFMRiskSummary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start := q.Get("start")
	end := q.Get("end")

	var query string
	var args []any

	if start != "" && end != "" {
		// Dynamic aggregation based on date range
		query = `
			SELECT 
				admin_level, 
				admin_code, 
				admin_name, 
				SUM(flood_polygon_count)::int as total_detections,
				COUNT(DISTINCT date)::int as flood_occurrence_count,
				MAX(flood_percentage) as risk_score,
				MAX(last_detected_at) as last_updated_at,
				SUM(total_flood_area_m2) as total_area
			FROM gfm_admin_daily_summary
			WHERE date BETWEEN $1 AND $2
			GROUP BY admin_level, admin_code, admin_name
			ORDER BY total_area DESC
			LIMIT 100
		`
		args = append(args, start, end)
	} else {
		// Static snapshot
		query = `
			SELECT 
				admin_level, admin_code, admin_name, total_detections, flood_occurrence_count, risk_score, last_updated_at,
				(risk_score * 1000000) as total_area -- fallback for ordering
			FROM gfm_admin_risk_score
			ORDER BY risk_score DESC
			LIMIT 100
		`
	}

	rows, err := a.Pool.Query(r.Context(), query, args...)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "failed to get risk summary: "+err.Error())
		return
	}
	defer rows.Close()

	var risks []map[string]any
	for rows.Next() {
		var adminLevel, adminCode, adminName string
		var totalDetections, floodOccurrenceCount int
		var riskScore, totalArea float64
		var lastUpdatedAt time.Time
		err := rows.Scan(&adminLevel, &adminCode, &adminName, &totalDetections, &floodOccurrenceCount, &riskScore, &lastUpdatedAt, &totalArea)
		if err != nil {
			errJSON(w, http.StatusInternalServerError, "failed to scan risk summary: "+err.Error())
			return
		}
		risks = append(risks, map[string]any{
			"admin_level":            adminLevel,
			"admin_code":             adminCode,
			"admin_name":             adminName,
			"total_detections":       totalDetections,
			"flood_occurrence_count": floodOccurrenceCount,
			"risk_score":             riskScore,
			"last_updated_at":        lastUpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(risks)
}
