package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type LockerRepository struct {
	db *sql.DB
}

func NewLockerRepository(db *sql.DB) *LockerRepository {
	return &LockerRepository{db: db}
}

func (r *LockerRepository) GetByID(ctx context.Context, id int64) (*domain.Locker, error) {
	query := `
		SELECT id, location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at, last_event_at
		FROM lockers WHERE id = $1
	`
	row := r.db.QueryRowContext(ctx, query, id)
	locker := &domain.Locker{}
	err := row.Scan(&locker.ID, &locker.LocationID, &locker.LockerNo, &locker.Size,
		&locker.Status, &locker.HardwareID, &locker.IsActive, &locker.Price,
		&locker.CreatedAt, &locker.UpdatedAt, &locker.LastEventAt)
	if err != nil {
		return nil, err
	}
	return locker, nil
}

func (r *LockerRepository) GetByLocationID(ctx context.Context, locationID int64) ([]*domain.Locker, error) {
	query := `
		SELECT id, location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at, last_event_at
		FROM lockers WHERE location_id = $1 ORDER BY locker_no
	`
	rows, err := r.db.QueryContext(ctx, query, locationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lockers []*domain.Locker
	for rows.Next() {
		locker := &domain.Locker{}
		err := rows.Scan(&locker.ID, &locker.LocationID, &locker.LockerNo, &locker.Size,
			&locker.Status, &locker.HardwareID, &locker.IsActive, &locker.Price,
			&locker.CreatedAt, &locker.UpdatedAt, &locker.LastEventAt)
		if err != nil {
			return nil, err
		}
		lockers = append(lockers, locker)
	}
	return lockers, rows.Err()
}

func (r *LockerRepository) GetFreeBySize(ctx context.Context, locationID int64, size domain.LockerSize) (*domain.Locker, error) {
	query := `
		SELECT id, location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at, last_event_at
		FROM lockers
		WHERE location_id = $1 AND size = $2 AND status = $3 AND is_active = true
		LIMIT 1 FOR UPDATE
	`
	row := r.db.QueryRowContext(ctx, query, locationID, size, domain.LockerStatusFree)
	locker := &domain.Locker{}
	err := row.Scan(&locker.ID, &locker.LocationID, &locker.LockerNo, &locker.Size,
		&locker.Status, &locker.HardwareID, &locker.IsActive, &locker.Price,
		&locker.CreatedAt, &locker.UpdatedAt, &locker.LastEventAt)
	if err != nil {
		return nil, err
	}
	return locker, nil
}

func (r *LockerRepository) UpdateStatus(ctx context.Context, id int64, status domain.LockerStatus) error {
	query := `UPDATE lockers SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, nowUnix(), id)
	return err
}

func (r *LockerRepository) Create(ctx context.Context, locker *domain.Locker) (int64, error) {
	query := `
		INSERT INTO lockers (location_id, locker_no, size, status, hardware_id, is_active, price, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query,
		locker.LocationID, locker.LockerNo, locker.Size, locker.Status,
		locker.HardwareID, locker.IsActive, locker.Price, nowUnix(), nowUnix()).Scan(&id)
	return id, err
}

func nowUnix() int64 {
	return 0 // TODO: import time.Now().Unix()
}
