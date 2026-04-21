-- +goose Up
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES storage_sessions(id),
    rental_id VARCHAR(64),
    payment_id VARCHAR(64) UNIQUE,
    amount INTEGER NOT NULL,
    currency VARCHAR(3) DEFAULT 'RUB',
    status VARCHAR(20) NOT NULL,
    external_payment_id VARCHAR(255),
    provider VARCHAR(50),
    qr_payload TEXT,
    payment_expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    raw_callback_json TEXT
);

CREATE INDEX idx_payments_session ON payments(session_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_external_id ON payments(external_payment_id);
CREATE INDEX idx_payments_rental_id ON payments(rental_id);
CREATE INDEX idx_payments_rental_created_desc ON payments(rental_id, created_at DESC);
CREATE INDEX idx_payments_session_created_desc ON payments(session_id, created_at DESC);

-- +goose Down
DROP TABLE payments;
