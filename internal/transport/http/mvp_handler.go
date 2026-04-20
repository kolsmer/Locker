package http

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type MVPHandler struct {
	db *sql.DB
}

func NewMVPHandler(db *sql.DB) *MVPHandler {
	h := &MVPHandler{db: db}
	_ = h.ensureDemoData(context.Background())
	return h
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

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func toISO(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func normalizePhone(phone string) (string, bool) {
	digits := make([]rune, 0, len(phone))
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	if len(digits) != 11 {
		return "", false
	}
	if digits[0] == '8' {
		digits[0] = '7'
	}
	if digits[0] != '7' {
		return "", false
	}
	return "+" + string(digits), true
}

func validSize(size string) bool {
	s := strings.ToLower(strings.TrimSpace(size))
	return s == "s" || s == "m" || s == "l" || s == "xl"
}

func sizeFromDimensions(length, width, height float64) (string, bool) {
	maxD := length
	if width > maxD {
		maxD = width
	}
	if height > maxD {
		maxD = height
	}
	if maxD <= 25 {
		return "s", true
	}
	if maxD <= 45 {
		return "m", true
	}
	if maxD <= 65 {
		return "l", true
	}
	if maxD <= 90 {
		return "xl", true
	}
	return "", false
}

func genID(prefix string) string {
	letters := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return prefix + "_" + string(b)
}

func genAccessCode() string {
	letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (h *MVPHandler) cleanupExpiredSelections(ctx context.Context) error {
	_, err := h.db.ExecContext(ctx, `
		WITH expired AS (
			UPDATE mvp_selections
			SET status = 'expired', updated_at = NOW()
			WHERE status = 'active' AND hold_expires_at < NOW()
			RETURNING locker_cell_id
		)
		UPDATE lockers l
		SET status = 'free', updated_at = EXTRACT(EPOCH FROM NOW())::bigint
		FROM expired e
		WHERE l.id = e.locker_cell_id AND l.status = 'reserved'
	`)
	return err
}

func (h *MVPHandler) ensureDemoData(ctx context.Context) error {
	if h.db == nil {
		return nil
	}

	_, err := h.db.ExecContext(ctx, `
		INSERT INTO locations (id, name, address, latitude, longitude, is_active, created_at, updated_at)
		VALUES (123, 'LOCK''IT Demo', 'ULITSA PUSHKINA, 14', 0, 0, true, EXTRACT(EPOCH FROM NOW())::bigint, EXTRACT(EPOCH FROM NOW())::bigint)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return err
	}

	rows := []struct {
		No   int
		Size string
	}{
		{101, "S"}, {102, "S"}, {103, "S"}, {104, "S"},
		{201, "M"}, {202, "M"},
		{301, "L"},
	}
	for _, r := range rows {
		_, err := h.db.ExecContext(ctx, `
			INSERT INTO lockers (location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at)
			VALUES (123, $1, $2, 'free', NULL, true, 900, EXTRACT(EPOCH FROM NOW())::bigint, EXTRACT(EPOCH FROM NOW())::bigint)
			ON CONFLICT (location_id, locker_no)
			DO UPDATE SET
				size = EXCLUDED.size,
				status = 'free',
				hardware_id = NULL,
				is_active = true,
				updated_at = EXTRACT(EPOCH FROM NOW())::bigint
		`, r.No, r.Size)
		if err != nil {
			return err
		}
	}
	return nil
}

// GET /api/v1/lockers
func (h *MVPHandler) GetLockers(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()

	if err := h.cleanupExpiredSelections(ctx); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "cleanup failed", nil)
		return
	}

	city := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("city")))
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 100
	offset := 0
	if limitStr != "" {
		v, err := strconv.Atoi(limitStr)
		if err != nil || v < 0 {
			writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_LIMIT", "Некорректный limit", map[string]interface{}{"field": "limit"})
			return
		}
		limit = v
	}
	if offsetStr != "" {
		v, err := strconv.Atoi(offsetStr)
		if err != nil || v < 0 {
			writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_OFFSET", "Некорректный offset", map[string]interface{}{"field": "offset"})
			return
		}
		offset = v
	}

	query := `
		SELECT l.id, l.address,
			SUM(CASE WHEN UPPER(k.size)='S' AND k.status='free' THEN 1 ELSE 0 END) AS s,
			SUM(CASE WHEN UPPER(k.size)='M' AND k.status='free' THEN 1 ELSE 0 END) AS m,
			SUM(CASE WHEN UPPER(k.size)='L' AND k.status='free' THEN 1 ELSE 0 END) AS l,
			SUM(CASE WHEN UPPER(k.size)='XL' AND k.status='free' THEN 1 ELSE 0 END) AS xl
		FROM locations l
		LEFT JOIN lockers k ON k.location_id = l.id
		WHERE ($1 = '' OR LOWER(l.name) LIKE '%' || $1 || '%' OR LOWER(l.address) LIKE '%' || $1 || '%')
		GROUP BY l.id, l.address
		ORDER BY l.id
		LIMIT $2 OFFSET $3
	`

	rows, err := h.db.QueryContext(ctx, query, city, limit, offset)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var street string
		var s, m, l, xl int
		if err := rows.Scan(&id, &street, &s, &m, &l, &xl); err != nil {
			writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "scan error", nil)
			return
		}
		items = append(items, map[string]interface{}{
			"id":     id,
			"street": street,
			"freeCells": map[string]int{
				"s":  s,
				"m":  m,
				"l":  l,
				"xl": xl,
			},
			"updatedAt": nowISO(),
		})
	}

	var total int
	totalQuery := `SELECT COUNT(*) FROM locations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%' OR LOWER(address) LIKE '%' || $1 || '%')`
	if err := h.db.QueryRowContext(ctx, totalQuery, city).Scan(&total); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "count error", nil)
		return
	}

	writeOK(w, rid, http.StatusOK, items, map[string]interface{}{"total": total})
}

// POST /api/v1/lockers/{lockerId}/cell-selection
func (h *MVPHandler) CreateCellSelection(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	vars := mux.Vars(r)
	lockerID, err := strconv.ParseInt(vars["lockerId"], 10, 64)
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

	size := strings.ToLower(strings.TrimSpace(req.Size))
	if size == "" && req.Dimensions != nil {
		if req.Dimensions.Length <= 0 || req.Dimensions.Width <= 0 || req.Dimensions.Height <= 0 {
			writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_DIMENSIONS", "Некорректные габариты", map[string]interface{}{"field": "dimensions"})
			return
		}
		if req.Dimensions.Unit != "" && strings.ToLower(req.Dimensions.Unit) != "cm" {
			writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_DIMENSIONS", "Поддерживается только unit=cm", map[string]interface{}{"field": "dimensions.unit"})
			return
		}
		mapped, ok := sizeFromDimensions(req.Dimensions.Length, req.Dimensions.Width, req.Dimensions.Height)
		if !ok {
			writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_DIMENSIONS", "Габариты не поддерживаются", map[string]interface{}{"field": "dimensions"})
			return
		}
		size = mapped
	}
	if !validSize(size) {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_SIZE", "Некорректный размер ячейки", map[string]interface{}{"field": "size"})
		return
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "tx start failed", nil)
		return
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(ctx, `
		WITH expired AS (
			UPDATE mvp_selections SET status='expired', updated_at=NOW()
			WHERE status='active' AND hold_expires_at < NOW()
			RETURNING locker_cell_id
		)
		UPDATE lockers l SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint
		FROM expired e WHERE l.id=e.locker_cell_id AND l.status='reserved'
	`)

	var locationExists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM locations WHERE id = $1`, lockerID).Scan(&locationExists); err != nil {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Локер не найден", nil)
		return
	}

	var cellID int64
	var cellNo int
	err = tx.QueryRowContext(ctx, `
		SELECT id, locker_no
		FROM lockers
		WHERE location_id=$1 AND UPPER(size)=UPPER($2) AND status='free' AND is_active=true
		ORDER BY locker_no
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`, lockerID, size).Scan(&cellID, &cellNo)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusConflict, "NO_CELLS_AVAILABLE", "Свободных ячеек этого размера нет", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}

	selectionID := genID("sel")
	holdExpires := time.Now().UTC().Add(90 * time.Second)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO mvp_selections (selection_id, locker_id, locker_cell_id, size, cell_number, hold_expires_at, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,'active',NOW(),NOW())
	`, selectionID, lockerID, cellID, strings.ToLower(size), cellNo, holdExpires); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "selection create failed", nil)
		return
	}

	if err := tx.Commit(); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "commit failed", nil)
		return
	}

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"selectionId":   selectionID,
		"lockerId":      lockerID,
		"size":          strings.ToLower(size),
		"cellNumber":    cellNo,
		"holdExpiresAt": toISO(holdExpires),
	}, nil)
}

