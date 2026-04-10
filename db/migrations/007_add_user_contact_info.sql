-- ============================================================================
-- MIGRATION: Add User Contact Information and Privacy Settings
-- ============================================================================
-- Description: Menambahkan field kontak (phone, address) dan privacy settings
--              untuk mengontrol visibility informasi di profil publik
-- Date: 2026-04-11
-- ============================================================================

-- Add contact information columns to users table
ALTER TABLE users 
ADD COLUMN phone VARCHAR(20),
ADD COLUMN address TEXT;

-- Add privacy settings columns to users table
ALTER TABLE users 
ADD COLUMN show_email BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN show_phone BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN show_address BOOLEAN NOT NULL DEFAULT FALSE;

-- Add comments for documentation
COMMENT ON COLUMN users.phone IS 'Nomor telepon user (opsional)';
COMMENT ON COLUMN users.address IS 'Alamat lengkap user (opsional)';
COMMENT ON COLUMN users.show_email IS 'Tampilkan email di profil publik (default: false)';
COMMENT ON COLUMN users.show_phone IS 'Tampilkan nomor telepon di profil publik (default: false)';
COMMENT ON COLUMN users.show_address IS 'Tampilkan alamat di profil publik (default: false)';

-- Create indexes for better query performance (optional, for future filtering)
CREATE INDEX idx_users_show_email ON users(show_email) WHERE show_email = TRUE;
CREATE INDEX idx_users_show_phone ON users(show_phone) WHERE show_phone = TRUE;
CREATE INDEX idx_users_show_address ON users(show_address) WHERE show_address = TRUE;
