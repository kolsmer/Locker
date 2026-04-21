package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var (
	ErrLockerNotFound   = errors.New("locker_not_found")
	ErrNoCellsAvailable = errors.New("no_cells_available")
	ErrSelectionExpired = errors.New("selection_expired")
	ErrInvalidAccess    = errors.New("invalid_access_code")
	ErrPaymentRequired  = errors.New("payment_required")
	ErrPaymentNotFound  = errors.New("payment_not_found")
	ErrRentalClosed     = errors.New("rental_closed")
	ErrRentalNotFound   = errors.New("rental_not_found")
)

type RentalFlowRepository struct {
	db *sql.DB
}

func NewRentalFlowRepository(db *sql.DB) *RentalFlowRepository {
	return &RentalFlowRepository{db: db}
}

func (r *RentalFlowRepository) EnsureDemoData(ctx context.Context) error {
	if r.db == nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, `
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
	for _, row := range rows {
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO lockers (location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at)
			VALUES (123, $1, $2, 'free', NULL, true, 900, EXTRACT(EPOCH FROM NOW())::bigint, EXTRACT(EPOCH FROM NOW())::bigint)
			ON CONFLICT (location_id, locker_no)
			DO UPDATE SET
				size = EXCLUDED.size,
				status = 'free',
				hardware_id = NULL,
				is_active = true,
				updated_at = EXTRACT(EPOCH FROM NOW())::bigint
		`, row.No, row.Size)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *RentalFlowRepository) CleanupExpiredSelections(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		WITH expired AS (
			UPDATE storage_sessions
			SET status = 'expired', updated_at = NOW()
			WHERE status = 'selected' AND hold_expires_at < NOW()
			RETURNING locker_cell_id
		)
		UPDATE lockers l
		SET status = 'free', updated_at = EXTRACT(EPOCH FROM NOW())::bigint
		FROM expired e
		WHERE l.id = e.locker_cell_id AND l.status = 'reserved'
	`)
	return err
}

func (r *RentalFlowRepository) ListLockers(ctx context.Context, city string, limit int, offset int) ([]map[string]interface{}, int, error) {
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

	rows, err := r.db.QueryContext(ctx, query, city, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int64
		var street string
		var s, m, l, xl int
		if err := rows.Scan(&id, &street, &s, &m, &l, &xl); err != nil {
			return nil, 0, err
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
			"updatedAt": time.Now().UTC().Format(time.RFC3339),
		})
	}

	var total int
	totalQuery := `SELECT COUNT(*) FROM locations WHERE ($1 = '' OR LOWER(name) LIKE '%' || $1 || '%' OR LOWER(address) LIKE '%' || $1 || '%')`
	if err := r.db.QueryRowContext(ctx, totalQuery, city).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *RentalFlowRepository) CreateCellSelection(ctx context.Context, lockerID int64, size string) (map[string]interface{}, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(ctx, `
		WITH expired AS (
			UPDATE storage_sessions SET status='expired', updated_at=NOW()
			WHERE status='selected' AND hold_expires_at < NOW()
			RETURNING locker_cell_id
		)
		UPDATE lockers l SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint
		FROM expired e WHERE l.id=e.locker_cell_id AND l.status = 'reserved'
	`)

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM locations WHERE id = $1`, lockerID).Scan(&exists); err != nil {
		return nil, ErrLockerNotFound
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
		return nil, ErrNoCellsAvailable
	}
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE lockers
		SET status='reserved', updated_at=EXTRACT(EPOCH FROM NOW())::bigint
		WHERE id=$1 AND status='free'
	`, cellID); err != nil {
		return nil, err
	}

	selectionID := genID("sel")
	holdExpires := time.Now().UTC().Add(90 * time.Second)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO storage_sessions (selection_id, locker_id, locker_cell_id, selected_size, cell_number, phone, hold_expires_at, status, started_at, created_at, updated_at, created_source)
		VALUES ($1,$2,$3,$4,$5,'',$6,'selected',$7,NOW(),NOW(),'postomat')
	`, selectionID, cellID, cellID, strings.ToLower(size), cellNo, holdExpires, time.Now().Unix()); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"selectionId":   selectionID,
		"lockerId":      lockerID,
		"size":          strings.ToLower(size),
		"cellNumber":    cellNo,
		"holdExpiresAt": holdExpires.UTC().Format(time.RFC3339),
	}, nil
}