// POST /api/v1/lockers/{lockerId}/bookings
func (h *MVPHandler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	vars := mux.Vars(r)
	lockerID, err := strconv.ParseInt(vars["lockerId"], 10, 64)
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

	normPhone, ok := normalizePhone(req.Phone)
	if !ok {
		writeErr(w, rid, http.StatusUnprocessableEntity, "INVALID_PHONE", "Введите корректный номер телефона", map[string]interface{}{"field": "phone"})
		return
	}

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "tx start failed", nil)
		return
	}
	defer tx.Rollback()

	var selCellID int64
	var selCellNo int
	var holdExpires time.Time
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT locker_cell_id, cell_number, hold_expires_at, status
		FROM mvp_selections
		WHERE selection_id=$1 AND locker_id=$2
		FOR UPDATE
	`, req.SelectionID, lockerID).Scan(&selCellID, &selCellNo, &holdExpires, &status)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusGone, "SELECTION_EXPIRED", "Бронь истекла", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "selection read failed", nil)
		return
	}
	if status != "active" || time.Now().After(holdExpires) {
		_, _ = tx.ExecContext(ctx, `UPDATE mvp_selections SET status='expired', updated_at=NOW() WHERE selection_id=$1`, req.SelectionID)
		_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1 AND status='reserved'`, selCellID)
		writeErr(w, rid, http.StatusGone, "SELECTION_EXPIRED", "Бронь истекла", nil)
		return
	}

	var cellStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status
		FROM lockers
		WHERE id=$1
		FOR UPDATE
	`, selCellID).Scan(&cellStatus)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "LOCKER_NOT_FOUND", "Ячейка не найдена", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "locker read failed", nil)
		return
	}
	if cellStatus == "occupied" {
		writeErr(w, rid, http.StatusConflict, "CELL_ALREADY_TAKEN", "Ячейка уже занята, выберите другую", nil)
		return
	}

	bookingID := genID("book")
	rentalID := genID("rent")
	accessCode := genAccessCode()
	openedAt := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO mvp_rentals (booking_id, rental_id, locker_id, locker_cell_id, cell_number, phone, access_code, state, opened_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,'active',$8,NOW(),NOW())
	`, bookingID, rentalID, lockerID, selCellID, selCellNo, normPhone, accessCode, openedAt)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "rental create failed", nil)
		return
	}

	paymentID := genID("pay")
	paymentExpires := time.Now().UTC().Add(5 * time.Minute)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO mvp_payments (payment_id, rental_id, amount, currency, status, qr_payload, payment_expires_at, created_at, updated_at)
		VALUES ($1,$2,900,'RUB','pending',$3,$4,NOW(),NOW())
	`, paymentID, rentalID, "lockit://pay/"+paymentID, paymentExpires)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "payment create failed", nil)
		return
	}

	_, _ = tx.ExecContext(ctx, `UPDATE mvp_selections SET status='used', updated_at=NOW() WHERE selection_id=$1`, req.SelectionID)
	_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='occupied', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1`, selCellID)

	if err := tx.Commit(); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "commit failed", nil)
		return
	}

	writeOK(w, rid, http.StatusCreated, map[string]interface{}{
		"bookingId":  bookingID,
		"rentalId":   rentalID,
		"lockerId":   lockerID,
		"cellNumber": selCellNo,
		"phone":      normPhone,
		"accessCode": accessCode,
		"state":      "active",
		"openedAt":   toISO(openedAt),
	}, nil)
}

