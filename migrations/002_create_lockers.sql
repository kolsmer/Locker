-- +goose Up
CREATE TABLE lockers (
    id SERIAL PRIMARY KEY,
    location_id INTEGER NOT NULL REFERENCES locations(id),
    locker_no INTEGER NOT NULL,
    size VARCHAR(2) NOT NULL,
    status VARCHAR(20) NOT NULL,
    hardware_id VARCHAR(100) UNIQUE,
    is_active BOOLEAN DEFAULT true,
    price NUMERIC(10,2) DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    last_event_at BIGINT
);

CREATE INDEX idx_lockers_location ON lockers(location_id);
CREATE INDEX idx_lockers_status ON lockers(status);
CREATE INDEX idx_lockers_hardware_id ON lockers(hardware_id);
CREATE UNIQUE INDEX idx_lockers_unique_no ON lockers(location_id, locker_no);

-- +goose Down
DROP TABLE lockers;