func (r *RentalFlowRepository) CreateBooking(ctx context.Context, lockerID int64, selectionID string, phone string) (map[string]interface{}, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var selCellID int64
	var selCellNo int
	var holdExpires time.Time
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT s.locker_cell_id, s.cell_number, s.hold_expires_at, s.status
		FROM storage_sessions s
		JOIN lockers l ON l.id = s.locker_cell_id
		WHERE s.selection_id=$1 AND l.location_id=$2
		FOR UPDATE
	`, selectionID, lockerID).Scan(&selCellID, &selCellNo, &holdExpires, &status)
	if err == sql.ErrNoRows {
		return nil, ErrSelectionExpired
	}
	if err != nil {
		return nil, err
	}
	if status != "selected" || time.Now().After(holdExpires) {
		_, _ = tx.ExecContext(ctx, `UPDATE storage_sessions SET status='expired', updated_at=NOW() WHERE selection_id=$1`, selectionID)
		_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1 AND status='reserved'`, selCellID)
		return nil, ErrSelectionExpired
	}

	bookingID := genID("book")
	rentalID := genID("rent")
	accessCode := genAccessCode()
	openedAt := time.Now().UTC()

	_, err = tx.ExecContext(ctx, `
		UPDATE storage_sessions
		SET booking_id=$1,
		    rental_id=$2,
		    locker_cell_id=$3,
		    cell_number=$4,
		    phone=$5,
		    access_code=$6,
		    status='active',
		    started_at=$7,
		    opened_at=$8,
		    updated_at=NOW()
		WHERE selection_id=$9
	`, bookingID, rentalID, selCellID, selCellNo, phone, accessCode, time.Now().Unix(), openedAt.Unix(), selectionID)
	if err != nil {
		return nil, err
	}

	paymentID := genID("pay")
	paymentExpires := time.Now().UTC().Add(5 * time.Minute)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payments (session_id, rental_id, amount, currency, status, payment_id, provider, qr_payload, payment_expires_at, created_at, updated_at)
		SELECT id, rental_id, 900, 'RUB', 'pending', $1, 'mock', $2, $3, NOW(), NOW()
		FROM storage_sessions
		WHERE rental_id=$4
	`, paymentID, "lockit://pay/"+paymentID, paymentExpires, rentalID)
	if err != nil {
		return nil, err
	}

	_, _ = tx.ExecContext(ctx, `UPDATE storage_sessions SET status='active', updated_at=NOW() WHERE selection_id=$1`, selectionID)
	_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='occupied', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1`, selCellID)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"bookingId":  bookingID,
		"rentalId":   rentalID,
		"lockerId":   lockerID,
		"cellNumber": selCellNo,
		"phone":      phone,
		"accessCode": accessCode,
		"state":      "active",
		"openedAt":   openedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (r *RentalFlowRepository) CheckAccessCode(ctx context.Context, lockerID int64, accessCode string) (map[string]interface{}, error) {
	var rentalID string
	var cellNumber int
	var phone string
	var state string
	err := r.db.QueryRowContext(ctx, `
		SELECT s.rental_id, s.cell_number, s.phone, s.status
		FROM storage_sessions s
		JOIN lockers l ON l.id = s.locker_cell_id
		WHERE l.location_id=$1 AND s.access_code=$2 AND s.status = 'active'
	`, lockerID, accessCode).Scan(&rentalID, &cellNumber, &phone, &state)
	if err == sql.ErrNoRows {
		return nil, ErrInvalidAccess
	}
	if err != nil {
		return nil, err
	}

	_ = r.refreshPaymentStatus(ctx, rentalID)

	var paymentID, paymentStatus, rentalStatus, currency, qrPayload string
	var amount int
	var paymentExpires time.Time
	err = r.db.QueryRowContext(ctx, `
		SELECT p.payment_id, p.status, p.amount, p.currency, p.qr_payload, p.payment_expires_at, s.status
		FROM payments p
		JOIN storage_sessions s ON s.id = p.session_id
		WHERE s.rental_id=$1
		ORDER BY p.created_at DESC
		LIMIT 1
	`, rentalID).Scan(&paymentID, &paymentStatus, &amount, &currency, &qrPayload, &paymentExpires, &rentalStatus)
	if err != nil {
		return nil, err
	}

	if rentalStatus != "active" {
		return nil, ErrRentalClosed
	}

	if paymentStatus == "paid" {
		return map[string]interface{}{
			"rentalId":        rentalID,
			"lockerId":        lockerID,
			"cellNumber":      cellNumber,
			"phone":           phone,
			"accessCode":      accessCode,
			"paymentRequired": false,
			"state":           state,
		}, nil
	}

	return map[string]interface{}{
		"rentalId":        rentalID,
		"lockerId":        lockerID,
		"cellNumber":      cellNumber,
		"phone":           phone,
		"accessCode":      accessCode,
		"paymentRequired": true,
		"payment": map[string]interface{}{
			"paymentId":        paymentID,
			"amount":           amount,
			"currency":         currency,
			"status":           paymentStatus,
			"qrPayload":        qrPayload,
			"paymentExpiresAt": paymentExpires.UTC().Format(time.RFC3339),
		},
	}, nil
}

