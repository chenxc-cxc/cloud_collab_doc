package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/collab-docs/backend/internal/db"
	"github.com/collab-docs/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ContextKey is a custom type for context keys
type ContextKey string

const (
	// UserContextKey is the key for storing user in context
	UserContextKey ContextKey = "user"
	// PermissionContextKey is the key for storing permission in context
	PermissionContextKey ContextKey = "permission"
)

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(user *models.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret-change-in-production"
	}

	claims := Claims{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "collab-docs",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "local-dev-secret-change-in-production"
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// AuthMiddleware validates JWT tokens and sets user in context
func AuthMiddleware(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		claims, err := ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token: " + err.Error()})
			c.Abort()
			return
		}

		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		user, err := database.GetUser(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			c.Abort()
			return
		}

		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		c.Set(string(UserContextKey), user)
		c.Next()
	}
}

// DevAuthMiddleware is a simplified auth for local development
// It accepts a user ID header for testing
func DevAuthMiddleware(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// First try normal JWT auth
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			AuthMiddleware(database)(c)
			if c.IsAborted() {
				return
			}
			c.Next()
			return
		}

		// For dev: accept X-User-ID header
		userIDStr := c.GetHeader("X-User-ID")
		if userIDStr == "" {
			// Default to Alice for local dev
			userIDStr = "11111111-1111-1111-1111-111111111111"
		}

		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
			c.Abort()
			return
		}

		user, err := database.GetUser(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			c.Abort()
			return
		}

		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		c.Set(string(UserContextKey), user)
		c.Next()
	}
}

// GetUserFromContext retrieves the authenticated user from context
func GetUserFromContext(c *gin.Context) *models.User {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}
	return user.(*models.User)
}

// GetUserFromStdContext retrieves user from standard context
func GetUserFromStdContext(ctx context.Context) *models.User {
	user := ctx.Value(UserContextKey)
	if user == nil {
		return nil
	}
	return user.(*models.User)
}

// RequirePermission middleware checks if user has permission for a document
func RequirePermission(database *db.DB, minRole string) gin.HandlerFunc {
	roleHierarchy := map[string]int{
		models.RoleView:    1,
		models.RoleComment: 2,
		models.RoleEdit:    3,
		models.RoleOwner:   4,
	}

	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			c.Abort()
			return
		}

		docIDStr := c.Param("id")
		docID, err := uuid.Parse(docIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
			c.Abort()
			return
		}

		perm, err := database.GetPermission(c.Request.Context(), docID, user.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			c.Abort()
			return
		}

		if perm == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "No access to this document"})
			c.Abort()
			return
		}

		if roleHierarchy[perm.Role] < roleHierarchy[minRole] {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set(string(PermissionContextKey), perm)
		c.Next()
	}
}
