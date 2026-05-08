-- +goose Up
CREATE TABLE IF NOT EXISTS withdrawals (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number TEXT NOT NULL,
    sum NUMERIC(12, 2) NOT NULL CHECK (sum > 0),
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS withdrawals_user_processed_at_idx
    ON withdrawals (user_id, processed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS withdrawals_user_processed_at_idx;
DROP TABLE IF EXISTS withdrawals;
