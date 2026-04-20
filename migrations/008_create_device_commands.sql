-- +goose Up
CREATE TABLE device_commands (
    id SERIAL PRIMARY KEY,
    device_id VARCHAR(100) NOT NULL,
    locker_id INTEGER NOT NULL REFERENCES lockers(id),
    session_id INTEGER REFERENCES storage_sessions(id),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    retries INTEGER DEFAULT 0,
    error TEXT,
    created_at BIGINT NOT NULL,
    fetched_at BIGINT,
    done_at BIGINT
);

CREATE INDEX idx_device_cmd_device ON device_commands(device_id);
CREATE INDEX idx_device_cmd_status ON device_commands(status);
CREATE INDEX idx_device_cmd_created ON device_commands(created_at);

-- +goose Down
DROP TABLE device_commands;
