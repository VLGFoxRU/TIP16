package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"example.com/pz16-integration/internal/models"
)

// NoteRepo — репозиторий для работы с заметками в PostgreSQL
type NoteRepo struct {
	DB *sql.DB
}

// Create создает новую заметку в БД и устанавливает ей ID
func (r NoteRepo) Create(ctx context.Context, n *models.Note) error {
	query := `
		INSERT INTO notes (title, content)
		VALUES ($1, $2)
		RETURNING id, created_at, updated_at
	`

	err := r.DB.QueryRowContext(ctx, query, n.Title, n.Content).
		Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

// GetByID получает заметку по ID
func (r NoteRepo) GetByID(ctx context.Context, id int64) (*models.Note, error) {
	query := `
		SELECT id, title, content, created_at, updated_at
		FROM notes
		WHERE id = $1
	`

	var n models.Note
	err := r.DB.QueryRowContext(ctx, query, id).
		Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("note not found")
		}
		return nil, fmt.Errorf("failed to get note: %w", err)
	}

	return &n, nil
}

// GetAll получает все заметки (с простой пагинацией)
func (r NoteRepo) GetAll(ctx context.Context, offset, limit int) ([]models.Note, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, title, content, created_at, updated_at
		FROM notes
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.DB.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var n models.Note
		if err := rows.Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// Update обновляет заметку
func (r NoteRepo) Update(ctx context.Context, id int64, title, content string) (*models.Note, error) {
	query := `
		UPDATE notes
		SET title = $1, content = $2
		WHERE id = $3
		RETURNING id, title, content, created_at, updated_at
	`

	var n models.Note
	err := r.DB.QueryRowContext(ctx, query, title, content, id).
		Scan(&n.ID, &n.Title, &n.Content, &n.CreatedAt, &n.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("note not found")
		}
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	return &n, nil
}

// Delete удаляет заметку по ID
func (r NoteRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM notes WHERE id = $1`

	result, err := r.DB.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.New("note not found")
	}

	return nil
}
