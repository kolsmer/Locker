package http

import (
	"encoding/json"
	"net/http"
)

type AdminHandler struct {
	// Will inject services later
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// POST /api/v1/admin/login
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: verify password
	// TODO: generate JWT
	_ = req

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": "jwt_token_here"})
}

// GET /api/v1/admin/revenue
func (h *AdminHandler) GetRevenue(w http.ResponseWriter, r *http.Request) {
	// TODO: sum all paid payments
	w.Header().Set("Content-Type", "application/json")
}

// POST /api/v1/admin/lockers/{id}/open
func (h *AdminHandler) ManualOpenLocker(w http.ResponseWriter, r *http.Request) {
	// TODO: create device command
	w.WriteHeader(http.StatusOK)
}

// GET /api/v1/admin/sessions
func (h *AdminHandler) GetAllSessions(w http.ResponseWriter, r *http.Request) {
	// TODO: list all sessions
	w.Header().Set("Content-Type", "application/json")
}
