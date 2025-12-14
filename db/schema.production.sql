-- =============================================================================
-- Multi-User Collaborative Document System Schema
-- Production Version - Without Test Data
-- =============================================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table (Supabase auth.users compatible)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT,
    name TEXT,
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Documents table
CREATE TABLE IF NOT EXISTS documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL DEFAULT 'Untitled Document',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Document permissions table
CREATE TABLE IF NOT EXISTS document_permissions (
    doc_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'edit', 'comment', 'view')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (doc_id, user_id)
);

-- Document snapshots for version control
CREATE TABLE IF NOT EXISTS doc_snapshots (
    doc_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    snapshot BYTEA NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (doc_id, version)
);

-- Comments table with selection support
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    doc_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    selection JSONB, -- { "anchor": number, "head": number, "blockId": string }
    resolved BOOLEAN DEFAULT FALSE,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE, -- For replies
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_documents_owner ON documents(owner_id);
CREATE INDEX IF NOT EXISTS idx_doc_permissions_user ON document_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_doc_permissions_doc ON document_permissions(doc_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_doc ON doc_snapshots(doc_id);
CREATE INDEX IF NOT EXISTS idx_comments_doc ON comments(doc_id);
CREATE INDEX IF NOT EXISTS idx_comments_user ON comments(user_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_documents_updated_at
    BEFORE UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- Migration: Add password_hash column to users table
-- Run this on existing databases to add the password_hash column

-- Add password_hash column if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;

-- Update test users with default password (password123)
-- bcrypt hash with cost 10
UPDATE users SET password_hash = '$2a$10$OTGhPWkBeXh.5k/tPdblHuK15nrOlfTbVj1xx0k2U3e61vyhhUIGq'
WHERE password_hash IS NULL;
-- Migration: Add access_requests table for permission request functionality
-- Users can request access to documents, owners can approve/reject

CREATE TABLE IF NOT EXISTS access_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    doc_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    requested_role TEXT NOT NULL DEFAULT 'view' CHECK (requested_role IN ('view', 'comment', 'edit')),
    message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(doc_id, requester_id)
);

-- Index for efficient queries
CREATE INDEX IF NOT EXISTS idx_access_requests_doc ON access_requests(doc_id);
CREATE INDEX IF NOT EXISTS idx_access_requests_requester ON access_requests(requester_id);
CREATE INDEX IF NOT EXISTS idx_access_requests_status ON access_requests(status);

-- Trigger for updated_at
CREATE TRIGGER update_access_requests_updated_at
    BEFORE UPDATE ON access_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- Migration: Add folders table for hierarchical document organization

-- Create folders table
CREATE TABLE IF NOT EXISTS folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL DEFAULT 'New Folder',
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Add folder_id to documents (NULL means root level)
ALTER TABLE documents ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES folders(id) ON DELETE SET NULL;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_folders_owner ON folders(owner_id);
CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_documents_folder ON documents(folder_id);

-- Trigger for updated_at
CREATE TRIGGER update_folders_updated_at
    BEFORE UPDATE ON folders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- Migration: Change document folder_id ON DELETE behavior from SET NULL to CASCADE
-- This ensures that when a folder is deleted, all documents within it are also deleted

-- Drop the existing foreign key constraint
ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_folder_id_fkey;

-- Add the new foreign key constraint with ON DELETE CASCADE
ALTER TABLE documents 
ADD CONSTRAINT documents_folder_id_fkey 
FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE;
