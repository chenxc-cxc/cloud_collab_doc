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

	// Public auth routes (no auth required)
	r.POST("/api/auth/register", h.Register)
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/forgot-password", h.ForgotPassword)
	r.POST("/api/auth/reset-password", h.ResetPassword)

	// Protected auth routes
	authRoutes := r.Group("/api/auth")
	authRoutes.Use(auth.AuthMiddleware(h.db))
	{
		authRoutes.GET("/me", h.GetCurrentUser)
		authRoutes.POST("/logout", h.Logout)
		authRoutes.PUT("/password", h.ChangePassword)
	}

	// Document routes
	docs := r.Group("/api/docs")
	docs.Use(auth.AuthMiddleware(h.db))
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

		// Access requests
		docs.POST("/:id/access-request", h.RequestAccess) // No permission required - user is requesting access
		docs.GET("/:id/access-requests", auth.RequirePermission(h.db, models.RoleOwner), h.ListAccessRequests)

		// Move document
		docs.PUT("/:id/move", auth.RequirePermission(h.db, models.RoleOwner), h.MoveDocument)
	}

	// Comment routes (for update/delete)
	comments := r.Group("/api/comments")
	comments.Use(auth.AuthMiddleware(h.db))
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

	// Access request routes (for update)
	accessReqs := r.Group("/api/access-requests")
	accessReqs.Use(auth.AuthMiddleware(h.db))
	{
		accessReqs.GET("/pending", h.ListMyPendingAccessRequests)
		accessReqs.PUT("/:id", h.UpdateAccessRequest)
	}

	// Folder routes
	folders := r.Group("/api/folders")
	folders.Use(auth.AuthMiddleware(h.db))
	{
		folders.POST("", h.CreateFolder)
		folders.GET("", h.GetFolderContents)  // Query param: folder_id (optional)
		folders.GET("/tree", h.GetFolderTree) // Get complete folder tree
		folders.GET("/:id", h.GetFolderByID)
		folders.GET("/:id/path", h.GetFolderPath) // Get full parent chain
		folders.PUT("/:id", h.UpdateFolder)
		folders.DELETE("/:id", h.DeleteFolder)
		folders.PUT("/:id/move", h.MoveFolder)
	}
}

// HealthCheck returns the health status
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Register handles user registration
func (h *Handler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	existingUser, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user, err := h.db.CreateUserWithPassword(c.Request.Context(), req.Email, req.Name, passwordHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Create welcome document for the new user
	welcomeTitle := "👋 Welcome to CollabDocs, " + user.Name + "!"
	_, err = h.db.CreateDocumentWithInitialContent(c.Request.Context(), welcomeTitle, user.ID)
	if err != nil {
		// Log error but don't fail registration
		// The user can still use the app, just won't have the welcome doc
		// In production, you might want to log this properly
	}

	// Generate token
	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// Login handles user login with email and password
func (h *Handler) Login(c *gin.Context) {
	var req models.LoginRequest
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		Token: token,
		User:  user,
	})
}

// Logout handles user logout
func (h *Handler) Logout(c *gin.Context) {
	// JWT tokens are stateless, so we just return success
	// In a production system, you might want to add the token to a blacklist in Redis
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ChangePassword handles password change for authenticated users
func (h *Handler) ChangePassword(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify old password
	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password
	if err := h.db.UpdateUserPassword(c.Request.Context(), user.ID, newPasswordHash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ForgotPassword handles forgot password request
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	user, err := h.db.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Always return success to prevent email enumeration
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link will be sent"})
		return
	}

	// Generate reset token
	resetToken, err := auth.GenerateResetToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate reset token"})
		return
	}

	// TODO: Store reset token in Redis with expiration
	// TODO: Send email with reset link
	// For now, just log it (development only)
	_ = resetToken // In production, send this via email

	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link will be sent"})
}

