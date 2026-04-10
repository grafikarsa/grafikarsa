-- ============================================================================
-- ROLLBACK MIGRATION: Remove User Contact Information and Privacy Settings
-- ============================================================================
-- Description: Rollback untuk migration 007_add_user_contact_info.sql
-- Date: 2026-04-11
-- ============================================================================

-- Drop indexes first
DROP INDEX IF EXISTS idx_users_show_address;
DROP INDEX IF EXISTS idx_users_show_phone;
DROP INDEX IF EXISTS idx_users_show_email;

-- Remove privacy settings columns
ALTER TABLE users 
DROP COLUMN IF EXISTS show_address,
DROP COLUMN IF EXISTS show_phone,
DROP COLUMN IF EXISTS show_email;

-- Remove contact information columns
ALTER TABLE users 
DROP COLUMN IF EXISTS address,
DROP COLUMN IF EXISTS phone;
