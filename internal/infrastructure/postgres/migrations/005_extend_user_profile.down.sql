-- 005_extend_user_profile.down.sql
ALTER TABLE users
    DROP COLUMN IF EXISTS phone_number,
    DROP COLUMN IF EXISTS date_of_birth,
    DROP COLUMN IF EXISTS gender,
    DROP COLUMN IF EXISTS location,
    DROP COLUMN IF EXISTS website_url,
    DROP COLUMN IF EXISTS social_twitter,
    DROP COLUMN IF EXISTS social_github,
    DROP COLUMN IF EXISTS social_linkedin,
    DROP COLUMN IF EXISTS language;
