package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type SessionRepository struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.StorageSession) (int64, error) {
	query := `
		INSERT INTO storage_sessions (locker_id, phone, access_code_hash, status, started_at, ends_at, paid_until, created_source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		session.LockerID, session.Phone, session.AccessCode, session.Status,
		session.StartedAt, session.EndsAt, session.PaidUntil, session.CreatedSource,
		nowUnix(), nowUnix()).Scan(&id)
	return id, err
}

func (r *SessionRepository) GetByID(ctx context.Context, id int64) (*domain.StorageSession, error) {
	query := `
		SELECT id, locker_id, phone, access_code_hash, status, started_at, ends_at, paid_until, closed_at, open_attempts, created_source, created_at, updated_at
		FROM storage_sessions WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	session := &domain.StorageSession{}
	err := row.Scan(&session.ID, &session.LockerID, &session.Phone, &session.AccessCode, &session.Status,
		&session.StartedAt, &session.EndsAt, &session.PaidUntil, &session.ClosedAt, &session.OpenAttempts,
		&session.CreatedSource, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) GetActiveByLocker(ctx context.Context, lockerID int64) (*domain.StorageSession, error) {
	query := `
		SELECT id, locker_id, phone, access_code_hash, status, started_at, ends_at, paid_until, closed_at, open_attempts, created_source, created_at, updated_at
		FROM storage_sessions
		WHERE locker_id = $1 AND status IN ('paid', 'active')
		LIMIT 1
	`
	row := r.db.QueryRowContext(ctx, query, lockerID)
	session := &domain.StorageSession{}
	err := row.Scan(&session.ID, &session.LockerID, &session.Phone, &session.AccessCode, &session.Status,
		&session.StartedAt, &session.EndsAt, &session.PaidUntil, &session.ClosedAt, &session.OpenAttempts,
		&session.CreatedSource, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *SessionRepository) UpdateStatus(ctx context.Context, id int64, status domain.SessionStatus) error {
	query := `UPDATE storage_sessions SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, nowUnix(), id)
	return err
}

func (r *SessionRepository) UpdatePaidUntil(ctx context.Context, id int64, paidUntil int64) error {
	query := `UPDATE storage_sessions SET paid_until = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, paidUntil, nowUnix(), id)
	return err
}

func (r *SessionRepository) IncrementOpenAttempts(ctx context.Context, id int64) error {
	query := `UPDATE storage_sessions SET open_attempts = open_attempts + 1, updated_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, nowUnix(), id)
	return err
}
