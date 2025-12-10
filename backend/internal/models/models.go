package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	AvatarURL string    `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
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
