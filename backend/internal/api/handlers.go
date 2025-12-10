package api

import (
	"encoding/base64"
	"net/http"

	"github.com/collab-docs/backend/internal/auth"
	"github.com/collab-docs/backend/internal/db"
	"github.com/collab-docs/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler holds the dependencies for API handlers
type Handler struct {
	db *db.DB
}

// NewHandler creates a new API handler
func NewHandler(database *db.DB) *Handler {
	return &Handler{db: database}
}

// RegisterRoutes registers all API routes
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Health check
	r.GET("/health", h.HealthCheck)

	// Auth routes (for local dev)
	r.POST("/api/auth/login", h.DevLogin)
	r.GET("/api/auth/me", auth.DevAuthMiddleware(h.db), h.GetCurrentUser)

	// Document routes
	docs := r.Group("/api/docs")
	docs.Use(auth.DevAuthMiddleware(h.db))
	{
		docs.GET("", h.ListDocuments)
		docs.POST("", h.CreateDocument)
		docs.GET("/:id", auth.RequirePermission(h.db, models.RoleView), h.GetDocument)
		docs.PUT("/:id", auth.RequirePermission(h.db, models.RoleEdit), h.UpdateDocument)
		docs.DELETE("/:id", auth.RequirePermission(h.db, models.RoleOwner), h.DeleteDocument)

		// Permissions
		docs.GET("/:id/permissions", auth.RequirePermission(h.db, models.RoleOwner), h.ListPermissions)
		docs.PUT("/:id/permissions", auth.RequirePermission(h.db, models.RoleOwner), h.SetPermission)
		docs.DELETE("/:id/permissions/:userId", auth.RequirePermission(h.db, models.RoleOwner), h.RemovePermission)

		// Comments
		docs.GET("/:id/comments", auth.RequirePermission(h.db, models.RoleView), h.ListComments)
		docs.POST("/:id/comments", auth.RequirePermission(h.db, models.RoleComment), h.CreateComment)

		// Snapshots
		docs.GET("/:id/snapshots", auth.RequirePermission(h.db, models.RoleView), h.ListSnapshots)

		// My permission (accessible to anyone with view access)
		docs.GET("/:id/my-permission", auth.RequirePermission(h.db, models.RoleView), h.GetMyPermission)
	}

	// Comment routes (for update/delete)
	comments := r.Group("/api/comments")
	comments.Use(auth.DevAuthMiddleware(h.db))
	{
		comments.PUT("/:id", h.UpdateComment)
		comments.DELETE("/:id", h.DeleteComment)
	}

	// Yjs snapshot routes (for y-websocket persistence)
	// These are called by y-websocket server, no auth required for internal use
	yjs := r.Group("/api/yjs")
	{
		yjs.GET("/:docId/snapshot", h.GetYjsSnapshot)
		yjs.POST("/:docId/snapshot", h.SaveYjsSnapshot)
	}
}

// HealthCheck returns the health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// DevLogin handles login for local development
func (h *Handler) DevLogin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// GetCurrentUser returns the current authenticated user
func (h *Handler) GetCurrentUser(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// ListDocuments returns all documents accessible by the user
func (h *Handler) ListDocuments(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	docs, err := h.db.ListDocuments(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list documents"})
		return
	}
	if docs == nil {
		docs = []*models.Document{}
	}
	c.JSON(http.StatusOK, docs)
}

// CreateDocument creates a new document
func (h *Handler) CreateDocument(c *gin.Context) {
	user := auth.GetUserFromContext(c)

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.db.CreateDocument(c.Request.Context(), req.Title, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document"})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

// GetDocument returns a single document
func (h *Handler) GetDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	doc, err := h.db.GetDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get document"})
		return
	}
	if doc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// UpdateDocument updates a document
func (h *Handler) UpdateDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	var req models.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.db.UpdateDocument(c.Request.Context(), docID, req.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}
	if doc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	c.JSON(http.StatusOK, doc)
}

// DeleteDocument deletes a document
func (h *Handler) DeleteDocument(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	if err := h.db.DeleteDocument(c.Request.Context(), docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted"})
}

// ListPermissions returns all permissions for a document
func (h *Handler) ListPermissions(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	perms, err := h.db.ListPermissions(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}
	if perms == nil {
		perms = []*models.DocumentPermission{}
	}
	c.JSON(http.StatusOK, perms)
}

// SetPermission sets a user's permission for a document
func (h *Handler) SetPermission(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	var req models.SetPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.SetPermission(c.Request.Context(), docID, userID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set permission"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission set"})
}

// RemovePermission removes a user's permission for a document
func (h *Handler) RemovePermission(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	userIDStr := c.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.RemovePermission(c.Request.Context(), docID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove permission"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permission removed"})
}

