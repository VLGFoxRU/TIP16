package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"example.com/pz16-integration/internal/models"
	"example.com/pz16-integration/internal/repo"
)

// Service — сервис для работы с заметками
type Service struct {
	Notes repo.NoteRepo
}

// CreateNote создает новую заметку с валидацией
func (s Service) CreateNote(ctx context.Context, input models.CreateNoteInput) (*models.Note, error) {
	// Валидация
	if strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("title cannot be empty")
	}

	if strings.TrimSpace(input.Content) == "" {
		return nil, errors.New("content cannot be empty")
	}

	// Ограничиваем длину
	if len(input.Title) > 255 {
		return nil, errors.New("title too long (max 255 characters)")
	}

	if len(input.Content) > 10000 {
		return nil, errors.New("content too long (max 10000 characters)")
	}

	// Создаем модель и передаем в репозиторий
	note := &models.Note{
		Title:   strings.TrimSpace(input.Title),
		Content: strings.TrimSpace(input.Content),
	}

	if err := s.Notes.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	return note, nil
}

// GetNote получает заметку по ID
func (s Service) GetNote(ctx context.Context, id int64) (*models.Note, error) {
	if id <= 0 {
		return nil, errors.New("invalid note ID")
	}

	return s.Notes.GetByID(ctx, id)
}

// GetNotes получает список заметок с пагинацией
func (s Service) GetNotes(ctx context.Context, offset, limit int) ([]models.Note, error) {
	return s.Notes.GetAll(ctx, offset, limit)
}

// UpdateNote обновляет заметку
func (s Service) UpdateNote(ctx context.Context, id int64, input models.UpdateNoteInput) (*models.Note, error) {
	if id <= 0 {
		return nil, errors.New("invalid note ID")
	}

	// Если оба поля пусты, это ошибка
	if strings.TrimSpace(input.Title) == "" && strings.TrimSpace(input.Content) == "" {
		return nil, errors.New("at least one field must be provided")
	}

	// Получаем текущую заметку
	current, err := s.Notes.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Используем новые значения или старые, если не указаны
	title := input.Title
	if strings.TrimSpace(title) == "" {
		title = current.Title
	}

	content := input.Content
	if strings.TrimSpace(content) == "" {
		content = current.Content
	}

	// Валидируем новые значения
	if len(title) > 255 {
		return nil, errors.New("title too long (max 255 characters)")
	}

	if len(content) > 10000 {
		return nil, errors.New("content too long (max 10000 characters)")
	}

	return s.Notes.Update(ctx, id, title, content)
}

// DeleteNote удаляет заметку
func (s Service) DeleteNote(ctx context.Context, id int64) error {
	if id <= 0 {
		return errors.New("invalid note ID")
	}

	return s.Notes.Delete(ctx, id)
}
