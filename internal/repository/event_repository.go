package repository

import (
	"context"
	"database/sql"
	"locker/internal/domain"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) LogLockerEvent(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO locker_events (locker_id, session_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, query, event.LockerID, event.SessionID, event.Type, event.Payload, nowUnix())
	return err
}

func (r *EventRepository) LogAudit(ctx context.Context, audit *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (actor_type, actor_id, action, object_type, object_id, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query, audit.ActorType, audit.ActorID, audit.Action,
		audit.ObjectType, audit.ObjectID, audit.Payload, nowUnix())
	return err
}

func (r *EventRepository) GetLockerEvents(ctx context.Context, lockerID int64, limit int) ([]*domain.Event, error) {
	query := `
		SELECT id, locker_id, session_id, event_type, payload, created_at
		FROM locker_events WHERE locker_id = $1
		ORDER BY created_at DESC LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, lockerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		event := &domain.Event{}
		err := rows.Scan(&event.ID, &event.LockerID, &event.SessionID, &event.Type, &event.Payload, &event.CreatedAt)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
