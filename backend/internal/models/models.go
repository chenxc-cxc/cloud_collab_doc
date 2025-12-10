package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose in JSON
	Name         string    `json:"name" db:"name"`
	AvatarURL    string    `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Document represents a collaborative document
type Document struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Title     string    `json:"title" db:"title"`
	OwnerID   uuid.UUID `json:"owner_id" db:"owner_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields
	Owner      *User  `json:"owner,omitempty"`
	Permission string `json:"permission,omitempty"`
}

// Permission roles
const (
	RoleOwner   = "owner"
	RoleEdit    = "edit"
	RoleComment = "comment"
	RoleView    = "view"
)

// DocumentPermission represents user access to a document
type DocumentPermission struct {
	DocID     uuid.UUID `json:"doc_id" db:"doc_id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	Role      string    `json:"role" db:"role"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// Joined fields
	User *User `json:"user,omitempty"`
}

// CanEdit returns true if the role allows editing
func (p *DocumentPermission) CanEdit() bool {
	return p.Role == RoleOwner || p.Role == RoleEdit
}

// CanComment returns true if the role allows commenting
func (p *DocumentPermission) CanComment() bool {
	return p.Role == RoleOwner || p.Role == RoleEdit || p.Role == RoleComment
}

// CanView returns true if the role allows viewing
func (p *DocumentPermission) CanView() bool {
	return true // All roles can view
}

// DocSnapshot represents a version snapshot of a document
type DocSnapshot struct {
	DocID     uuid.UUID `json:"doc_id" db:"doc_id"`
	Version   int       `json:"version" db:"version"`
	Snapshot  []byte    `json:"snapshot" db:"snapshot"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Selection represents a text selection in the document
type Selection struct {
	Anchor  int    `json:"anchor"`
	Head    int    `json:"head"`
	BlockID string `json:"blockId,omitempty"`
}

// Comment represents a comment on a document
type Comment struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	DocID     uuid.UUID  `json:"doc_id" db:"doc_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Content   string     `json:"content" db:"content"`
	Selection *Selection `json:"selection,omitempty" db:"selection"`
	Resolved  bool       `json:"resolved" db:"resolved"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`

	// Joined fields
	User    *User      `json:"user,omitempty"`
	Replies []*Comment `json:"replies,omitempty"`
}

// CreateDocumentRequest represents requests to create a document
type CreateDocumentRequest struct {
	Title string `json:"title" binding:"required"`
}

// UpdateDocumentRequest represents requests to update a document
type UpdateDocumentRequest struct {
	Title string `json:"title" binding:"required"`
}

// SetPermissionRequest represents a request to set document permissions
type SetPermissionRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=owner edit comment view"`
}

// CreateCommentRequest represents a request to create a comment
type CreateCommentRequest struct {
	Content   string     `json:"content" binding:"required"`
	Selection *Selection `json:"selection,omitempty"`
	ParentID  *string    `json:"parent_id,omitempty"`
}

// UpdateCommentRequest represents a request to update a comment
type UpdateCommentRequest struct {
	Content  *string `json:"content,omitempty"`
	Resolved *bool   `json:"resolved,omitempty"`
}

// Presence represents a user's cursor position and state
type Presence struct {
	UserID string          `json:"userId"`
	Name   string          `json:"name"`
	Color  string          `json:"color"`
	Cursor *CursorPosition `json:"cursor,omitempty"`
}

// CursorPosition represents a cursor position in the document
type CursorPosition struct {
	Anchor int `json:"anchor"`
	Head   int `json:"head"`
}

// CollabMessage represents a message in the collaboration protocol
type CollabMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Message types for collaboration
const (
	MsgTypeSync       = "sync"
	MsgTypeUpdate     = "update"
	MsgTypePresence   = "presence"
	MsgTypeError      = "error"
	MsgTypeConnected  = "connected"
	MsgTypeDisconnect = "disconnect"
)

// Auth request/response types

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ForgotPasswordRequest represents a forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Access request status constants
const (
	AccessRequestPending  = "pending"
	AccessRequestApproved = "approved"
	AccessRequestRejected = "rejected"
)

// AccessRequest represents a request for document access
type AccessRequest struct {
	ID            uuid.UUID `json:"id" db:"id"`
	DocID         uuid.UUID `json:"doc_id" db:"doc_id"`
	RequesterID   uuid.UUID `json:"requester_id" db:"requester_id"`
	Status        string    `json:"status" db:"status"`
	RequestedRole string    `json:"requested_role" db:"requested_role"`
	Message       string    `json:"message,omitempty" db:"message"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`

	// Joined fields
	Requester *User     `json:"requester,omitempty"`
	Document  *Document `json:"document,omitempty"`
}

// CreateAccessRequestRequest represents a request to create an access request
type CreateAccessRequestRequest struct {
	RequestedRole string `json:"requested_role,omitempty"` // defaults to 'view'
	Message       string `json:"message,omitempty"`
}

// UpdateAccessRequestRequest represents a request to update an access request status
type UpdateAccessRequestRequest struct {
	Status string `json:"status" binding:"required,oneof=approved rejected"`
}
