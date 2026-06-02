-- +goose Up
CREATE TABLE IF NOT EXISTS balances (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_balance NUMERIC NOT NULL DEFAULT 0 CHECK (current_balance >= 0),
    withdrawn_total NUMERIC NOT NULL DEFAULT 0 CHECK (withdrawn_total >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS balances;
