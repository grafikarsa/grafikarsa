package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
)

// TagService handles tag business logic
type TagService struct {
	tagRepo *repository.TagRepository
}

// NewTagService creates a new TagService
func NewTagService(tagRepo *repository.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

// ListTags lists all tags
func (s *TagService) ListTags(ctx context.Context, search string) ([]domain.Tag, error) {
	tags, err := s.tagRepo.GetAll(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// ListTagsWithCount lists tags with portfolio count (for admin)
func (s *TagService) ListTagsWithCount(ctx context.Context) ([]domain.Tag, error) {
	tags, err := s.tagRepo.GetAllWithCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// CreateTag creates a new tag (admin)
func (s *TagService) CreateTag(ctx context.Context, name string) (*domain.Tag, error) {
	// Check if name exists
	exists, _ := s.tagRepo.NameExists(ctx, name, nil)
	if exists {
		return nil, ErrTagExists
	}

	now := time.Now()
	tag := &domain.Tag{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}

	return tag, nil
}

// UpdateTag updates a tag (admin)
func (s *TagService) UpdateTag(ctx context.Context, id uuid.UUID, name string) (*domain.Tag, error) {
	tag, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if tag == nil {
		return nil, ErrTagNotFound
	}

	// Check if name exists (excluding current)
	exists, _ := s.tagRepo.NameExists(ctx, name, &id)
	if exists {
		return nil, ErrTagExists
	}

	tag.Name = name
	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}

	return tag, nil
}

// DeleteTag deletes a tag (admin)
func (s *TagService) DeleteTag(ctx context.Context, id uuid.UUID) error {
	tag, err := s.tagRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if tag == nil {
		return ErrTagNotFound
	}

	if err := s.tagRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	return nil
}

// Tag service errors
var (
	ErrTagNotFound = fmt.Errorf("tag not found")
	ErrTagExists   = fmt.Errorf("tag already exists")
)
