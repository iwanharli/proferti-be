package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"proferti-be/internal/repo"
)

// GET /api/projects/{id}/units
func (a *API) ListProjectUnits(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	units, err := repo.ListUnitsByProject(r.Context(), a.Pool, projectID)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"units": units})
}

// POST /api/projects/{id}/units
func (a *API) CreateUnit(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")
	var body struct {
		UnitTypeID string  `json:"unitTypeId"`
		Block      string  `json:"block"`
		Number     string  `json:"number"`
		Facing     string  `json:"facing"`
		Price      float64 `json:"price"`
		Status     string  `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil || body.UnitTypeID == "" {
		errJSON(w, http.StatusBadRequest, "unitTypeId is required")
		return
	}

	u, err := repo.CreateUnit(
		r.Context(), a.Pool, projectID, body.UnitTypeID,
		nullableStr(body.Block), nullableStr(body.Number), nullableStr(body.Facing),
		body.Price, body.Status,
	)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"unit": u})
}

// PATCH /api/units/{id}/status
func (a *API) PatchUnitStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status string `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil || body.Status == "" {
		errJSON(w, http.StatusBadRequest, "status is required")
		return
	}

	if err := repo.UpdateUnitStatus(r.Context(), a.Pool, id, body.Status); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
