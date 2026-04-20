package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type LockerHandler struct {
	// Will inject services later
}

func NewLockerHandler() *LockerHandler {
	return &LockerHandler{}
}

// GET /api/v1/locations/{id}/lockers
func (h *LockerHandler) GetLocationLockers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid location id", http.StatusBadRequest)
		return
	}

	// TODO: call service
	lockers := []interface{}{}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lockers)
}

// GET /api/v1/lockers/{id}
func (h *LockerHandler) GetLocker(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	_, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "invalid locker id", http.StatusBadRequest)
		return
	}

	// TODO: call service

	w.Header().Set("Content-Type", "application/json")
}

// POST /api/v1/sessions
func (h *LockerHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LockerID int64  `json:"locker_id"`
		Phone   string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: call service
	_ = req

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}
