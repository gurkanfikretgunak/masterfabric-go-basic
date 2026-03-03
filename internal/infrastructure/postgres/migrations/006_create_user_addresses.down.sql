-- 006_create_user_addresses.down.sql
DROP INDEX  IF EXISTS idx_user_addresses_default;
DROP INDEX  IF EXISTS idx_user_addresses_user_id;
DROP TABLE  IF EXISTS user_addresses;
