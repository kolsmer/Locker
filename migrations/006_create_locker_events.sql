-- +goose Up
CREATE TABLE locker_events (
    id SERIAL PRIMARY KEY,
    locker_id INTEGER NOT NULL REFERENCES lockers(id),
    session_id INTEGER REFERENCES storage_sessions(id),
    event_type VARCHAR(50) NOT NULL,
    payload JSONB,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_events_locker ON locker_events(locker_id);
CREATE INDEX idx_events_session ON locker_events(session_id);
CREATE INDEX idx_events_type ON locker_events(event_type);
CREATE INDEX idx_events_created ON locker_events(created_at);

-- +goose Down
DROP TABLE locker_events;
