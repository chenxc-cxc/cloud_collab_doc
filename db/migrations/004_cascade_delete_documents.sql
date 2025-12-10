-- Migration: Change document folder_id ON DELETE behavior from SET NULL to CASCADE
-- This ensures that when a folder is deleted, all documents within it are also deleted

-- Drop the existing foreign key constraint
ALTER TABLE documents DROP CONSTRAINT IF EXISTS documents_folder_id_fkey;

-- Add the new foreign key constraint with ON DELETE CASCADE
ALTER TABLE documents 
ADD CONSTRAINT documents_folder_id_fkey 
FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE;
