package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"locker/internal/auth"
	"locker/internal/service"

	"github.com/gorilla/mux"
)

type AdminHandler struct {
	svc *service.AdminService
}

func NewAdminHandler(svc *service.AdminService) *AdminHandler {
	return &AdminHandler{svc: svc}
}

func (h *AdminHandler) handleError(w http.ResponseWriter, rid string, err error) {
	if appErr, ok := err.(*service.AppError); ok {
		writeErr(w, rid, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error", nil)
}

// POST /api/v1/admin/login
func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}
	data, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// GET /api/v1/admin/me
func (h *AdminHandler) Me(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, rid, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация", nil)
		return
	}
	data, err := h.svc.Me(r.Context(), claims.AdminID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// GET /api/v1/admin/locations
func (h *AdminHandler) ListLocations(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	data, total, err := h.svc.ListLocations(
		r.Context(),
		r.URL.Query().Get("search"),
		r.URL.Query().Get("isActive"),
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
	)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, map[string]interface{}{"total": total})
}

// GET /api/v1/admin/locations/{locationId}/lockers
func (h *AdminHandler) ListLocationLockers(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	locationID, err := strconv.ParseInt(mux.Vars(r)["locationId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCATION_NOT_FOUND", "Локация не найдена", nil)
		return
	}
	data, total, err := h.svc.ListLocationLockers(
		r.Context(),
		locationID,
		r.URL.Query().Get("status"),
		r.URL.Query().Get("size"),
		r.URL.Query().Get("isActive"),
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
	)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, map[string]interface{}{"total": total})
}

// GET /api/v1/admin/lockers/{lockerId}
func (h *AdminHandler) GetLocker(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Ячейка не найдена", nil)
		return
	}
	data, err := h.svc.LockerDetail(r.Context(), lockerID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// PATCH /api/v1/admin/lockers/{lockerId}/status
func (h *AdminHandler) PatchLockerStatus(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, rid, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация", nil)
		return
	}
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Ячейка не найдена", nil)
		return
	}
	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}
	data, err := h.svc.UpdateLockerStatus(r.Context(), claims.AdminID, claims.Role, lockerID, req.Status, req.Reason)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// POST /api/v1/admin/lockers/{lockerId}/open
func (h *AdminHandler) ManualOpenLocker(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, rid, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация", nil)
		return
	}
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Ячейка не найдена", nil)
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}
	data, err := h.svc.ManualOpenLocker(r.Context(), claims.AdminID, claims.Role, lockerID, req.Reason)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// GET /api/v1/admin/sessions
func (h *AdminHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	data, total, err := h.svc.ListSessions(
		r.Context(),
		r.URL.Query().Get("locationId"),
		r.URL.Query().Get("lockerId"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("phone"),
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("limit"),
		r.URL.Query().Get("offset"),
	)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, map[string]interface{}{"total": total})
}

// GET /api/v1/admin/revenue/export
func (h *AdminHandler) RevenueExport(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		writeErr(w, rid, http.StatusUnauthorized, "UNAUTHORIZED", "Требуется авторизация", nil)
		return
	}
	data, filename, err := h.svc.RevenueExport(
		r.Context(),
		claims.Role,
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
		r.URL.Query().Get("locationId"),
	)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
