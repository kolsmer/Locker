package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
)

type AdminPanelRepository struct {
	db *sql.DB
}

func NewAdminPanelRepository(db *sql.DB) *AdminPanelRepository {
	return &AdminPanelRepository{db: db}
}

func (r *AdminPanelRepository) ListLocations(ctx context.Context, search string, isActive *bool, limit int, offset int) ([]map[string]interface{}, int, error) {
	query := `
		SELECT
			l.id,
			l.name,
			l.address,
			l.is_active,
			COUNT(k.id) AS cells_total,
			COUNT(*) FILTER (WHERE k.status = 'free') AS free_cnt,
			COUNT(*) FILTER (WHERE k.status = 'reserved') AS reserved_cnt,
			COUNT(*) FILTER (WHERE k.status = 'occupied') AS occupied_cnt,
			COUNT(*) FILTER (WHERE k.status = 'locked') AS locked_cnt,
			COUNT(*) FILTER (WHERE k.status = 'open') AS open_cnt,
			COUNT(*) FILTER (WHERE k.status = 'maintenance') AS maintenance_cnt,
			COUNT(*) FILTER (WHERE k.status = 'out_of_service') AS out_of_service_cnt,
			COALESCE(MAX(k.updated_at), l.updated_at) AS updated_epoch
		FROM locations l
		LEFT JOIN lockers k ON k.location_id = l.id
		WHERE ($1 = '' OR LOWER(l.name) LIKE '%' || $1 || '%' OR LOWER(l.address) LIKE '%' || $1 || '%')
			AND ($2::boolean IS NULL OR l.is_active = $2)
		GROUP BY l.id, l.name, l.address, l.is_active
		ORDER BY l.id
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, query, search, isActive, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var locationID int64
		var name string
		var address sql.NullString
		var active bool
		var total, free, reserved, occupied, locked, open, maintenance, outOfService int
		var updatedEpoch int64
		if err := rows.Scan(
			&locationID,
			&name,
			&address,
			&active,
			&total,
			&free,
			&reserved,
			&occupied,
			&locked,
			&open,
			&maintenance,
			&outOfService,
			&updatedEpoch,
		); err != nil {
			return nil, 0, err
		}

		items = append(items, map[string]interface{}{
			"locationId": locationID,
			"name":       name,
			"address":    address.String,
			"isActive":   active,
			"cellsTotal": total,
			"cellsByStatus": map[string]int{
				"free":           free,
				"reserved":       reserved,
				"occupied":       occupied,
				"locked":         locked,
				"open":           open,
				"maintenance":    maintenance,
				"out_of_service": outOfService,
			},
			"updatedAt": time.Unix(updatedEpoch, 0).UTC().Format(time.RFC3339),
		})
	}

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM locations l WHERE ($1 = '' OR LOWER(l.name) LIKE '%' || $1 || '%' OR LOWER(l.address) LIKE '%' || $1 || '%') AND ($2::boolean IS NULL OR l.is_active = $2)`,
		search,
		isActive,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *AdminPanelRepository) ListLocationLockers(ctx context.Context, locationID int64, statuses []string, sizes []string, isActive *bool, limit int, offset int) ([]map[string]interface{}, int, error) {
	var statusesArg interface{}
	if len(statuses) > 0 {
		statusesArg = pq.Array(statuses)
	}
	var sizesArg interface{}
	if len(sizes) > 0 {
		sizesArg = pq.Array(sizes)
	}

	query := `
		SELECT id, locker_no, size, status, is_active, price, hardware_id, last_event_at, updated_at
		FROM lockers
		WHERE location_id=$1
			AND ($2::text[] IS NULL OR status = ANY($2))
			AND ($3::text[] IS NULL OR UPPER(size) = ANY($3))
			AND ($4::boolean IS NULL OR is_active=$4)
		ORDER BY locker_no
		LIMIT $5 OFFSET $6
	`

	rows, err := r.db.QueryContext(ctx, query, locationID, statusesArg, sizesArg, isActive, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var lockerID int64
		var lockerNo int
		var size string
		var status string
		var active bool
		var price float64
		var hardwareID sql.NullString
		var lastEventAt sql.NullInt64
		var updatedAt int64
		if err := rows.Scan(&lockerID, &lockerNo, &size, &status, &active, &price, &hardwareID, &lastEventAt, &updatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, map[string]interface{}{
			"lockerId":    lockerID,
			"lockerNo":    lockerNo,
			"size":        strings.ToUpper(size),
			"status":      status,
			"isActive":    active,
			"price":       int(price),
			"hardwareId":  hardwareID.String,
			"lastEventAt": nullableInt(lastEventAt),
			"updatedAt":   updatedAt,
		})
	}

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM lockers WHERE location_id=$1 AND ($2::text[] IS NULL OR status = ANY($2)) AND ($3::text[] IS NULL OR UPPER(size)=ANY($3)) AND ($4::boolean IS NULL OR is_active=$4)`,
		locationID,
		statusesArg,
		sizesArg,
		isActive,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *AdminPanelRepository) GetLockerDetail(ctx context.Context, lockerID int64) (map[string]interface{}, error) {
	var locationID int64
	var lockerNo int
	var size string
	var status string
	var isActive bool
	var price float64
	var hardwareID sql.NullString
	if err := r.db.QueryRowContext(ctx, `SELECT location_id, locker_no, size, status, is_active, price, hardware_id FROM lockers WHERE id=$1`, lockerID).
		Scan(&locationID, &lockerNo, &size, &status, &isActive, &price, &hardwareID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLockerNotFound
		}
		return nil, err
	}

	locker := map[string]interface{}{
		"lockerId":   lockerID,
		"locationId": locationID,
		"lockerNo":   lockerNo,
		"size":       strings.ToUpper(size),
		"status":     status,
		"isActive":   isActive,
		"price":      int(price),
		"hardwareId": hardwareID.String,
	}

	var rentalID sql.NullString
	var rentalState sql.NullString
	var phone sql.NullString
	var openedAt sql.NullInt64
	var finishedAt sql.NullInt64
	_ = r.db.QueryRowContext(ctx, `
		SELECT rental_id, status, phone, opened_at, finished_at
		FROM storage_sessions
		WHERE locker_cell_id=$1 AND status='active'
		ORDER BY updated_at DESC
		LIMIT 1
	`, lockerID).Scan(&rentalID, &rentalState, &phone, &openedAt, &finishedAt)

	var activeRental interface{}
	if rentalID.Valid {
		activeRental = map[string]interface{}{
			"rentalId":    rentalID.String,
			"state":       rentalState.String,
			"phoneMasked": maskPhone(phone.String),
			"openedAt":    nullableUnixRFC3339(openedAt),
			"finishedAt":  nullableUnixRFC3339(finishedAt),
		}
	}

	var paymentID, paymentStatus, paymentCurrency sql.NullString
	var paymentAmount sql.NullInt64
	var paidAt sql.NullTime
	_ = r.db.QueryRowContext(ctx, `
		SELECT p.payment_id, p.status, p.amount, p.currency, p.paid_at
		FROM payments p
		JOIN storage_sessions s ON s.id = p.session_id
		WHERE s.locker_cell_id=$1
		ORDER BY p.created_at DESC
		LIMIT 1
	`, lockerID).Scan(&paymentID, &paymentStatus, &paymentAmount, &paymentCurrency, &paidAt)

	var lastPayment interface{}
	if paymentID.Valid {
		var paidAtValue interface{}
		if paidAt.Valid {
			paidAtValue = paidAt.Time.UTC().Format(time.RFC3339)
		}
		lastPayment = map[string]interface{}{
			"paymentId": paymentID.String,
			"status":    paymentStatus.String,
			"amount":    paymentAmount.Int64,
			"currency":  paymentCurrency.String,
			"paidAt":    paidAtValue,
		}
	}

	eventsRows, err := r.db.QueryContext(ctx, `
		SELECT id, event_type, payload, created_at
		FROM locker_events
		WHERE locker_id=$1
		ORDER BY created_at DESC
		LIMIT 20
	`, lockerID)
	if err != nil {
		return nil, err
	}
	defer eventsRows.Close()

	recentEvents := make([]map[string]interface{}, 0)
	for eventsRows.Next() {
		var id int64
		var eventType string
		var payload []byte
		var createdAt int64
		if err := eventsRows.Scan(&id, &eventType, &payload, &createdAt); err != nil {
			return nil, err
		}
		var payloadObj interface{}
		if len(payload) > 0 {
			_ = json.Unmarshal(payload, &payloadObj)
		}
		recentEvents = append(recentEvents, map[string]interface{}{
			"id":        id,
			"eventType": eventType,
			"payload":   payloadObj,
			"createdAt": createdAt,
		})
	}

	return map[string]interface{}{
		"locker":       locker,
		"activeRental": activeRental,
		"lastPayment":  lastPayment,
		"recentEvents": recentEvents,
	}, nil
}

func (r *AdminPanelRepository) UpdateLockerStatus(ctx context.Context, lockerID int64, newStatus string, reason string, actorID int64) (string, int64, error) {
	var prevStatus string
	if err := r.db.QueryRowContext(ctx, `SELECT status FROM lockers WHERE id=$1`, lockerID).Scan(&prevStatus); err != nil {
		if err == sql.ErrNoRows {
			return "", 0, ErrLockerNotFound
		}
		return "", 0, err
	}

	if newStatus == "free" || newStatus == "maintenance" {
		var activeCnt int
		if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM storage_sessions WHERE locker_cell_id=$1 AND status='active'`, lockerID).Scan(&activeCnt); err != nil {
			return "", 0, err
		}
		if activeCnt > 0 {
			return "", 0, ErrPaymentRequired
		}
	}

	now := time.Now().Unix()
	if _, err := r.db.ExecContext(ctx, `UPDATE lockers SET status=$1, updated_at=$2 WHERE id=$3`, newStatus, now, lockerID); err != nil {
		return "", 0, err
	}

	eventPayload := map[string]interface{}{"previousStatus": prevStatus, "newStatus": newStatus, "reason": reason, "actorId": actorID}
	eventJSON, _ := json.Marshal(eventPayload)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO locker_events (locker_id, session_id, event_type, payload, created_at) VALUES ($1, NULL, $2, $3, $4)`, lockerID, "admin_status_changed", string(eventJSON), now)

	auditPayload := map[string]interface{}{"previousStatus": prevStatus, "newStatus": newStatus, "reason": reason}
	auditJSON, _ := json.Marshal(auditPayload)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO audit_logs (actor_type, actor_id, action, object_type, object_id, payload, created_at) VALUES ('admin', $1, $2, 'locker', $3, $4, $5)`, actorID, "admin_locker_status_update", lockerID, string(auditJSON), now)

	return prevStatus, now, nil
}

