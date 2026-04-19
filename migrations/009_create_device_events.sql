-- +goose Up
CREATE TABLE device_events (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    locker_id INTEGER NOT NULL REFERENCES lockers(id),
    event_type VARCHAR(50) NOT NULL,
    payload JSONB,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_device_events_device ON device_events(device_id);
CREATE INDEX idx_device_events_locker ON device_events(locker_id);
CREATE INDEX idx_device_events_created ON device_events(created_at);

-- +goose Down
DROP TABLE device_events;
