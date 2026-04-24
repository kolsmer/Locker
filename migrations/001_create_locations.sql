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

CREATE INDEX idx_locations_name_lower ON locations (LOWER(name));
CREATE INDEX idx_locations_address_lower ON locations (LOWER(address));

-- +goose Down
DROP TABLE locations;
