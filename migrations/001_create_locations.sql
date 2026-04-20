-- +goose Up
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT,
    latitude NUMERIC(9,6),
    longitude NUMERIC(9,6),
    is_active BOOLEAN DEFAULT true,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX idx_locations_active ON locations(is_active);

-- +goose Down
DROP TABLE locations;