func (h *MVPHandler) refreshPaymentStatus(ctx context.Context, rentalID string) error {
	_, err := h.db.ExecContext(ctx, `
		UPDATE mvp_payments
		SET status = CASE
			WHEN status='pending' AND payment_expires_at < NOW() THEN 'expired'
			WHEN status='pending' AND created_at <= NOW() - INTERVAL '5 seconds' THEN 'paid'
			ELSE status END,
			paid_at = CASE WHEN status='pending' AND created_at <= NOW() - INTERVAL '5 seconds' THEN NOW() ELSE paid_at END,
			updated_at = NOW()
		WHERE rental_id = $1
	`)
	return err
}

// POST /api/v1/lockers/{lockerId}/access-code/check
func (h *MVPHandler) CheckAccessCode(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	vars := mux.Vars(r)
	lockerID, err := strconv.ParseInt(vars["lockerId"], 10, 64)
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
	code := strings.ToUpper(strings.TrimSpace(req.AccessCode))
	if len(code) != 6 {
		writeErr(w, rid, http.StatusNotFound, "INVALID_ACCESS_CODE", "Неверный код доступа", nil)
		return
	}

	var rentalID string
	var cellNumber int
	var phone string
	var state string
	err = h.db.QueryRowContext(ctx, `
		SELECT rental_id, cell_number, phone, state
		FROM mvp_rentals
		WHERE locker_id=$1 AND access_code=$2 AND state='active'
	`, lockerID, code).Scan(&rentalID, &cellNumber, &phone, &state)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "INVALID_ACCESS_CODE", "Неверный код доступа", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}

	_ = h.refreshPaymentStatus(ctx, rentalID)

	var paymentID, paymentStatus, currency, qrPayload string
	var amount int
	var paymentExpires time.Time
	err = h.db.QueryRowContext(ctx, `
		SELECT payment_id, status, amount, currency, qr_payload, payment_expires_at
		FROM mvp_payments
		WHERE rental_id=$1
		ORDER BY created_at DESC
		LIMIT 1
	`, rentalID).Scan(&paymentID, &paymentStatus, &amount, &currency, &qrPayload, &paymentExpires)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "payment read failed", nil)
		return
	}

	if paymentStatus == "paid" {
		writeOK(w, rid, http.StatusOK, map[string]interface{}{
			"rentalId":        rentalID,
			"lockerId":        lockerID,
			"cellNumber":      cellNumber,
			"phone":           phone,
			"accessCode":      code,
			"paymentRequired": false,
			"state":           state,
		}, nil)
		return
	}

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"rentalId":        rentalID,
		"lockerId":        lockerID,
		"cellNumber":      cellNumber,
		"phone":           phone,
		"accessCode":      code,
		"paymentRequired": true,
		"payment": map[string]interface{}{
			"paymentId":        paymentID,
			"amount":           amount,
			"currency":         currency,
			"status":           paymentStatus,
			"qrPayload":        qrPayload,
			"paymentExpiresAt": toISO(paymentExpires),
		},
	}, nil)
}

