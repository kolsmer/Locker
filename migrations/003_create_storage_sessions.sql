-- +goose Up
CREATE TABLE storage_sessions (
    id SERIAL PRIMARY KEY,
    locker_id INTEGER NOT NULL REFERENCES lockers(id),
    phone VARCHAR(20) NOT NULL,
    access_code_hash VARCHAR(255),
    selection_id VARCHAR(64) UNIQUE,
    selected_size VARCHAR(4),
    hold_expires_at TIMESTAMPTZ,
    booking_id VARCHAR(64) UNIQUE,
    rental_id VARCHAR(64) UNIQUE,
    locker_cell_id INTEGER REFERENCES lockers(id),
    cell_number INTEGER,
    access_code VARCHAR(16),
    opened_at BIGINT,
    finished_at BIGINT,
    status VARCHAR(20) NOT NULL,
    started_at BIGINT NOT NULL,
    ends_at BIGINT,
    paid_until BIGINT,
    closed_at BIGINT,
    open_attempts INTEGER DEFAULT 0,
    created_source VARCHAR(20),
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_locker ON storage_sessions(locker_id);
CREATE INDEX idx_sessions_phone ON storage_sessions(phone);
CREATE INDEX idx_sessions_status ON storage_sessions(status);
CREATE INDEX idx_sessions_paid_until ON storage_sessions(paid_until);
CREATE INDEX idx_sessions_selected_expiry ON storage_sessions(hold_expires_at)
WHERE status = 'selected';
CREATE INDEX idx_sessions_access_active_cell ON storage_sessions(access_code, locker_cell_id)
WHERE status = 'active';

-- +goose Down
DROP TABLE storage_sessions;