func (r *AdminPanelRepository) ManualOpenLocker(ctx context.Context, lockerID int64, reason string, actorID int64) (int64, error) {
	var hardwareID sql.NullString
	var status string
	var isActive bool
	if err := r.db.QueryRowContext(ctx, `SELECT hardware_id, status, is_active FROM lockers WHERE id=$1`, lockerID).Scan(&hardwareID, &status, &isActive); err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrLockerNotFound
		}
		return 0, err
	}
	if !isActive || status == "maintenance" || status == "out_of_service" {
		return 0, ErrNoCellsAvailable
	}

	deviceID := hardwareID.String
	if deviceID == "" {
		deviceID = fmt.Sprintf("locker-%d", lockerID)
	}

	now := time.Now().Unix()
	var commandID int64
	if err := r.db.QueryRowContext(ctx, `
		INSERT INTO device_commands (device_id, locker_id, session_id, type, status, retries, created_at)
		VALUES ($1, $2, NULL, 'open_lock', 'pending', 0, $3)
		RETURNING id
	`, deviceID, lockerID, now).Scan(&commandID); err != nil {
		return 0, err
	}

	eventPayload := map[string]interface{}{"reason": reason, "commandId": commandID, "actorId": actorID}
	eventJSON, _ := json.Marshal(eventPayload)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO locker_events (locker_id, session_id, event_type, payload, created_at) VALUES ($1, NULL, $2, $3, $4)`, lockerID, "admin_manual_open", string(eventJSON), now)

	auditPayload := map[string]interface{}{"reason": reason, "commandId": commandID}
	auditJSON, _ := json.Marshal(auditPayload)
	_, _ = r.db.ExecContext(ctx, `INSERT INTO audit_logs (actor_type, actor_id, action, object_type, object_id, payload, created_at) VALUES ('admin', $1, $2, 'locker', $3, $4, $5)`, actorID, "admin_manual_open", lockerID, string(auditJSON), now)

	return commandID, nil
}

func (r *AdminPanelRepository) ListSessions(ctx context.Context, locationID *int64, lockerID *int64, statuses []string, phone string, from *int64, to *int64, limit int, offset int) ([]map[string]interface{}, int, error) {
	var statusesArg interface{}
	if len(statuses) > 0 {
		statusesArg = pq.Array(statuses)
	}

	query := `
		SELECT s.id, s.locker_cell_id, l.locker_no, l.location_id, s.phone, s.status, s.started_at, s.paid_until, s.closed_at
		FROM storage_sessions s
		JOIN lockers l ON l.id = s.locker_cell_id
		WHERE ($1::bigint IS NULL OR l.location_id=$1)
			AND ($2::bigint IS NULL OR s.locker_cell_id=$2)
			AND ($3::text[] IS NULL OR s.status = ANY($3))
			AND ($4 = '' OR s.phone LIKE '%' || $4 || '%')
			AND ($5::bigint IS NULL OR s.started_at >= $5)
			AND ($6::bigint IS NULL OR s.started_at <= $6)
		ORDER BY s.started_at DESC
		LIMIT $7 OFFSET $8
	`

	rows, err := r.db.QueryContext(ctx, query, locationID, lockerID, statusesArg, phone, from, to, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var sessionID int64
		var cellLockerID int64
		var lockerNo int
		var locID int64
		var phoneRaw string
		var status string
		var startedAt int64
		var paidUntil sql.NullInt64
		var closedAt sql.NullInt64
		if err := rows.Scan(&sessionID, &cellLockerID, &lockerNo, &locID, &phoneRaw, &status, &startedAt, &paidUntil, &closedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, map[string]interface{}{
			"sessionId":   sessionID,
			"lockerId":    cellLockerID,
			"lockerNo":    lockerNo,
			"locationId":  locID,
			"phoneMasked": maskPhone(phoneRaw),
			"status":      status,
			"startedAt":   startedAt,
			"paidUntil":   nullableInt(paidUntil),
			"closedAt":    nullableInt(closedAt),
		})
	}

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM storage_sessions s JOIN lockers l ON l.id=s.locker_cell_id WHERE ($1::bigint IS NULL OR l.location_id=$1) AND ($2::bigint IS NULL OR s.locker_cell_id=$2) AND ($3::text[] IS NULL OR s.status = ANY($3)) AND ($4 = '' OR s.phone LIKE '%' || $4 || '%') AND ($5::bigint IS NULL OR s.started_at >= $5) AND ($6::bigint IS NULL OR s.started_at <= $6)`,
		locationID,
		lockerID,
		statusesArg,
		phone,
		from,
		to,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