// GET /api/v1/payments/{paymentId}
func (h *MVPHandler) GetPayment(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	vars := mux.Vars(r)
	paymentID := vars["paymentId"]

	var rentalID string
	if err := h.db.QueryRowContext(ctx, `SELECT rental_id FROM mvp_payments WHERE payment_id=$1`, paymentID).Scan(&rentalID); err == nil {
		_ = h.refreshPaymentStatus(ctx, rentalID)
	}

	var status, currency string
	var amount int
	var paidAt sql.NullTime
	err := h.db.QueryRowContext(ctx, `
		SELECT status, amount, currency, paid_at
		FROM mvp_payments
		WHERE payment_id=$1
	`, paymentID).Scan(&status, &amount, &currency, &paidAt)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "PAYMENT_NOT_FOUND", "Платеж не найден", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}

	var paidAtValue interface{}
	if paidAt.Valid {
		paidAtValue = toISO(paidAt.Time)
	}

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"paymentId": paymentID,
		"status":    status,
		"amount":    amount,
		"currency":  currency,
		"paidAt":    paidAtValue,
	}, nil)
}

// POST /api/v1/rentals/{rentalId}/open
func (h *MVPHandler) OpenRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	rentalID := mux.Vars(r)["rentalId"]

	_ = h.refreshPaymentStatus(ctx, rentalID)

	var cellNumber int
	var rentalState string
	var status string
	err := h.db.QueryRowContext(ctx, `
		SELECT r.cell_number, r.state, p.status
		FROM mvp_rentals r
		LEFT JOIN mvp_payments p ON p.rental_id=r.rental_id
		WHERE r.rental_id=$1
		ORDER BY p.created_at DESC
		LIMIT 1
	`, rentalID).Scan(&cellNumber, &rentalState, &status)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "RENTAL_NOT_FOUND", "Аренда не найдена", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}
	if rentalState != "active" {
		writeErr(w, rid, http.StatusConflict, "RENTAL_CLOSED", "Аренда уже завершена", nil)
		return
	}
	if status != "paid" {
		writeErr(w, rid, http.StatusPaymentRequired, "PAYMENT_REQUIRED", "Требуется оплата", nil)
		return
	}

	openedAt := time.Now().UTC()
	_, _ = h.db.ExecContext(ctx, `UPDATE mvp_rentals SET opened_at=$1, updated_at=NOW() WHERE rental_id=$2`, openedAt, rentalID)

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"rentalId":   rentalID,
		"cellNumber": cellNumber,
		"opened":     true,
		"openedAt":   toISO(openedAt),
	}, nil)
}

