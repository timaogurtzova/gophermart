-- +goose Up
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    number TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'NEW' CHECK (status IN ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED')),
    accrual NUMERIC CHECK (accrual IS NULL OR accrual >= 0),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS orders_user_uploaded_at_idx
    ON orders (user_id, uploaded_at DESC);

CREATE INDEX IF NOT EXISTS orders_status_idx
    ON orders (status);

-- +goose Down
DROP INDEX IF EXISTS orders_status_idx;
DROP INDEX IF EXISTS orders_user_uploaded_at_idx;
DROP TABLE IF EXISTS orders;
