-- Migration: Add teacher role and NIP field
-- Created: 2026-05-18
-- Description: Adds 'teacher' role to user_role enum and nip column to users table

-- 1. Add 'teacher' to user_role enum
ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'teacher';

-- 2. Add nip column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS nip VARCHAR(30);

-- 3. Add check constraint for nip (must be numeric if provided)
ALTER TABLE users ADD CONSTRAINT users_nip_numeric CHECK (nip IS NULL OR nip ~ '^\d+$');