// ResetPassword handles password reset with token
func (h *Handler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Validate reset token from Redis and get associated user email
	// For now, this is a placeholder that returns an error
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Password reset via email is not configured. Please contact an administrator."})
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

// RequestAccess handles access request from users without permission
func (h *Handler) RequestAccess(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	docIDStr := c.Param("id")
	docID, err := uuid.Parse(docIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	// Check if document exists
	doc, err := h.db.GetDocument(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if doc == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Check if user already has access - allow upgrade requests (view -> edit)
	perm, err := h.db.GetPermission(c.Request.Context(), docID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Parse the requested role from the body first to check if it's an upgrade
	var req models.CreateAccessRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body - defaults will be used
		req = models.CreateAccessRequestRequest{}
	}

	requestedRole := req.RequestedRole
	if requestedRole == "" {
		requestedRole = models.RoleView
	}

	if perm != nil {
		// User already has permission - check if they're requesting an upgrade
		if perm.Role == models.RoleOwner || perm.Role == "edit" {
			// Already has owner or edit permission, no upgrade possible
			c.JSON(http.StatusConflict, gin.H{"error": "You already have edit access or higher to this document"})
			return
		}
		// User has view permission and is requesting upgrade to edit
		if requestedRole != "edit" {
			c.JSON(http.StatusConflict, gin.H{"error": "You already have view access to this document"})
			return
		}
		// Allow the upgrade request
	}

	accessReq, err := h.db.CreateAccessRequest(c.Request.Context(), docID, user.ID, req.RequestedRole, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access request"})
		return
	}

	c.JSON(http.StatusCreated, accessReq)
}

// ListAccessRequests returns all access requests for a document (owner only)
func (h *Handler) ListAccessRequests(c *gin.Context) {
	docIDStr := c.Param("id")
	docID, _ := uuid.Parse(docIDStr)

	requests, err := h.db.ListAccessRequestsByDoc(c.Request.Context(), docID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list access requests"})
		return
	}
	if requests == nil {
		requests = []*models.AccessRequest{}
	}
	c.JSON(http.StatusOK, requests)
}

// UpdateAccessRequest updates an access request status (approve/reject)
func (h *Handler) UpdateAccessRequest(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	reqIDStr := c.Param("id")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	// Get the access request
	accessReq, err := h.db.GetAccessRequest(c.Request.Context(), reqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if accessReq == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Access request not found"})
		return
	}

	// Check if user is the document owner
	perm, err := h.db.GetPermission(c.Request.Context(), accessReq.DocID, user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if perm == nil || perm.Role != models.RoleOwner {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only document owner can manage access requests"})
		return
	}

	var req models.UpdateAccessRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update the request status
	updated, err := h.db.UpdateAccessRequestStatus(c.Request.Context(), reqID, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update access request"})
		return
	}

	// If approved, grant permission
	if req.Status == models.AccessRequestApproved {
		// Use granted_role if provided, otherwise use the originally requested role
		role := req.GrantedRole
		if role == "" {
			role = accessReq.RequestedRole
		}
		if role == "" {
			role = models.RoleView
		}
		if err := h.db.SetPermission(c.Request.Context(), accessReq.DocID, accessReq.RequesterID, role); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant permission"})
			return
		}
	}

	c.JSON(http.StatusOK, updated)
}

// ListMyPendingAccessRequests returns all pending access requests for documents owned by the current user
func (h *Handler) ListMyPendingAccessRequests(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	requests, err := h.db.ListPendingAccessRequestsForOwner(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list access requests"})
		return
	}
	if requests == nil {
		requests = []*models.AccessRequest{}
	}
	c.JSON(http.StatusOK, requests)
}

// ========== Folder Handlers ==========

// CreateFolder creates a new folder
func (h *Handler) CreateFolder(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var req models.CreateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	folder, err := h.db.CreateFolder(c.Request.Context(), req.Name, user.ID, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

// GetFolderContents returns folders and documents in a folder (or root)
func (h *Handler) GetFolderContents(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	var folderID *uuid.UUID
	if folderIDStr := c.Query("folder_id"); folderIDStr != "" {
		id, err := uuid.Parse(folderIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
			return
		}
		folderID = &id
	}

	contents, err := h.db.GetFolderContents(c.Request.Context(), user.ID, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get folder contents"})
		return
	}

	c.JSON(http.StatusOK, contents)
}

// GetFolderByID returns a folder by its ID
func (h *Handler) GetFolderByID(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	folder, err := h.db.GetFolder(c.Request.Context(), folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get folder"})
		return
	}
	if folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	// Check ownership
	if folder.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	c.JSON(http.StatusOK, folder)
}

// GetFolderPath returns the full path from root to the folder
func (h *Handler) GetFolderPath(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	path, err := h.db.GetFolderPath(c.Request.Context(), folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get folder path"})
		return
	}

	// Verify user owns (or has access to) the folders
	if len(path) > 0 && path[0].OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	c.JSON(http.StatusOK, path)
}

// UpdateFolder updates a folder's name
func (h *Handler) UpdateFolder(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	// Check ownership
	folder, err := h.db.GetFolder(c.Request.Context(), folderID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}
	if folder.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	var req models.UpdateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updated, err := h.db.UpdateFolder(c.Request.Context(), folderID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// DeleteFolder deletes a folder
func (h *Handler) DeleteFolder(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	// Check ownership
	folder, err := h.db.GetFolder(c.Request.Context(), folderID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}
	if folder.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if err := h.db.DeleteFolder(c.Request.Context(), folderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder deleted"})
}

// MoveFolder moves a folder to a new parent
func (h *Handler) MoveFolder(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	folderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
		return
	}

	// Check ownership
	folder, err := h.db.GetFolder(c.Request.Context(), folderID)
	if err != nil || folder == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}
	if folder.OwnerID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	var req models.MoveItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.MoveFolder(c.Request.Context(), folderID, req.FolderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move folder"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder moved"})
}

// MoveDocument moves a document to a folder
func (h *Handler) MoveDocument(c *gin.Context) {
	docID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	var req models.MoveItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.MoveDocument(c.Request.Context(), docID, req.FolderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document moved"})
}

// GetFolderTree returns the complete folder tree for the current user
func (h *Handler) GetFolderTree(c *gin.Context) {
	user := auth.GetUserFromContext(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	tree, err := h.db.GetFolderTree(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get folder tree"})
		return
	}

	if tree == nil {
		tree = []*models.FolderTreeNode{}
	}

	c.JSON(http.StatusOK, tree)
}
