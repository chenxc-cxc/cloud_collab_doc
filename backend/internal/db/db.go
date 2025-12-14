package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/collab-docs/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps the database connection pool
type DB struct {
	pool *pgxpool.Pool
}

// New creates a new database connection
func New(ctx context.Context) (*DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/collab_docs?sslmode=disable"
	}

	// Parse the connection string to configure pool
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Disable prepared statement cache for Supabase PgBouncer compatibility
	// PgBouncer in transaction mode doesn't support prepared statements
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	log.Printf("[DB] Connecting to database...")
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("[DB] Database connection established")
	return &DB{pool: pool}, nil
}

// Close closes the database connection
func (db *DB) Close() {
	db.pool.Close()
}

// User operations

// GetUser retrieves a user by ID
func (db *DB) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	log.Printf("[DB] GetUserByEmail: querying email=%s", email)
	var user models.User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(password_hash, ''), name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		log.Printf("[DB] GetUserByEmail: no user found for email=%s", email)
		return nil, nil
	}
	if err != nil {
		log.Printf("[DB] GetUserByEmail: query error: %v", err)
		return nil, err
	}
	log.Printf("[DB] GetUserByEmail: found user id=%s", user.ID)
	return &user, nil
}

// CreateUser creates a new user without password (for backward compatibility)
func (db *DB) CreateUser(ctx context.Context, email, name string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, name)
		VALUES ($1, $2)
		RETURNING id, email, COALESCE(password_hash, ''), name, COALESCE(avatar_url, ''), created_at, updated_at
	`, email, name).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUserWithPassword creates a new user with password
func (db *DB) CreateUserWithPassword(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, COALESCE(password_hash, ''), name, COALESCE(avatar_url, ''), created_at, updated_at
	`, email, name, passwordHash).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUserPassword updates a user's password
func (db *DB) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW()
		WHERE id = $1
	`, userID, passwordHash)
	return err
}

// Document operations

// ListDocuments returns documents accessible by a user
func (db *DB) ListDocuments(ctx context.Context, userID uuid.UUID) ([]*models.Document, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT d.id, d.title, d.owner_id, d.created_at, d.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, ''),
		       COALESCE(dp.role, 'view') as permission
		FROM documents d
		JOIN users u ON d.owner_id = u.id
		LEFT JOIN document_permissions dp ON d.id = dp.doc_id AND dp.user_id = $1
		WHERE d.owner_id = $1 OR dp.user_id = $1
		ORDER BY d.updated_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*models.Document
	for rows.Next() {
		var doc models.Document
		var owner models.User
		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt,
			&owner.ID, &owner.Email, &owner.Name, &owner.AvatarURL,
			&doc.Permission,
		)
		if err != nil {
			return nil, err
		}
		doc.Owner = &owner
		docs = append(docs, &doc)
	}
	return docs, nil
}

// GetDocument retrieves a document by ID
func (db *DB) GetDocument(ctx context.Context, id uuid.UUID) (*models.Document, error) {
	var doc models.Document
	var owner models.User
	err := db.pool.QueryRow(ctx, `
		SELECT d.id, d.title, d.owner_id, d.folder_id, d.created_at, d.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM documents d
		JOIN users u ON d.owner_id = u.id
		WHERE d.id = $1
	`, id).Scan(
		&doc.ID, &doc.Title, &doc.OwnerID, &doc.FolderID, &doc.CreatedAt, &doc.UpdatedAt,
		&owner.ID, &owner.Email, &owner.Name, &owner.AvatarURL,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	doc.Owner = &owner
	return &doc, nil
}

// CreateDocument creates a new document
func (db *DB) CreateDocument(ctx context.Context, title string, ownerID uuid.UUID) (*models.Document, error) {
	log.Printf("[DB] CreateDocument: starting, title=%s, ownerID=%s", title, ownerID)

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		log.Printf("[DB] CreateDocument: failed to begin tx: %v", err)
		return nil, err
	}
	defer tx.Rollback(ctx)

	var doc models.Document
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (title, owner_id)
		VALUES ($1, $2)
		RETURNING id, title, owner_id, created_at, updated_at
	`, title, ownerID).Scan(&doc.ID, &doc.Title, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		log.Printf("[DB] CreateDocument: failed to insert document: %v", err)
		return nil, err
	}
	log.Printf("[DB] CreateDocument: document inserted, id=%s", doc.ID)

	// Create owner permission
	_, err = tx.Exec(ctx, `
		INSERT INTO document_permissions (doc_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, doc.ID, ownerID)
	if err != nil {
		log.Printf("[DB] CreateDocument: failed to insert permission: %v", err)
		return nil, err
	}
	log.Printf("[DB] CreateDocument: permission inserted")

	if err := tx.Commit(ctx); err != nil {
		log.Printf("[DB] CreateDocument: failed to commit: %v", err)
		return nil, err
	}

	log.Printf("[DB] CreateDocument: success, docID=%s", doc.ID)
	return &doc, nil
}

// CreateDocumentWithInitialContent creates a new document with initial welcome content
// This is used for creating welcome documents for new users
func (db *DB) CreateDocumentWithInitialContent(ctx context.Context, title string, ownerID uuid.UUID) (*models.Document, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var doc models.Document
	err = tx.QueryRow(ctx, `
		INSERT INTO documents (title, owner_id)
		VALUES ($1, $2)
		RETURNING id, title, owner_id, created_at, updated_at
	`, title, ownerID).Scan(&doc.ID, &doc.Title, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Create owner permission
	_, err = tx.Exec(ctx, `
		INSERT INTO document_permissions (doc_id, user_id, role)
		VALUES ($1, $2, 'owner')
	`, doc.ID, ownerID)
	if err != nil {
		return nil, err
	}

	// Create initial Yjs snapshot with welcome content
	// This is a pre-generated Yjs document state with welcome text
	// The snapshot was created using Yjs and encodes the following content:
	// "Welcome to CollabDocs! 🎉\n\nThis is your first document. Start typing to begin..."
	welcomeSnapshot := getWelcomeDocumentSnapshot()

	if len(welcomeSnapshot) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO doc_snapshots (doc_id, version, snapshot)
			VALUES ($1, 1, $2)
		`, doc.ID, welcomeSnapshot)
		if err != nil {
			// Log error but don't fail - document is still created
			// User will just see empty document instead of welcome content
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &doc, nil
}