// ListComments returns all comments for a document
func (h *Handler) ListComments(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	comments, err := h.db.ListComments(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list comments"})
		return
	}
	if comments == nil {
		comments = []*models.Comment{}
	}
	c.JSON(http.StatusOK, comments)
}

// CreateComment creates a new comment
func (h *Handler) CreateComment(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var parentID *uuid.UUID
	if req.ParentID != nil {
		id, err := uuid.Parse(*req.ParentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parent ID"})
			return
		}
		parentID = &id
	}

	comment, err := h.db.CreateComment(c.Request.Context(), docID, user.ID, req.Content, req.Selection, parentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// UpdateComment updates a comment
func (h *Handler) UpdateComment(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// Check ownership
	existing, err := h.db.GetComment(c.Request.Context(), commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if existing.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot edit other's comment"})
		return
	}

	var req models.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment, err := h.db.UpdateComment(c.Request.Context(), commentID, req.Content, req.Resolved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update comment"})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// DeleteComment deletes a comment
func (h *Handler) DeleteComment(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	commentIDStr := c.Param("id")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid comment ID"})
		return
	}

	// Check ownership
	existing, err := h.db.GetComment(c.Request.Context(), commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Comment not found"})
		return
	}
	if existing.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete other's comment"})
		return
	}

	if err := h.db.DeleteComment(c.Request.Context(), commentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Comment deleted"})
}

// ListSnapshots returns all snapshots for a document
func (h *Handler) ListSnapshots(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	snapshots, err := h.db.ListSnapshots(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list snapshots"})
		return
	}
	if snapshots == nil {
		snapshots = []*models.DocSnapshot{}
	}
	c.JSON(http.StatusOK, snapshots)
}

// GetMyPermission returns the current user's permission for a document
func (h *Handler) GetMyPermission(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	perm, err := h.db.GetPermission(c.Request.Context(), docID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get permission"})
		return
	}

	role := "view"
	if perm != nil {
		role = perm.Role
	}

	c.JSON(http.StatusOK, gin.H{"role": role})
}

// GetYjsSnapshot returns the latest Yjs snapshot for a document
func (h *Handler) GetYjsSnapshot(c *gin.Context) {
	docIDStr := c.Param("docId")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	snapshot, err := h.db.GetLatestSnapshot(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get snapshot"})
		return
	}

	if snapshot == nil {
		c.JSON(http.StatusOK, gin.H{"snapshot": nil})
		return
	}

	// Encode snapshot to base64 for transmission
	snapshotBase64 := base64.StdEncoding.EncodeToString(snapshot.Snapshot)
	c.JSON(http.StatusOK, gin.H{
		"snapshot": snapshotBase64,
		"version":  snapshot.Version,
	})
}

// SaveYjsSnapshot saves a Yjs snapshot for a document
func (h *Handler) SaveYjsSnapshot(c *gin.Context) {
	docIDStr := c.Param("docId")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var req struct {
		Snapshot string `json:"snapshot" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save snapshot (base64 encoded)
	_, err = h.db.SaveSnapshotBase64(c.Request.Context(), docID, req.Snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save snapshot"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Snapshot saved"})
}
