// User types
export interface User {
    id: string
    email: string
    name: string
    avatar_url?: string
    created_at: string
    updated_at: string
}

// Document types
export interface Document {
    id: string
    title: string
    owner_id: string
    folder_id?: string
    owner?: User
    permission?: string
    created_at: string
    updated_at: string
}

// Permission types
export type PermissionRole = 'owner' | 'edit' | 'comment' | 'view'

export interface DocumentPermission {
    doc_id: string
    user_id: string
    role: PermissionRole
    user?: User
    created_at: string
}

// Comment types
export interface Selection {
    anchor: number
    head: number
    blockId?: string
}

export interface Comment {
    id: string
    doc_id: string
    user_id: string
    user?: User
    content: string
    selection?: Selection
    resolved: boolean
    parent_id?: string
    replies?: Comment[]
    created_at: string
    updated_at: string
}

// Snapshot types
export interface DocSnapshot {
    doc_id: string
    version: number
    created_at: string
}

// Collaboration types
export interface Collaborator {
    userId: string
    name: string
    color: string
    cursor?: CursorPosition
}

export interface CursorPosition {
    anchor: number
    head: number
}

export interface Presence {
    userId: string
    name: string
    color: string
    cursor?: CursorPosition
}

// API response types
export interface LoginResponse {
    token: string
    user: User
}

export interface ApiError {
    error: string
}

// Access request types
export type AccessRequestStatus = 'pending' | 'approved' | 'rejected'

export interface AccessRequest {
    id: string
    doc_id: string
    requester_id: string
    requester?: User
    document?: Document  // Document info for display in notifications
    status: AccessRequestStatus
    requested_role: string
    message?: string
    created_at: string
    updated_at: string
}

// Folder types
export interface Folder {
    id: string
    name: string
    owner_id: string
    parent_id?: string
    created_at: string
    updated_at: string
}

export interface FolderContents {
    folder?: Folder
    folders: Folder[]
    documents: Document[]
}
