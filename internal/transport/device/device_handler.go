package device

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type DeviceHandler struct {
	// Will inject services
}

func NewDeviceHandler() *DeviceHandler {
	return &DeviceHandler{}
}

// GET /api/v1/device/{hardware_id}/commands
// Постомат заходит сюда и берёт команды
func (h *DeviceHandler) GetPendingCommands(w http.ResponseWriter, r *http.Request) {

	// TODO: fetch pending commands for this device

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]interface{}{})
}

// POST /api/v1/device/{hardware_id}/cmd/{id}/done
// Постомат отчитывается о выполнении
func (h *DeviceHandler) ReportCommandDone(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cmdIDStr := vars["id"]
	_, err := strconv.ParseInt(cmdIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid command id", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: update command status

	w.WriteHeader(http.StatusOK)
}

// POST /api/v1/device/{hardware_id}/events
// Постомат отправляет события (дверь открыта, дверь закрыта, ошибка)
func (h *DeviceHandler) ReportEvent(w http.ResponseWriter, r *http.Request) {

	var req struct {
		LockerNo  int    `json:"locker_no"`
		EventType string `json:"event_type"`
		Payload   string `json:"payload,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: process device event

	w.WriteHeader(http.StatusOK)
}

// POST /api/v1/device/{hardware_id}/heartbeat
// Постомат периодически отправляет heartbeat
func (h *DeviceHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Status   string `json:"status"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// TODO: update device status

	w.WriteHeader(http.StatusOK)
}
