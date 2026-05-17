package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"proferti-be/internal/repo"
)

// POST /api/leads — public endpoint, any visitor can submit a lead
func (a *API) SubmitLead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string   `json:"projectId"`
		Name      string   `json:"name"`
		Phone     string   `json:"phone"`
		Email     string   `json:"email"`
		Message   string   `json:"message"`
		Budget    *float64 `json:"budget,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil {
		errJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ProjectID == "" || body.Name == "" {
		errJSON(w, http.StatusBadRequest, "projectId and name are required")
		return
	}

	lead, err := repo.CreateLead(
		r.Context(), a.Pool,
		body.ProjectID, body.Name,
		nullableStr(body.Phone), nullableStr(body.Email), nullableStr(body.Message),
		body.Budget,
	)
	if err != nil {
		if strings.Contains(err.Error(), "no rows") {
			errJSON(w, http.StatusNotFound, "proyek tidak ditemukan")
			return
		}
		errJSON(w, http.StatusInternalServerError, "server error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"ok": true, "lead": lead})
}

// GET /api/leads?userId={id}&limit={n}&skip={n}
func (a *API) ListMyLeads(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		errJSON(w, http.StatusBadRequest, "userId is required")
		return
	}
	dev, err := repo.GetDeveloperByUserID(r.Context(), a.Pool, userID)
	if err != nil {
		errJSON(w, http.StatusForbidden, "bukan developer terdaftar")
		return
	}
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 50)
	skip := parsePositiveInt(r.URL.Query().Get("skip"), 0)

	leads, total, err := repo.ListLeadsByDeveloper(r.Context(), a.Pool, dev.ID, limit, skip)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"leads": leads,
		"pagination": map[string]int64{
			"total": total,
			"limit": int64(limit),
			"skip":  int64(skip),
		},
	})
}

// PATCH /api/leads/{id}/status
func (a *API) PatchLeadStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		UserID string `json:"userId"`
		Status string `json:"status"`
	}
	if err := decodeBody(r, &body); err != nil || body.Status == "" || body.UserID == "" {
		errJSON(w, http.StatusBadRequest, "userId and status are required")
		return
	}
	validStatuses := map[string]bool{"new": true, "contacted": true, "qualified": true, "closed": true}
	if !validStatuses[body.Status] {
		errJSON(w, http.StatusBadRequest, "invalid status value")
		return
	}
	dev, err := repo.GetDeveloperByUserID(r.Context(), a.Pool, body.UserID)
	if err != nil {
		errJSON(w, http.StatusForbidden, "bukan developer terdaftar")
		return
	}
	if err := repo.UpdateLeadStatus(r.Context(), a.Pool, id, dev.ID, body.Status); err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// GET /api/leads/{id}/notes
func (a *API) ListLeadNotes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	notes, err := repo.ListLeadNotes(r.Context(), a.Pool, id)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, "query failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": notes})
}

// POST /api/leads/{id}/notes
func (a *API) AddLeadNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		UserID         string     `json:"userId"`
		Note           string     `json:"note"`
		NextFollowupAt *time.Time `json:"nextFollowupAt,omitempty"`
	}
	if err := decodeBody(r, &body); err != nil || body.Note == "" || body.UserID == "" {
		errJSON(w, http.StatusBadRequest, "userId and note are required")
		return
	}

	note, err := repo.AddLeadNote(r.Context(), a.Pool, id, body.UserID, body.Note, body.NextFollowupAt)
	if err != nil {
		errJSON(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"note": note})
}

