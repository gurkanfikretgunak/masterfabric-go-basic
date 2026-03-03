-- 006_create_user_addresses.up.sql
-- Creates the user_addresses table: one-to-many relationship with users.
-- A user may have multiple addresses; exactly one may be marked as the default.

CREATE TABLE IF NOT EXISTS user_addresses (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title        TEXT        NOT NULL DEFAULT '',          -- e.g. "Home", "Work", "Billing"
    address_line1 TEXT       NOT NULL DEFAULT '',          -- street / address line 1
    address_line2 TEXT       NOT NULL DEFAULT '',          -- apartment, suite, unit, etc.
    city         TEXT        NOT NULL DEFAULT '',
    state        TEXT        NOT NULL DEFAULT '',          -- state or province
    postal_code  TEXT        NOT NULL DEFAULT '',
    country      TEXT        NOT NULL DEFAULT '',
    is_default   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup of all addresses for a given user
CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);

-- Enforce at most one default address per user at the database level
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_addresses_default
    ON user_addresses(user_id)
    WHERE is_default = TRUE;
