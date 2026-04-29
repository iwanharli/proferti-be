package handlers

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"proferti-be/internal/repo"
)

type API struct {
	Pool *pgxpool.Pool
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

// GET /api/projects — kompatibel dengan query Nitro `/api/projects`
func (a *API) ListProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := parsePositiveInt(q.Get("limit"), 20)
	if limit > 100 {
		limit = 100
	}
	skip := parsePositiveInt(q.Get("skip"), 0)

	f := repo.ProjectListFilters{
		City:        q.Get("city"),
		DeveloperID: q.Get("developerId"),
		Status:      q.Get("status"),
		Search:      q.Get("q"),
		Limit:       limit,
		Skip:        skip,
	}
	f.MinPrice = parseFloat(q.Get("minPrice"))
	f.MaxPrice = parseFloat(q.Get("maxPrice"))

	projects, total, err := repo.ListProjects(r.Context(), a.Pool, f)
	if err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"projects": projects,
		"pagination": map[string]int64{
			"total": total,
			"limit": int64(limit),
			"skip":  int64(skip),
		},
	})
}

// GET /api/projects/{id}
func (a *API) GetProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, `{"message":"ID proyek diperlukan"}`, http.StatusBadRequest)
		return
	}
	p, err := repo.GetProjectByID(r.Context(), a.Pool, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, `{"message":"Proyek tidak ditemukan"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"project": p})
}

type developerRow struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Logo *string `json:"logo,omitempty"`
}

// GET /api/developers
func (a *API) ListDevelopers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Pool.Query(r.Context(), `
SELECT id, name, logo FROM "Developer" ORDER BY name ASC LIMIT 200`)
	if err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []developerRow
	for rows.Next() {
		var d developerRow
		if err := rows.Scan(&d.ID, &d.Name, &d.Logo); err != nil {
			http.Error(w, `{"error":"scan failed"}`, http.StatusInternalServerError)
			return
		}
		list = append(list, d)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"developers": list})
}
