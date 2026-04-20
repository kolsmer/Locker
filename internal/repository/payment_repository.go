package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, payment *domain.Payment) (int64, error) {
	query := `
		INSERT INTO payments (session_id, amount, currency, status, provider, qr_payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		payment.SessionID, payment.Amount, payment.Currency, payment.Status,
		payment.Provider, payment.QRPayload, nowUnix(), nowUnix()).Scan(&id)
	return id, err
}

func (r *PaymentRepository) GetByID(ctx context.Context, id int64) (*domain.Payment, error) {
	query := `
		SELECT id, session_id, amount, currency, status, external_payment_id, provider, qr_payload, paid_at, created_at, updated_at, raw_callback_json
		FROM payments WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	payment := &domain.Payment{}
	err := row.Scan(&payment.ID, &payment.SessionID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.ExternalPaymentID, &payment.Provider, &payment.QRPayload,
		&payment.PaidAt, &payment.CreatedAt, &payment.UpdatedAt, &payment.RawCallbackJSON)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

func (r *PaymentRepository) GetBySession(ctx context.Context, sessionID int64) (*domain.Payment, error) {
	query := `
		SELECT id, session_id, amount, currency, status, external_payment_id, provider, qr_payload, paid_at, created_at, updated_at, raw_callback_json
		FROM payments WHERE session_id = $1 LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, sessionID)
	payment := &domain.Payment{}
	err := row.Scan(&payment.ID, &payment.SessionID, &payment.Amount, &payment.Currency,
		&payment.Status, &payment.ExternalPaymentID, &payment.Provider, &payment.QRPayload,
		&payment.PaidAt, &payment.CreatedAt, &payment.UpdatedAt, &payment.RawCallbackJSON)
	if err != nil {
		return nil, err
	}
	return payment, nil
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, id int64, status domain.PaymentStatus, paidAt int64, extID string) error {
	query := `UPDATE payments SET status = $1, paid_at = $2, external_payment_id = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, status, paidAt, extID, nowUnix(), id)
	return err
}

func (r *PaymentRepository) SaveCallback(ctx context.Context, id int64, rawJSON string) error {
	query := `UPDATE payments SET raw_callback_json = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, rawJSON, nowUnix(), id)
	return err
}
