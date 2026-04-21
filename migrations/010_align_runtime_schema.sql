-- +goose Up
ALTER TABLE storage_sessions
    ADD COLUMN IF NOT EXISTS selection_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS selected_size VARCHAR(4),
    ADD COLUMN IF NOT EXISTS hold_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS booking_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS rental_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS locker_cell_id INTEGER REFERENCES lockers(id),
    ADD COLUMN IF NOT EXISTS cell_number INTEGER,
    ADD COLUMN IF NOT EXISTS access_code VARCHAR(16),
    ADD COLUMN IF NOT EXISTS opened_at BIGINT,
    ADD COLUMN IF NOT EXISTS finished_at BIGINT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'storage_sessions'
          AND column_name = 'created_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE storage_sessions
            ALTER COLUMN created_at TYPE TIMESTAMPTZ
            USING to_timestamp(created_at);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'storage_sessions'
          AND column_name = 'updated_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE storage_sessions
            ALTER COLUMN updated_at TYPE TIMESTAMPTZ
            USING to_timestamp(updated_at);
    END IF;
END $$;

ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS rental_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS payment_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS payment_expires_at TIMESTAMPTZ;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'paid_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE payments
            ALTER COLUMN paid_at TYPE TIMESTAMPTZ
            USING CASE WHEN paid_at IS NULL OR paid_at <= 0 THEN NULL ELSE to_timestamp(paid_at) END;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'created_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE payments
            ALTER COLUMN created_at TYPE TIMESTAMPTZ
            USING to_timestamp(created_at);
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'payments'
          AND column_name = 'updated_at'
          AND data_type = 'bigint'
    ) THEN
        ALTER TABLE payments
            ALTER COLUMN updated_at TYPE TIMESTAMPTZ
            USING to_timestamp(updated_at);
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_selection_id_unique
    ON storage_sessions(selection_id)
    WHERE selection_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_booking_id_unique
    ON storage_sessions(booking_id)
    WHERE booking_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_rental_id_unique
    ON storage_sessions(rental_id)
    WHERE rental_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_selected_expiry
    ON storage_sessions(hold_expires_at)
    WHERE status = 'selected';

CREATE INDEX IF NOT EXISTS idx_sessions_access_active_cell
    ON storage_sessions(access_code, locker_cell_id)
    WHERE status = 'active';

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_payment_id_unique
    ON payments(payment_id)
    WHERE payment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_payments_rental_id
    ON payments(rental_id);

CREATE INDEX IF NOT EXISTS idx_payments_rental_created_desc
    ON payments(rental_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payments_session_created_desc
    ON payments(session_id, created_at DESC);

-- +goose Down
-- This migration aligns legacy schemas with the current runtime contract.
-- It is intentionally non-reversible in automated down migration.