type RevenueRow struct {
	LocationID     int64
	LocationName   string
	Address        string
	PaymentsCount  int64
	RevenueRUB     int64
	AvgCheckRUB    float64
	FirstPaymentAt *time.Time
	LastPaymentAt  *time.Time
}

func (r *AdminPanelRepository) RevenueByLocation(ctx context.Context, from time.Time, to time.Time, locationID *int64) ([]RevenueRow, error) {
	query := `
		SELECT
			loc.id,
			loc.name,
			COALESCE(loc.address, ''),
			COUNT(*) AS payments_count,
			COALESCE(SUM(p.amount), 0) AS revenue_rub,
			COALESCE(AVG(p.amount), 0) AS avg_check_rub,
			MIN(COALESCE(p.paid_at, p.created_at)) AS first_payment_at,
			MAX(COALESCE(p.paid_at, p.created_at)) AS last_payment_at
		FROM payments p
		JOIN storage_sessions s ON s.id = p.session_id
		JOIN lockers l ON l.id = s.locker_cell_id
		JOIN locations loc ON loc.id = l.location_id
		WHERE p.status IN ('paid', 'confirmed')
			AND COALESCE(p.paid_at, p.created_at) >= $1
			AND COALESCE(p.paid_at, p.created_at) < $2
			AND ($3::bigint IS NULL OR loc.id = $3)
		GROUP BY loc.id, loc.name, loc.address
		ORDER BY loc.id
	`

	rows, err := r.db.QueryContext(ctx, query, from, to, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]RevenueRow, 0)
	for rows.Next() {
		var row RevenueRow
		var first sql.NullTime
		var last sql.NullTime
		if err := rows.Scan(&row.LocationID, &row.LocationName, &row.Address, &row.PaymentsCount, &row.RevenueRUB, &row.AvgCheckRUB, &first, &last); err != nil {
			return nil, err
		}
		if first.Valid {
			row.FirstPaymentAt = &first.Time
		}
		if last.Valid {
			row.LastPaymentAt = &last.Time
		}
		items = append(items, row)
	}

	return items, rows.Err()
}

func nullableInt(v sql.NullInt64) interface{} {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func nullableUnixRFC3339(v sql.NullInt64) interface{} {
	if !v.Valid {
		return nil
	}
	return time.Unix(v.Int64, 0).UTC().Format(time.RFC3339)
}

func maskPhone(phone string) string {
	if len(phone) < 6 {
		return phone
	}
	if len(phone) <= 8 {
		return phone[:2] + "******"
	}
	return phone[:3] + "******" + phone[len(phone)-2:]
}
