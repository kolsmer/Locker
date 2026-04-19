-- +goose Up
CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    session_id INTEGER NOT NULL REFERENCES storage_sessions(id),
    amount NUMERIC(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'RUB',
    status VARCHAR(20) NOT NULL,
    external_payment_id VARCHAR(255),
    provider VARCHAR(50),
    qr_payload TEXT,
    paid_at BIGINT,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    raw_callback_json TEXT
);

CREATE INDEX idx_payments_session ON payments(session_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_external_id ON payments(external_payment_id);

-- +goose Down
DROP TABLE payments;