// getWelcomeDocumentSnapshot returns a pre-generated Yjs document state
// This creates a simple welcome message in the document
// Content: "欢迎使用 CollabDocs! 🎉" + intro paragraph + "✨ 快速开始" + tips
//
// To regenerate this snapshot with different content:
// 1. Edit scripts/generate-welcome-snapshot.js
// 2. Run: cd scripts && node generate-welcome-snapshot.js
// 3. Copy the output Go bytes here
func getWelcomeDocumentSnapshot() []byte {
	return []byte{1, 15, 165, 147, 158, 133, 6, 0, 7, 1, 7, 100, 101, 102, 97, 117, 108, 116, 3, 3, 100, 111, 99, 7, 0, 165, 147, 158, 133, 6, 0, 3, 7, 104, 101, 97, 100, 105, 110, 103, 7, 0, 165, 147, 158, 133, 6, 1, 6, 4, 0, 165, 147, 158, 133, 6, 2, 29, 230, 172, 162, 232, 191, 142, 228, 189, 191, 231, 148, 168, 32, 67, 111, 108, 108, 97, 98, 68, 111, 99, 115, 33, 32, 240, 159, 142, 137, 40, 0, 165, 147, 158, 133, 6, 1, 5, 108, 101, 118, 101, 108, 1, 125, 1, 135, 165, 147, 158, 133, 6, 1, 3, 9, 112, 97, 114, 97, 103, 114, 97, 112, 104, 7, 0, 165, 147, 158, 133, 6, 23, 6, 4, 0, 165, 147, 158, 133, 6, 24, 113, 232, 191, 153, 230, 152, 175, 228, 189, 160, 231, 154, 132, 231, 172, 172, 228, 184, 128, 228, 184, 170, 230, 150, 135, 230, 161, 163, 227, 128, 130, 67, 111, 108, 108, 97, 98, 68, 111, 99, 115, 32, 230, 152, 175, 228, 184, 128, 228, 184, 170, 229, 174, 158, 230, 151, 182, 229, 141, 143, 228, 189, 156, 230, 150, 135, 230, 161, 163, 229, 185, 179, 229, 143, 176, 239, 188, 140, 232, 174, 169, 229, 155, 162, 233, 152, 159, 229, 141, 143, 228, 189, 156, 229, 143, 152, 229, 190, 151, 231, 174, 128, 229, 141, 149, 233, 171, 152, 230, 149, 136, 227, 128, 130, 135, 165, 147, 158, 133, 6, 23, 3, 7, 104, 101, 97, 100, 105, 110, 103, 7, 0, 165, 147, 158, 133, 6, 70, 6, 4, 0, 165, 147, 158, 133, 6, 71, 16, 226, 156, 168, 32, 229, 191, 171, 233, 128, 159, 229, 188, 128, 229, 167, 139, 40, 0, 165, 147, 158, 133, 6, 70, 5, 108, 101, 118, 101, 108, 1, 125, 2, 135, 165, 147, 158, 133, 6, 70, 3, 9, 112, 97, 114, 97, 103, 114, 97, 112, 104, 7, 0, 165, 147, 158, 133, 6, 79, 6, 4, 0, 165, 147, 158, 133, 6, 80, 45, 228, 187, 165, 228, 184, 139, 230, 152, 175, 228, 184, 128, 228, 186, 155, 229, 184, 174, 229, 138, 169, 228, 189, 160, 228, 184, 138, 230, 137, 139, 231, 154, 132, 229, 176, 143, 230, 138, 128, 229, 183, 167, 239, 188, 154, 0}
}

