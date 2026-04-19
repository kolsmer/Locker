-- +goose Up
CREATE TABLE storage_sessions (
    id SERIAL PRIMARY KEY,
    locker_id INTEGER NOT NULL REFERENCES lockers(id),
    phone VARCHAR(20) NOT NULL,
    access_code_hash VARCHAR(255),
    status VARCHAR(20) NOT NULL,
    started_at BIGINT NOT NULL,
    ends_at BIGINT,
    paid_until BIGINT,
    closed_at BIGINT,
    open_attempts INTEGER DEFAULT 0,
    created_source VARCHAR(20),
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX idx_sessions_locker ON storage_sessions(locker_id);
CREATE INDEX idx_sessions_phone ON storage_sessions(phone);
CREATE INDEX idx_sessions_status ON storage_sessions(status);
CREATE INDEX idx_sessions_paid_until ON storage_sessions(paid_until);

-- +goose Down
DROP TABLE storage_sessions;
