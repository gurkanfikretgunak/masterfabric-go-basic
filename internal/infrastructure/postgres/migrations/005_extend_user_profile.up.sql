-- 005_extend_user_profile.up.sql
-- Adds richer profile fields to the users table.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone_number   TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS date_of_birth  DATE,
    ADD COLUMN IF NOT EXISTS gender         TEXT        NOT NULL DEFAULT ''
                                              CHECK (gender IN ('', 'male', 'female', 'other', 'prefer_not_to_say')),
    ADD COLUMN IF NOT EXISTS location       TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website_url    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS social_twitter TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS social_github  TEXT        NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS social_linkedin TEXT       NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS language       TEXT        NOT NULL DEFAULT 'en';
