-- +goose Up
CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    actor_type VARCHAR(20) NOT NULL,
    actor_id INTEGER,
    action VARCHAR(100) NOT NULL,
    object_type VARCHAR(50) NOT NULL,
    object_id INTEGER,
    payload JSONB,
    created_at BIGINT NOT NULL
);

CREATE INDEX idx_audit_actor ON audit_logs(actor_type, actor_id);
CREATE INDEX idx_audit_action ON audit_logs(action);
CREATE INDEX idx_audit_created ON audit_logs(created_at);

-- +goose Down
DROP TABLE audit_logs;