func (r *RentalFlowRepository) GetPayment(ctx context.Context, paymentID string) (map[string]interface{}, error) {
	var rentalID string
	if err := r.db.QueryRowContext(ctx, `SELECT rental_id FROM payments WHERE payment_id=$1`, paymentID).Scan(&rentalID); err == nil {
		_ = r.refreshPaymentStatus(ctx, rentalID)
	} else {
		if err := r.db.QueryRowContext(ctx, `SELECT s.rental_id FROM payments p JOIN storage_sessions s ON s.id = p.session_id WHERE p.payment_id=$1`, paymentID).Scan(&rentalID); err == nil {
			_ = r.refreshPaymentStatus(ctx, rentalID)
		}
	}

	var status, currency string
	var amount int
	var paidAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT status, amount, currency, paid_at
		FROM payments
		WHERE payment_id=$1
	`, paymentID).Scan(&status, &amount, &currency, &paidAt)
	if err == sql.ErrNoRows {
		return nil, ErrPaymentNotFound
	}
	if err != nil {
		return nil, err
	}

	var paidAtValue interface{}
	if paidAt.Valid {
		paidAtValue = paidAt.Time.UTC().Format(time.RFC3339)
	}

	return map[string]interface{}{
		"paymentId": paymentID,
		"status":    status,
		"amount":    amount,
		"currency":  currency,
		"paidAt":    paidAtValue,
	}, nil
}

func (r *RentalFlowRepository) OpenRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	_ = r.refreshPaymentStatus(ctx, rentalID)

	var cellNumber int
	var status string
	err := r.db.QueryRowContext(ctx, `
		SELECT r.cell_number, p.status
		FROM storage_sessions r
		LEFT JOIN payments p ON p.session_id=r.id
		WHERE r.rental_id=$1
		ORDER BY p.created_at DESC
		LIMIT 1
	`, rentalID).Scan(&cellNumber, &status)
	if err == sql.ErrNoRows {
		return nil, ErrRentalNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != "paid" {
		return nil, ErrPaymentRequired
	}

	openedAt := time.Now().UTC()
	_, _ = r.db.ExecContext(ctx, `UPDATE storage_sessions SET opened_at=$1, updated_at=NOW() WHERE rental_id=$2`, openedAt.Unix(), rentalID)

	return map[string]interface{}{
		"rentalId":   rentalID,
		"cellNumber": cellNumber,
		"opened":     true,
		"openedAt":   openedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (r *RentalFlowRepository) FinishRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var cellID int64
	if err := tx.QueryRowContext(ctx, `SELECT locker_cell_id FROM storage_sessions WHERE rental_id=$1 FOR UPDATE`, rentalID).Scan(&cellID); err == sql.ErrNoRows {
		return nil, ErrRentalNotFound
	} else if err != nil {
		return nil, err
	}

	finishedAt := time.Now().UTC()
	_, _ = tx.ExecContext(ctx, `UPDATE storage_sessions SET status='closed', finished_at=$1, closed_at=$2, updated_at=NOW() WHERE rental_id=$3`, finishedAt.Unix(), finishedAt.Unix(), rentalID)
	_, _ = tx.ExecContext(ctx, `UPDATE lockers SET status='free', updated_at=EXTRACT(EPOCH FROM NOW())::bigint WHERE id=$1`, cellID)

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"rentalId":   rentalID,
		"state":      "closed",
		"finishedAt": finishedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (r *RentalFlowRepository) GetRental(ctx context.Context, rentalID string) (map[string]interface{}, error) {
	var bookingID string
	var lockerID int64
	var cellNumber int
	var phone, accessCode, state string
	var openedAt sql.NullInt64
	var finishedAt sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT s.booking_id, l.location_id, s.cell_number, s.phone, s.access_code, s.status, s.opened_at, s.finished_at
		FROM storage_sessions s
		JOIN lockers l ON l.id = s.locker_cell_id
		WHERE s.rental_id=$1
	`, rentalID).Scan(&bookingID, &lockerID, &cellNumber, &phone, &accessCode, &state, &openedAt, &finishedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRentalNotFound
	}
	if err != nil {
		return nil, err
	}

	var finishedAtValue interface{}
	if finishedAt.Valid {
		finishedAtValue = time.Unix(finishedAt.Int64, 0).UTC().Format(time.RFC3339)
	}
	var openedAtValue interface{}
	if openedAt.Valid {
		openedAtValue = time.Unix(openedAt.Int64, 0).UTC().Format(time.RFC3339)
	}

	return map[string]interface{}{
		"bookingId":  bookingID,
		"rentalId":   rentalID,
		"lockerId":   lockerID,
		"cellNumber": cellNumber,
		"phone":      phone,
		"accessCode": accessCode,
		"state":      state,
		"openedAt":   openedAtValue,
		"finishedAt": finishedAtValue,
	}, nil
}

func (r *RentalFlowRepository) refreshPaymentStatus(ctx context.Context, rentalID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments
		SET status = CASE
			WHEN status='pending' AND payment_expires_at < NOW() THEN 'expired'
			WHEN status='pending' AND created_at <= NOW() - INTERVAL '5 seconds' THEN 'paid'
			ELSE status END,
			paid_at = CASE WHEN status='pending' AND created_at <= NOW() - INTERVAL '5 seconds' THEN NOW() ELSE paid_at END,
			updated_at = NOW()
		WHERE rental_id = $1
	`, rentalID)
	return err
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

func DebugRepoError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("repo_error: %v", err)
}