// POST /api/v1/rentals/{rentalId}/finish
func (h *MVPHandler) FinishRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	rentalID := mux.Vars(r)["rentalId"]

	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "tx start failed", nil)
		return
	}
	defer tx.Rollback()

	var cellID int64
	if err := tx.QueryRowContext(ctx, `SELECT locker_cell_id FROM mvp_rentals WHERE rental_id=$1 FOR UPDATE`, rentalID).Scan(&cellID); err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "RENTAL_NOT_FOUND", "Аренда не найдена", nil)
		return
	} else if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "rental read failed", nil)
		return
	}

	finishedAt := time.Now().UTC()
	_, _ = tx.ExecContext(ctx, `UPDATE mvp_rentals SET state='closed', finished_at=$1, updated_at=NOW() WHERE rental_id=$2`, finishedAt, rentalID)
	_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1`, cellID)

	if err := tx.Commit(); err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "commit failed", nil)
		return
	}

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"rentalId":   rentalID,
		"state":      "closed",
		"finishedAt": toISO(finishedAt),
	}, nil)
}

// GET /api/v1/rentals/{rentalId}
func (h *MVPHandler) GetRental(w http.ResponseWriter, r *http.Request) {
	rid := requestID(r)
	ctx := r.Context()
	rentalID := mux.Vars(r)["rentalId"]

	var bookingID string
	var lockerID int64
	var cellNumber int
	var phone, accessCode, state string
	var openedAt time.Time
	var finishedAt sql.NullTime
	err := h.db.QueryRowContext(ctx, `
		SELECT booking_id, locker_id, cell_number, phone, access_code, state, opened_at, finished_at
		FROM mvp_rentals
		WHERE rental_id=$1
	`, rentalID).Scan(&bookingID, &lockerID, &cellNumber, &phone, &accessCode, &state, &openedAt, &finishedAt)
	if err == sql.ErrNoRows {
		writeErr(w, rid, http.StatusNotFound, "RENTAL_NOT_FOUND", "Аренда не найдена", nil)
		return
	}
	if err != nil {
		writeErr(w, rid, http.StatusInternalServerError, "INTERNAL_ERROR", "db error", nil)
		return
	}

	var finishedAtValue interface{}
	if finishedAt.Valid {
		finishedAtValue = toISO(finishedAt.Time)
	}

	writeOK(w, rid, http.StatusOK, map[string]interface{}{
		"bookingId":  bookingID,
		"rentalId":   rentalID,
		"lockerId":   lockerID,
		"cellNumber": cellNumber,
		"phone":      phone,
		"accessCode": accessCode,
		"state":      state,
		"openedAt":   toISO(openedAt),
		"finishedAt": finishedAtValue,
	}, nil)
}
