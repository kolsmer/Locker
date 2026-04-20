-- +goose Up
CREATE TABLE admins (
    id SERIAL PRIMARY KEY,
    login VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);

CREATE INDEX idx_admins_login ON admins(login);
CREATE INDEX idx_admins_active ON admins(is_active);

-- +goose Down
DROP TABLE admins;
