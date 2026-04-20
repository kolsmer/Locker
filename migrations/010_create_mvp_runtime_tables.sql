-- +goose Up
CREATE TABLE IF NOT EXISTS mvp_selections (
    id SERIAL PRIMARY KEY,
    selection_id VARCHAR(64) UNIQUE NOT NULL,
    locker_id INTEGER NOT NULL REFERENCES locations(id),
    locker_cell_id INTEGER NOT NULL REFERENCES lockers(id),
    size VARCHAR(4) NOT NULL,
    cell_number INTEGER NOT NULL,
    hold_expires_at TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mvp_selections_active ON mvp_selections(status, hold_expires_at);
CREATE INDEX IF NOT EXISTS idx_mvp_selections_locker ON mvp_selections(locker_id);

CREATE TABLE IF NOT EXISTS mvp_rentals (
    id SERIAL PRIMARY KEY,
    booking_id VARCHAR(64) UNIQUE NOT NULL,
    rental_id VARCHAR(64) UNIQUE NOT NULL,
    locker_id INTEGER NOT NULL REFERENCES locations(id),
    locker_cell_id INTEGER NOT NULL REFERENCES lockers(id),
    cell_number INTEGER NOT NULL,
    phone VARCHAR(20) NOT NULL,
    access_code VARCHAR(16) NOT NULL,
    state VARCHAR(20) NOT NULL,
    opened_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_mvp_rentals_code_locker ON mvp_rentals(locker_id, access_code);
CREATE INDEX IF NOT EXISTS idx_mvp_rentals_state ON mvp_rentals(state);

CREATE TABLE IF NOT EXISTS mvp_payments (
    id SERIAL PRIMARY KEY,
    payment_id VARCHAR(64) UNIQUE NOT NULL,
    rental_id VARCHAR(64) NOT NULL,
    amount INTEGER NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL,
    qr_payload TEXT NOT NULL,
    payment_expires_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_mvp_payments_rental FOREIGN KEY (rental_id) REFERENCES mvp_rentals(rental_id)
);

CREATE INDEX IF NOT EXISTS idx_mvp_payments_rental ON mvp_payments(rental_id);
CREATE INDEX IF NOT EXISTS idx_mvp_payments_status ON mvp_payments(status);

-- +goose Down
DROP TABLE IF EXISTS mvp_payments;
DROP TABLE IF EXISTS mvp_rentals;
DROP TABLE IF EXISTS mvp_selections;
