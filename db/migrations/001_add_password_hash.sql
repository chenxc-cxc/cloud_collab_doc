-- Migration: Add password_hash column to users table
-- Run this on existing databases to add the password_hash column

-- Add password_hash column if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Update test users with default password (password123)
-- bcrypt hash with cost 10
UPDATE users SET password_hash = '$2a$10$OTGhPWkBeXh.5k/tPdblHuK15nrOlfTbVj1xx0k2U3e61vyhhUIGq'
WHERE password_hash IS NULL;
