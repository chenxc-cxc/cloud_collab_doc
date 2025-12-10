package db

import (
	"context"
	"encoding/json"
	"fmt"
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

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

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
		SELECT id, email, name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
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
	var user models.User
	err := db.pool.QueryRow(ctx, `
		SELECT id, email, name, COALESCE(avatar_url, ''), created_at, updated_at
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (db *DB) CreateUser(ctx context.Context, email, name string) (*models.User, error) {
	var user models.User
	err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, name)
		VALUES ($1, $2)
		RETURNING id, email, name, COALESCE(avatar_url, ''), created_at, updated_at
	`, email, name).Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
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
		SELECT d.id, d.title, d.owner_id, d.created_at, d.updated_at,
		       u.id, u.email, u.name, COALESCE(u.avatar_url, '')
		FROM documents d
		JOIN users u ON d.owner_id = u.id
		WHERE d.id = $1
	`, id).Scan(
		&doc.ID, &doc.Title, &doc.OwnerID, &doc.CreatedAt, &doc.UpdatedAt,
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

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &doc, nil
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

// SaveSnapshot saves a new snapshot for a document
func (db *DB) SaveSnapshot(ctx context.Context, docID uuid.UUID, data []byte) (*models.DocSnapshot, error) {
	var snapshot models.DocSnapshot
	err := db.pool.QueryRow(ctx, `
		INSERT INTO doc_snapshots (doc_id, version, snapshot)
		SELECT $1, COALESCE(MAX(version), 0) + 1, $2
		FROM doc_snapshots WHERE doc_id = $1
		RETURNING doc_id, version, snapshot, created_at
	`, docID, data).Scan(&snapshot.DocID, &snapshot.Version, &snapshot.Snapshot, &snapshot.CreatedAt)
	if err != nil {
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

// SaveSnapshotBase64 saves a new snapshot for a document from base64 encoded data
func (db *DB) SaveSnapshotBase64(ctx context.Context, docID uuid.UUID, base64Data string) (*models.DocSnapshot, error) {
	var snapshot models.DocSnapshot
	// Use PostgreSQL's decode function to convert base64 to bytea
	err := db.pool.QueryRow(ctx, `
		INSERT INTO doc_snapshots (doc_id, version, snapshot)
		SELECT $1, COALESCE(MAX(version), 0) + 1, decode($2, 'base64')
		FROM doc_snapshots WHERE doc_id = $1
		RETURNING doc_id, version, snapshot, created_at
	`, docID, base64Data).Scan(&snapshot.DocID, &snapshot.Version, &snapshot.Snapshot, &snapshot.CreatedAt)
	if err != nil {
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
	var selectionJSON []byte
	if selection != nil {
		selectionJSON, _ = json.Marshal(selection)
	}

	var comment models.Comment
	err := db.pool.QueryRow(ctx, `
		INSERT INTO comments (doc_id, user_id, content, selection, parent_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, doc_id, user_id, content, selection, resolved, parent_id, created_at, updated_at
	`, docID, userID, content, selectionJSON, parentID).Scan(
		&comment.ID, &comment.DocID, &comment.UserID, &comment.Content, &selectionJSON,
		&comment.Resolved, &comment.ParentID, &comment.CreatedAt, &comment.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if selectionJSON != nil {
		json.Unmarshal(selectionJSON, &comment.Selection)
	}
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
