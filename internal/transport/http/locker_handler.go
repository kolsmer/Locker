package http

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"locker/internal/service"

	"github.com/gorilla/mux"
)

type LockerHandler struct {
	svc *service.RentalFlowService
}

func NewLockerHandler(svc *service.RentalFlowService) *LockerHandler {
	return &LockerHandler{svc: svc}
}

type apiError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func writeOK(w http.ResponseWriter, requestID string, status int, data interface{}, meta interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"data":      data,
		"meta":      meta,
		"requestId": requestID,
	})
}

func writeErr(w http.ResponseWriter, requestID string, status int, code string, message string, details map[string]interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": false,
		"error": apiError{
			Code:    code,
			Message: message,
			Details: details,
		},
		"requestId": requestID,
	})
}

func requestID(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Request-Id")); v != "" {
		return v
	}
	return fmt.Sprintf("req_%d_%06d", time.Now().UnixNano(), rand.Intn(1000000))
}

func (h *LockerHandler) handleError(w http.ResponseWriter, rid string, err error) {
	if appErr, ok := err.(*service.AppError); ok {
		writeErr(w, rid, appErr.Status, appErr.Code, appErr.Message, appErr.Details)
		return
	}
	writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error", nil)
}

// GET /api/v1/lockers
func (h *LockerHandler) GetLockers(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	data, total, err := h.svc.GetLockers(r.Context(), r.URL.Query().Get("city"), r.URL.Query().Get("limit"), r.URL.Query().Get("offset"))
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, map[string]interface{}{"total": total})
}

// POST /api/v1/lockers/{lockerId}/cell-selection
func (h *LockerHandler) CreateCellSelection(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Локер не найден", nil)
		return
	}

	var req struct {
		Size       string `json:"size"`
		Dimensions *struct {
			Length float64 `json:"length"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
			Unit   string  `json:"unit"`
		} `json:"dimensions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}

	var dimensions map[string]interface{}
	if req.Dimensions != nil {
		dimensions = map[string]interface{}{
			"length": req.Dimensions.Length,
			"width":  req.Dimensions.Width,
			"height": req.Dimensions.Height,
			"unit":   req.Dimensions.Unit,
		}
	}

	data, err := h.svc.CreateCellSelection(r.Context(), lockerID, req.Size, dimensions)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// POST /api/v1/lockers/{lockerId}/bookings
func (h *LockerHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Локер не найден", nil)
		return
	}

	var req struct {
		SelectionID string `json:"selectionId"`
		Phone       string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}

	data, err := h.svc.CreateBooking(r.Context(), lockerID, req.SelectionID, req.Phone)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusCreated, data, nil)
}

// POST /api/v1/lockers/{lockerId}/access-code/check
func (h *LockerHandler) CheckAccessCode(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	lockerID, err := strconv.ParseInt(mux.Vars(r)["lockerId"], 10, 64)
	if err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Локер не найден", nil)
		return
	}

	var req struct {
		AccessCode string `json:"accessCode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_BODY", "Некорректное тело запроса", nil)
		return
	}

	data, err := h.svc.CheckAccessCode(r.Context(), lockerID, req.AccessCode)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// GET /api/v1/payments/{paymentId}
func (h *LockerHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	paymentID := mux.Vars(r)["paymentId"]
	data, err := h.svc.GetPayment(r.Context(), paymentID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// POST /api/v1/rentals/{rentalId}/open
func (h *LockerHandler) OpenRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	rentalID := mux.Vars(r)["rentalId"]
	data, err := h.svc.OpenRental(r.Context(), rentalID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// POST /api/v1/rentals/{rentalId}/finish
func (h *LockerHandler) FinishRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	rentalID := mux.Vars(r)["rentalId"]
	data, err := h.svc.FinishRental(r.Context(), rentalID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}

// GET /api/v1/rentals/{rentalId}
func (h *LockerHandler) GetRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	rentalID := mux.Vars(r)["rentalId"]
	data, err := h.svc.GetRental(r.Context(), rentalID)
	if err != nil {
		h.handleError(w, rid, err)
		return
	}
	writeOK(w, rid, http.StatusOK, data, nil)
}
