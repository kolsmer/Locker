package http

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type PaymentHandler struct {
	// Will inject services later
}

func NewPaymentHandler() *PaymentHandler {
	return &PaymentHandler{}
}

// POST /api/v1/sessions/{id}/pay
func (h *PaymentHandler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Amount float64 `json:"amount"`
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

// POST /api/v1/webhook/payment
func (h *PaymentHandler) PaymentWebhook(w http.ResponseWriter, r *http.Request) {
	var callback map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		http.Error(w, "invalid callback", http.StatusBadRequest)
		return
	}

	// TODO: verify signature
	// TODO: process callback
	_ = callback

	w.WriteHeader(http.StatusOK)
}

// POST /api/v1/sessions/{id}/verify-code
func (h *PaymentHandler) VerifyCode(w http.ResponseWriter, r *http.Request) {
	sessionIDstr := r.URL.Query().Get("session_id")
	_, err := strconv.ParseInt(sessionIDstr, 10, 64)
	if err != nil {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	var req struct {
		AccessCode string `json:"access_code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: call service to verify

	w.Header().Set("Content-Type", "application/json")
}
