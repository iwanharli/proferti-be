package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"proferti-be/internal/repo"
)

// POST /api/projects
func (a *API) CreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DeveloperID string   `json:"developerId"`
		Name        string   `json:"name"`
		LocationID  string   `json:"locationId"`
		Description string   `json:"description"`
		CoverImage  string   `json:"coverImage"`
		Promo       string   `json:"promo"`
		StartPrice  float64  `json:"startPrice"`
		Status      string   `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.DeveloperID == "" || body.Name == "" || body.LocationID == "" {
		errJSON(w, http.StatusBadRequest, "developerId, name, and locationId are required")
		return
	}
	slug := fmt.Sprintf("%s-%d", toSlug(body.Name), time.Now().Year())
	proj, err := repo.CreateProject(
		r.Context(), a.Pool,
		body.DeveloperID, body.Name, slug, body.LocationID,
		nullableStr(body.Description), nullableStr(body.CoverImage), nullableStr(body.Promo),
		body.StartPrice, body.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			errJSON(w, http.StatusConflict, "nama proyek sudah digunakan pada developer ini")
			return
		}
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"project": proj})
}

// PUT /api/projects/{id}
func (a *API) UpdateProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		DeveloperID string  `json:"developerId"`
		Name        string  `json:"name"`
		LocationID  string  `json:"locationId"`
		Description string  `json:"description"`
		CoverImage  string  `json:"coverImage"`
		Promo       string  `json:"promo"`
		StartPrice  float64 `json:"startPrice"`
		Status      string  `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	slug := fmt.Sprintf("%s-%d", toSlug(body.Name), time.Now().Year())
	err := repo.UpdateProject(
		r.Context(), a.Pool,
		id, body.DeveloperID, body.Name, slug, body.LocationID,
		nullableStr(body.Description), nullableStr(body.CoverImage), nullableStr(body.Promo),
		body.StartPrice, body.Status,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			errJSON(w, http.StatusNotFound, err.Error())
			return
		}
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}


// DELETE /api/projects/{id}
func (a *API) DeleteProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		DeveloperID string `json:"developerId"`
	}
	_ = decodeBody(r, &body)
	if body.DeveloperID == "" {
		body.DeveloperID = r.URL.Query().Get("developerId")
	}
	if err := repo.DeleteProject(r.Context(), a.Pool, id, body.DeveloperID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			errJSON(w, http.StatusNotFound, err.Error())
			return
		}
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/projects/{id}/unit-types
func (a *API) ListUnitTypes(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	units, err := repo.ListUnitTypes(r.Context(), a.Pool, projectID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unitTypes": units})
}

// POST /api/projects/{id}/unit-types
func (a *API) CreateUnitType(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var body struct {
		TypeName     string  `json:"typeName"`
		LandSize     string  `json:"landSize"`
		BuildingSize string  `json:"buildingSize"`
		Bedroom      *int16  `json:"bedroom"`
		Bathroom     *int16  `json:"bathroom"`
		Garage       *int16  `json:"garage"`
		Price        float64 `json:"price"`
		Stock        int     `json:"stock"`
	}
	if err := decodeBody(r, &body); err != nil || body.TypeName == "" {
		errJSON(w, http.StatusBadRequest, "typeName is required")
		return
	}
	slug := toSlug(body.TypeName)
	u, err := repo.CreateUnitType(
		r.Context(), a.Pool, projectID, body.TypeName, slug,
		nullableStr(body.LandSize), nullableStr(body.BuildingSize),
		body.Bedroom, body.Bathroom, body.Garage,
		body.Price, body.Stock,
	)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"unitType": u})
}

// PUT /api/unit-types/{id}
func (a *API) UpdateUnitType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		ProjectID    string  `json:"projectId"`
		TypeName     string  `json:"typeName"`
		LandSize     string  `json:"landSize"`
		BuildingSize string  `json:"buildingSize"`
		Bedroom      *int16  `json:"bedroom"`
		Bathroom     *int16  `json:"bathroom"`
		Garage       *int16  `json:"garage"`
		Price        float64 `json:"price"`
		Stock        int     `json:"stock"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid body")
		return
	}
	slug := toSlug(body.TypeName)
	err := repo.UpdateUnitType(
		r.Context(), a.Pool, id, body.ProjectID, body.TypeName, slug,
		nullableStr(body.LandSize), nullableStr(body.BuildingSize),
		body.Bedroom, body.Bathroom, body.Garage,
		body.Price, body.Stock,
	)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// DELETE /api/unit-types/{id}
func (a *API) DeleteUnitType(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	projectID := r.URL.Query().Get("projectId")
	if err := repo.DeleteUnitType(r.Context(), a.Pool, id, projectID); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// POST /api/projects/{id}/gallery
func (a *API) AddGallery(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var body struct {
		URL   string  `json:"url"`
		Title *string `json:"title,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil || body.URL == "" {
		errJSON(w, http.StatusBadRequest, "url is required")
		return
	}
	g, err := repo.AddGalleryImage(r.Context(), a.Pool, projectID, body.URL, body.Title)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"gallery": g})
}

// DELETE /api/gallery/{id}
func (a *API) DeleteGallery(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	projectID := r.URL.Query().Get("projectId")
	if err := repo.DeleteGalleryImage(r.Context(), a.Pool, id, projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "not found") {
			errJSON(w, http.StatusNotFound, err.Error())
			return
		}
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// nullableStr converts empty string to nil pointer.
func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