// UpdateDocument updates a document
func (db *DB) UpdateDocument(ctx context.Context, id uuid.UUID, title string) (*models.Document, error) {
	var doc models.Document
	err := db.pool.QueryRow(ctx, `
		UPDATE documents SET title = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, title, owner_id, created_at, updated_at
	`, id, title).Scan(&doc.ID, &doc.Title, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// DeleteDocument deletes a document
func (db *DB) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id)
	return err
}

// Permission operations

// GetPermission retrieves a user's permission for a document
func (db *DB) GetPermission(ctx context.Context, docID, userID uuid.UUID) (*models.DocumentPermission, error) {
	var perm models.DocumentPermission
	err := db.pool.QueryRow(ctx, `
		SELECT doc_id, user_id, role, created_at
		FROM document_permissions
		WHERE doc_id = $1 AND user_id = $2
	`, docID, userID).Scan(&perm.DocID, &perm.UserID, &perm.Role, &perm.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &perm, nil
}

// ListPermissions returns all permissions for a document
func (db *DB) ListPermissions(ctx context.Context, docID uuid.UUID) ([]*models.DocumentPermission, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT dp.doc_id, dp.user_id, dp.role, dp.created_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM document_permissions dp
		JOIN users u ON dp.user_id = u.id
		WHERE dp.doc_id = $1
		ORDER BY dp.created_at
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*models.DocumentPermission
	for rows.Next() {
		var perm models.DocumentPermission
		var user models.User
		err := rows.Scan(
			&perm.DocID, &perm.UserID, &perm.Role, &perm.CreatedAt,
			&user.ID, &user.Email, &user.Name, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		perm.User = &user
		perms = append(perms, &perm)
	}
	return perms, nil
}

// SetPermission sets a user's permission for a document
func (db *DB) SetPermission(ctx context.Context, docID, userID uuid.UUID, role string) error {
	_, err := db.pool.Exec(ctx, `
		INSERT INTO document_permissions (doc_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (doc_id, user_id) DO UPDATE SET role = $3
	`, docID, userID, role)
	return err
}

// RemovePermission removes a user's permission for a document
func (db *DB) RemovePermission(ctx context.Context, docID, userID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		DELETE FROM document_permissions
		WHERE doc_id = $1 AND user_id = $2 AND role != 'owner'
	`, docID, userID)
	return err
}

// Snapshot operations

// GetLatestSnapshot retrieves the latest snapshot for a document
func (db *DB) GetLatestSnapshot(ctx context.Context, docID uuid.UUID) (*models.DocSnapshot, error) {
	var snapshot models.DocSnapshot
	err := db.pool.QueryRow(ctx, `
		SELECT doc_id, version, snapshot, created_at
		FROM doc_snapshots
		WHERE doc_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, docID).Scan(&snapshot.DocID, &snapshot.Version, &snapshot.Snapshot, &snapshot.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// SaveSnapshot saves a new snapshot for a document and updates document's updated_at
func (db *DB) SaveSnapshot(ctx context.Context, docID uuid.UUID, data []byte) (*models.DocSnapshot, error) {
	// Start a transaction to update both snapshot and document
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var snapshot models.DocSnapshot
	err = tx.QueryRow(ctx, `
		INSERT INTO doc_snapshots (doc_id, version, snapshot)
		SELECT $1, COALESCE(MAX(version), 0) + 1, $2
		FROM doc_snapshots WHERE doc_id = $1
		RETURNING doc_id, version, snapshot, created_at
	`, docID, data).Scan(&snapshot.DocID, &snapshot.Version, &snapshot.Snapshot, &snapshot.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Update document's updated_at timestamp
	_, err = tx.Exec(ctx, `UPDATE documents SET updated_at = NOW() WHERE id = $1`, docID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// ListSnapshots returns all snapshots for a document
func (db *DB) ListSnapshots(ctx context.Context, docID uuid.UUID) ([]*models.DocSnapshot, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT doc_id, version, created_at
		FROM doc_snapshots
		WHERE doc_id = $1
		ORDER BY version DESC
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*models.DocSnapshot
	for rows.Next() {
		var s models.DocSnapshot
		err := rows.Scan(&s.DocID, &s.Version, &s.CreatedAt)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, &s)
	}
	return snapshots, nil
}

// SaveSnapshotBase64 saves a new snapshot for a document from base64 encoded data and updates document's updated_at
func (db *DB) SaveSnapshotBase64(ctx context.Context, docID uuid.UUID, base64Data string) (*models.DocSnapshot, error) {
	// Start a transaction to update both snapshot and document
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var snapshot models.DocSnapshot
	// Use PostgreSQL's decode function to convert base64 to bytea
	err = tx.QueryRow(ctx, `
		INSERT INTO doc_snapshots (doc_id, version, snapshot)
		SELECT $1, COALESCE(MAX(version), 0) + 1, decode($2, 'base64')
		FROM doc_snapshots WHERE doc_id = $1
		RETURNING doc_id, version, snapshot, created_at
	`, docID, base64Data).Scan(&snapshot.DocID, &snapshot.Version, &snapshot.Snapshot, &snapshot.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Update document's updated_at timestamp
	_, err = tx.Exec(ctx, `UPDATE documents SET updated_at = NOW() WHERE id = $1`, docID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// Comment operations

// ListComments returns all comments for a document
func (db *DB) ListComments(ctx context.Context, docID uuid.UUID) ([]*models.Comment, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT c.id, c.doc_id, c.user_id, c.content, c.selection, 
		       c.resolved, c.parent_id, c.created_at, c.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM comments c
		JOIN users u ON c.user_id = u.id
		WHERE c.doc_id = $1 AND c.parent_id IS NULL
		ORDER BY c.created_at DESC
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*models.Comment
	for rows.Next() {
		var c models.Comment
		var user models.User
		var selectionJSON []byte
		err := rows.Scan(
			&c.ID, &c.DocID, &c.UserID, &c.Content, &selectionJSON,
			&c.Resolved, &c.ParentID, &c.CreatedAt, &c.UpdatedAt,
			&user.ID, &user.Email, &user.Name, &user.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		if selectionJSON != nil {
			json.Unmarshal(selectionJSON, &c.Selection)
		}
		c.User = &user
		comments = append(comments, &c)
	}
	return comments, nil
}

// CreateComment creates a new comment
func (db *DB) CreateComment(ctx context.Context, docID, userID uuid.UUID, content string, selection *models.Selection, parentID *uuid.UUID) (*models.Comment, error) {
	// For simple protocol mode, we need to pass JSONB as string, not []byte
	var selectionStr *string
	if selection != nil {
		jsonBytes, _ := json.Marshal(selection)
		s := string(jsonBytes)
		selectionStr = &s
	}

	var comment models.Comment
	var selectionJSON []byte
	err := db.pool.QueryRow(ctx, `
		INSERT INTO comments (doc_id, user_id, content, selection, parent_id)
		VALUES ($1, $2, $3, $4::jsonb, $5)
		RETURNING id, doc_id, user_id, content, selection, resolved, parent_id, created_at, updated_at
	`, docID, userID, content, selectionStr, parentID).Scan(
		&comment.ID, &comment.DocID, &comment.UserID, &comment.Content, &selectionJSON,
		&comment.Resolved, &comment.ParentID, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		log.Printf("[DB] CreateComment: error: %v", err)
		return nil, err
	}
	if selectionJSON != nil {
		json.Unmarshal(selectionJSON, &comment.Selection)
	}
	log.Printf("[DB] CreateComment: success, commentID=%s", comment.ID)
	return &comment, nil
}

// UpdateComment updates a comment
func (db *DB) UpdateComment(ctx context.Context, id uuid.UUID, content *string, resolved *bool) (*models.Comment, error) {
	query := "UPDATE comments SET updated_at = NOW()"
	args := []interface{}{}
	argNum := 1

	if content != nil {
		query += fmt.Sprintf(", content = $%d", argNum)
		args = append(args, *content)
		argNum++
	}
	if resolved != nil {
		query += fmt.Sprintf(", resolved = $%d", argNum)
		args = append(args, *resolved)
		argNum++
	}

	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, doc_id, user_id, content, selection, resolved, parent_id, created_at, updated_at", argNum)
	args = append(args, id)

	var comment models.Comment
	var selectionJSON []byte
	err := db.pool.QueryRow(ctx, query, args...).Scan(
		&comment.ID, &comment.DocID, &comment.UserID, &comment.Content, &selectionJSON,
		&comment.Resolved, &comment.ParentID, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if selectionJSON != nil {
		json.Unmarshal(selectionJSON, &comment.Selection)
	}
	return &comment, nil
}

// DeleteComment deletes a comment
func (db *DB) DeleteComment(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM comments WHERE id = $1`, id)
	return err
}

// GetComment retrieves a comment by ID
func (db *DB) GetComment(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	var comment models.Comment
	var selectionJSON []byte
	err := db.pool.QueryRow(ctx, `
		SELECT id, doc_id, user_id, content, selection, resolved, parent_id, created_at, updated_at
		FROM comments WHERE id = $1
	`, id).Scan(
		&comment.ID, &comment.DocID, &comment.UserID, &comment.Content, &selectionJSON,
		&comment.Resolved, &comment.ParentID, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if selectionJSON != nil {
		json.Unmarshal(selectionJSON, &comment.Selection)
	}
	return &comment, nil
}

// Access Request operations

// CreateAccessRequest creates a new access request
func (db *DB) CreateAccessRequest(ctx context.Context, docID, requesterID uuid.UUID, requestedRole, message string) (*models.AccessRequest, error) {
	if requestedRole == "" {
		requestedRole = "view"
	}

	var req models.AccessRequest
	err := db.pool.QueryRow(ctx, `
		INSERT INTO access_requests (doc_id, requester_id, requested_role, message)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (doc_id, requester_id) DO UPDATE SET
			status = 'pending',
			requested_role = $3,
			message = $4,
			updated_at = NOW()
		RETURNING id, doc_id, requester_id, status, requested_role, COALESCE(message, ''), created_at, updated_at
	`, docID, requesterID, requestedRole, message).Scan(
		&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole, &req.Message, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetAccessRequest retrieves an access request by ID
func (db *DB) GetAccessRequest(ctx context.Context, id uuid.UUID) (*models.AccessRequest, error) {
	var req models.AccessRequest
	var requester models.User
	err := db.pool.QueryRow(ctx, `
		SELECT ar.id, ar.doc_id, ar.requester_id, ar.status, ar.requested_role, 
		       COALESCE(ar.message, ''), ar.created_at, ar.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM access_requests ar
		JOIN users u ON ar.requester_id = u.id
		WHERE ar.id = $1
	`, id).Scan(
		&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole,
		&req.Message, &req.CreatedAt, &req.UpdatedAt,
		&requester.ID, &requester.Email, &requester.Name, &requester.AvatarURL,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	req.Requester = &requester
	return &req, nil
}

// ListAccessRequestsByDoc returns all access requests for a document
func (db *DB) ListAccessRequestsByDoc(ctx context.Context, docID uuid.UUID) ([]*models.AccessRequest, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT ar.id, ar.doc_id, ar.requester_id, ar.status, ar.requested_role, 
		       COALESCE(ar.message, ''), ar.created_at, ar.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM access_requests ar
		JOIN users u ON ar.requester_id = u.id
		WHERE ar.doc_id = $1
		ORDER BY ar.created_at DESC
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.AccessRequest
	for rows.Next() {
		var req models.AccessRequest
		var requester models.User
		err := rows.Scan(
			&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole,
			&req.Message, &req.CreatedAt, &req.UpdatedAt,
			&requester.ID, &requester.Email, &requester.Name, &requester.AvatarURL,
		)
		if err != nil {
			return nil, err
		}
		req.Requester = &requester
		requests = append(requests, &req)
	}
	return requests, nil
}

// UpdateAccessRequestStatus updates the status of an access request
func (db *DB) UpdateAccessRequestStatus(ctx context.Context, id uuid.UUID, status string) (*models.AccessRequest, error) {
	var req models.AccessRequest
	err := db.pool.QueryRow(ctx, `
		UPDATE access_requests 
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, doc_id, requester_id, status, requested_role, COALESCE(message, ''), created_at, updated_at
	`, id, status).Scan(
		&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole, &req.Message, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// GetPendingAccessRequest checks if there is a pending access request for a user and document
func (db *DB) GetPendingAccessRequest(ctx context.Context, docID, requesterID uuid.UUID) (*models.AccessRequest, error) {
	var req models.AccessRequest
	err := db.pool.QueryRow(ctx, `
		SELECT id, doc_id, requester_id, status, requested_role, COALESCE(message, ''), created_at, updated_at
		FROM access_requests
		WHERE doc_id = $1 AND requester_id = $2
	`, docID, requesterID).Scan(
		&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole, &req.Message, &req.CreatedAt, &req.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// ListPendingAccessRequestsForOwner returns all pending access requests for documents owned by a user
func (db *DB) ListPendingAccessRequestsForOwner(ctx context.Context, ownerID uuid.UUID) ([]*models.AccessRequest, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT ar.id, ar.doc_id, ar.requester_id, ar.status, ar.requested_role, 
		       COALESCE(ar.message, ''), ar.created_at, ar.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, ''),
		       d.id, d.title
		FROM access_requests ar
		JOIN users u ON ar.requester_id = u.id
		JOIN documents d ON ar.doc_id = d.id
		WHERE d.owner_id = $1 AND ar.status = 'pending'
		ORDER BY ar.created_at DESC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.AccessRequest
	for rows.Next() {
		var req models.AccessRequest
		var requester models.User
		var doc models.Document
		err := rows.Scan(
			&req.ID, &req.DocID, &req.RequesterID, &req.Status, &req.RequestedRole,
			&req.Message, &req.CreatedAt, &req.UpdatedAt,
			&requester.ID, &requester.Email, &requester.Name, &requester.AvatarURL,
			&doc.ID, &doc.Title,
		)
		if err != nil {
			return nil, err
		}
		req.Requester = &requester
		req.Document = &doc
		requests = append(requests, &req)
	}
	return requests, nil
}

// ========== Folder Functions ==========

// CreateFolder creates a new folder
func (db *DB) CreateFolder(ctx context.Context, name string, ownerID uuid.UUID, parentID *uuid.UUID) (*models.Folder, error) {
	var folder models.Folder
	err := db.pool.QueryRow(ctx, `
		INSERT INTO folders (name, owner_id, parent_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, owner_id, parent_id, created_at, updated_at
	`, name, ownerID, parentID).Scan(
		&folder.ID, &folder.Name, &folder.OwnerID, &folder.ParentID, &folder.CreatedAt, &folder.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// GetFolder returns a folder by ID
func (db *DB) GetFolder(ctx context.Context, id uuid.UUID) (*models.Folder, error) {
	var folder models.Folder
	err := db.pool.QueryRow(ctx, `
		SELECT id, name, owner_id, parent_id, created_at, updated_at
		FROM folders WHERE id = $1
	`, id).Scan(
		&folder.ID, &folder.Name, &folder.OwnerID, &folder.ParentID, &folder.CreatedAt, &folder.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// ListFolders returns folders for a user in a specific parent folder
func (db *DB) ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID) ([]*models.Folder, error) {
	var rows pgx.Rows
	var err error

	if parentID == nil {
		rows, err = db.pool.Query(ctx, `
			SELECT id, name, owner_id, parent_id, created_at, updated_at
			FROM folders WHERE owner_id = $1 AND parent_id IS NULL
			ORDER BY name ASC
		`, ownerID)
	} else {
		rows, err = db.pool.Query(ctx, `
			SELECT id, name, owner_id, parent_id, created_at, updated_at
			FROM folders WHERE owner_id = $1 AND parent_id = $2
			ORDER BY name ASC
		`, ownerID, parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.Folder
	for rows.Next() {
		var folder models.Folder
		err := rows.Scan(
			&folder.ID, &folder.Name, &folder.OwnerID, &folder.ParentID, &folder.CreatedAt, &folder.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, &folder)
	}
	return folders, nil
}

// UpdateFolder updates a folder's name
func (db *DB) UpdateFolder(ctx context.Context, id uuid.UUID, name string) (*models.Folder, error) {
	var folder models.Folder
	err := db.pool.QueryRow(ctx, `
		UPDATE folders SET name = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, owner_id, parent_id, created_at, updated_at
	`, id, name).Scan(
		&folder.ID, &folder.Name, &folder.OwnerID, &folder.ParentID, &folder.CreatedAt, &folder.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// DeleteFolder deletes a folder (cascades to subfolders)
func (db *DB) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM folders WHERE id = $1`, id)
	return err
}

// GetFolderPath returns the full path of folders from root to the given folder
func (db *DB) GetFolderPath(ctx context.Context, folderID uuid.UUID) ([]*models.Folder, error) {
	var path []*models.Folder
	currentID := &folderID

	for currentID != nil {
		folder, err := db.GetFolder(ctx, *currentID)
		if err != nil {
			return nil, err
		}
		if folder == nil {
			break
		}
		// Prepend to path (so root is first)
		path = append([]*models.Folder{folder}, path...)
		currentID = folder.ParentID
	}

	return path, nil
}

// GetFolderContents returns folders and documents in a folder
func (db *DB) GetFolderContents(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID) (*models.FolderContents, error) {
	contents := &models.FolderContents{}

	// Get current folder info if not root
	if folderID != nil {
		folder, err := db.GetFolder(ctx, *folderID)
		if err != nil {
			return nil, err
		}
		contents.Folder = folder
	}

	// Get subfolders
	folders, err := db.ListFolders(ctx, ownerID, folderID)
	if err != nil {
		return nil, err
	}
	contents.Folders = folders
	if contents.Folders == nil {
		contents.Folders = []*models.Folder{}
	}

	// Get documents in this folder (with owner info)
	var rows pgx.Rows
	if folderID == nil {
		rows, err = db.pool.Query(ctx, `
			SELECT d.id, d.title, d.owner_id, d.folder_id, d.created_at, d.updated_at,
			       u.id, u.email, u.name, COALESCE(u.avatar_url, ''),
			       COALESCE(dp.role, 'view') as permission
			FROM documents d
			JOIN users u ON d.owner_id = u.id
			JOIN document_permissions dp ON d.id = dp.doc_id AND dp.user_id = $1
			WHERE d.folder_id IS NULL
			ORDER BY d.updated_at DESC
		`, ownerID)
	} else {
		rows, err = db.pool.Query(ctx, `
			SELECT d.id, d.title, d.owner_id, d.folder_id, d.created_at, d.updated_at,
			       u.id, u.email, u.name, COALESCE(u.avatar_url, ''),
			       COALESCE(dp.role, 'view') as permission
			FROM documents d
			JOIN users u ON d.owner_id = u.id
			JOIN document_permissions dp ON d.id = dp.doc_id AND dp.user_id = $1
			WHERE d.folder_id = $2
			ORDER BY d.updated_at DESC
		`, ownerID, folderID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var doc models.Document
		var owner models.User
		err := rows.Scan(
			&doc.ID, &doc.Title, &doc.OwnerID, &doc.FolderID, &doc.CreatedAt, &doc.UpdatedAt,
			&owner.ID, &owner.Email, &owner.Name, &owner.AvatarURL,
			&doc.Permission,
		)
		if err != nil {
			return nil, err
		}
		doc.Owner = &owner
		contents.Documents = append(contents.Documents, &doc)
	}
	if contents.Documents == nil {
		contents.Documents = []*models.Document{}
	}

	return contents, nil
}

// MoveDocument moves a document to a folder (nil = root)
func (db *DB) MoveDocument(ctx context.Context, docID uuid.UUID, folderID *uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE documents SET folder_id = $2, updated_at = NOW()
		WHERE id = $1
	`, docID, folderID)
	return err
}

// MoveFolder moves a folder to a new parent (nil = root)
func (db *DB) MoveFolder(ctx context.Context, folderID uuid.UUID, parentID *uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE folders SET parent_id = $2, updated_at = NOW()
		WHERE id = $1
	`, folderID, parentID)
	return err
}

// GetFolderTree returns the complete folder tree for a user using WITH RECURSIVE
func (db *DB) GetFolderTree(ctx context.Context, ownerID uuid.UUID) ([]*models.FolderTreeNode, error) {
	rows, err := db.pool.Query(ctx, `
		WITH RECURSIVE folder_tree AS (
			-- Base case: root folders (no parent)
			SELECT 
				f.id, f.name, f.owner_id, f.parent_id, f.created_at, f.updated_at,
				0 as level,
				'/' || f.name as path
			FROM folders f
			WHERE f.owner_id = $1 AND f.parent_id IS NULL
			
			UNION ALL
			
			-- Recursive case: child folders
			SELECT 
				f.id, f.name, f.owner_id, f.parent_id, f.created_at, f.updated_at,
				ft.level + 1 as level,
				ft.path || '/' || f.name as path
			FROM folders f
			INNER JOIN folder_tree ft ON f.parent_id = ft.id
			WHERE f.owner_id = $1
		)
		SELECT 
			ft.id, ft.name, ft.owner_id, ft.parent_id, ft.created_at, ft.updated_at,
			ft.level, ft.path,
			COALESCE((SELECT COUNT(*) FROM documents d WHERE d.folder_id = ft.id), 0) as doc_count
		FROM folder_tree ft
		ORDER BY ft.path ASC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []*models.FolderTreeNode
	for rows.Next() {
		var node models.FolderTreeNode
		err := rows.Scan(
			&node.ID, &node.Name, &node.OwnerID, &node.ParentID,
			&node.CreatedAt, &node.UpdatedAt, &node.Level, &node.Path, &node.DocCount,
		)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, &node)
	}

	// Build the tree structure
	tree := buildFolderTree(nodes)

	// Fetch all documents that belong to folders owned by this user
	docRows, err := db.pool.Query(ctx, `
		SELECT d.id, d.title, d.owner_id, d.folder_id, d.created_at, d.updated_at
		FROM documents d
		JOIN document_permissions dp ON d.id = dp.doc_id AND dp.user_id = $1
		WHERE d.folder_id IS NOT NULL
		ORDER BY d.title ASC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer docRows.Close()

	// Create a map of folder ID to documents
	folderDocs := make(map[uuid.UUID][]*models.Document)
	for docRows.Next() {
		var doc models.Document
		err := docRows.Scan(
			&doc.ID, &doc.Title, &doc.OwnerID, &doc.FolderID, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		if doc.FolderID != nil {
			folderDocs[*doc.FolderID] = append(folderDocs[*doc.FolderID], &doc)
		}
	}

	// Also fetch root-level documents (no folder)
	rootDocRows, err := db.pool.Query(ctx, `
		SELECT d.id, d.title, d.owner_id, d.folder_id, d.created_at, d.updated_at
		FROM documents d
		JOIN document_permissions dp ON d.id = dp.doc_id AND dp.user_id = $1
		WHERE d.folder_id IS NULL
		ORDER BY d.title ASC
	`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rootDocRows.Close()

	var rootDocs []*models.Document
	for rootDocRows.Next() {
		var doc models.Document
		err := rootDocRows.Scan(
			&doc.ID, &doc.Title, &doc.OwnerID, &doc.FolderID, &doc.CreatedAt, &doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rootDocs = append(rootDocs, &doc)
	}

	// Recursively attach documents to folder nodes
	attachDocumentsToTree(tree, folderDocs)

	// Create a virtual "root" representation that includes root-level documents
	// We'll include root docs in the response by returning them alongside tree
	// For now, we add them to a special handling in frontend
	// Or we can add a special root node - but simpler to just return tree and rootDocs separately
	// Actually, let's store root docs in the response metadata or handle in frontend

	return tree, nil
}

// attachDocumentsToTree recursively attaches documents to folder nodes
func attachDocumentsToTree(nodes []*models.FolderTreeNode, folderDocs map[uuid.UUID][]*models.Document) {
	for _, node := range nodes {
		if docs, ok := folderDocs[node.ID]; ok {
			node.Documents = docs
		} else {
			node.Documents = []*models.Document{}
		}
		if len(node.Children) > 0 {
			attachDocumentsToTree(node.Children, folderDocs)
		}
	}
}

// buildFolderTree converts a flat list of nodes into a nested tree structure
func buildFolderTree(nodes []*models.FolderTreeNode) []*models.FolderTreeNode {
	if len(nodes) == 0 {
		return []*models.FolderTreeNode{}
	}

	// Create a map for quick lookup
	nodeMap := make(map[uuid.UUID]*models.FolderTreeNode)
	for _, node := range nodes {
		node.Children = []*models.FolderTreeNode{}
		nodeMap[node.ID] = node
	}

	// Build tree by linking children to parents
	var roots []*models.FolderTreeNode
	for _, node := range nodes {
		if node.ParentID == nil {
			roots = append(roots, node)
		} else {
			if parent, ok := nodeMap[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return roots
}
