-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

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

CREATE INDEX idx_locations_name_trgm ON locations USING GIN (LOWER(name) gin_trgm_ops);
CREATE INDEX idx_locations_address_trgm ON locations USING GIN (LOWER(address) gin_trgm_ops);

-- +goose Down
DROP TABLE locations;
