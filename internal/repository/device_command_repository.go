package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type DeviceCommandRepository struct {
	db *sql.DB
}

func NewDeviceCommandRepository(db *sql.DB) *DeviceCommandRepository {
	return &DeviceCommandRepository{db: db}
}

func (r *DeviceCommandRepository) Create(ctx context.Context, cmd *domain.DeviceCommand) (int64, error) {
	query := `
		INSERT INTO device_commands (device_id, locker_id, session_id, type, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, cmd.DeviceID, cmd.LockerID, cmd.SessionID, cmd.Type, cmd.Status, nowUnix()).Scan(&id)
	return id, err
}

func (r *DeviceCommandRepository) GetPendingByDevice(ctx context.Context, deviceID string) ([]*domain.DeviceCommand, error) {
	query := `
		SELECT id, device_id, locker_id, session_id, type, status, retries, error, created_at, fetched_at, done_at
		FROM device_commands
		WHERE device_id = $1 AND status = $2
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, query, deviceID, domain.CmdStatusPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []*domain.DeviceCommand
	for rows.Next() {
		cmd := &domain.DeviceCommand{}
		err := rows.Scan(&cmd.ID, &cmd.DeviceID, &cmd.LockerID, &cmd.SessionID, &cmd.Type,
			&cmd.Status, &cmd.Retries, &cmd.Error, &cmd.CreatedAt, &cmd.FetchedAt, &cmd.DoneAt)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

func (r *DeviceCommandRepository) UpdateStatus(ctx context.Context, id int64, status domain.DeviceCommandStatus, error string) error {
	query := `UPDATE device_commands SET status = $1, error = $2, done_at = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, status, error, nowUnix(), id)
	return err
}

func (r *DeviceCommandRepository) UpdateFetched(ctx context.Context, id int64) error {
	query := `UPDATE device_commands SET fetched_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, nowUnix(), id)
	return err
}
