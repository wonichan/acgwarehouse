package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
	"github.com/wonichan/acgwarehouse-backend/internal/repository"
)

// CollectionService provides business logic for collection management
type CollectionService struct {
	repo repository.CollectionRepository
}

// NewCollectionService creates a new CollectionService instance
func NewCollectionService(repo repository.CollectionRepository) *CollectionService {
	return &CollectionService{repo: repo}
}

// CreateCollection creates a new collection with the given name and description
func (s *CollectionService) CreateCollection(ctx context.Context, name, description string) (*domain.Collection, error) {
	// Check if collection with same name already exists
	existing, err := s.repo.FindByName(ctx, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("collection with this name already exists")
	}

	collection := &domain.Collection{
		Name:        name,
		Description: description,
		ImageCount:  0,
	}

	if err := s.repo.Save(ctx, collection); err != nil {
		return nil, err
	}

	return collection, nil
}

// GetCollection retrieves a collection by ID
func (s *CollectionService) GetCollection(ctx context.Context, id int64) (*domain.Collection, error) {
	return s.repo.FindByID(ctx, id)
}

// ListCollections retrieves all collections with pagination
func (s *CollectionService) ListCollections(ctx context.Context, limit, offset int) ([]domain.Collection, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.FindAll(ctx, limit, offset)
}

// UpdateCollection updates a collection's name and description
func (s *CollectionService) UpdateCollection(ctx context.Context, id int64, name, description string) (*domain.Collection, error) {
	collection, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if new name conflicts with existing collection
	if name != collection.Name {
		existing, err := s.repo.FindByName(ctx, name)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if existing != nil && existing.ID != id {
			return nil, errors.New("collection with this name already exists")
		}
	}

	collection.Name = name
	collection.Description = description

	if err := s.repo.Update(ctx, collection); err != nil {
		return nil, err
	}

	return collection, nil
}

// DeleteCollection deletes a collection by ID
func (s *CollectionService) DeleteCollection(ctx context.Context, id int64) error {
	// Verify collection exists
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repo.Delete(ctx, id)
}

// AddImageToCollection adds an image to a collection and updates the cover if needed
func (s *CollectionService) AddImageToCollection(ctx context.Context, collectionID, imageID int64) error {
	// Verify collection exists
	_, err := s.repo.FindByID(ctx, collectionID)
	if err != nil {
		return err
	}

	// Add image to collection
	if err := s.repo.AddImage(ctx, collectionID, imageID); err != nil {
		return err
	}

	// Auto-update cover if this is the first image
	count, err := s.repo.CountImages(ctx, collectionID)
	if err != nil {
		return err
	}
	if count == 1 {
		// This is the first image, set it as cover
		return s.repo.UpdateCover(ctx, collectionID, imageID)
	}

	return nil
}

// RemoveImageFromCollection removes an image from a collection and updates cover if needed
func (s *CollectionService) RemoveImageFromCollection(ctx context.Context, collectionID, imageID int64) error {
	// Verify collection exists
	collection, err := s.repo.FindByID(ctx, collectionID)
	if err != nil {
		return err
	}

	// Remove image from collection
	if err := s.repo.RemoveImage(ctx, collectionID, imageID); err != nil {
		return err
	}

	// If removed image was the cover, auto-update cover
	if collection.CoverImageID != nil && *collection.CoverImageID == imageID {
		return s.AutoUpdateCover(ctx, collectionID)
	}

	return nil
}

// SetCoverImage sets the cover image for a collection
func (s *CollectionService) SetCoverImage(ctx context.Context, collectionID, imageID int64) error {
	// Verify collection exists
	_, err := s.repo.FindByID(ctx, collectionID)
	if err != nil {
		return err
	}

	return s.repo.UpdateCover(ctx, collectionID, imageID)
}

// AutoUpdateCover automatically sets the cover to the most recently added image
func (s *CollectionService) AutoUpdateCover(ctx context.Context, collectionID int64) error {
	latestImageID, err := s.repo.GetLatestImageID(ctx, collectionID)
	if err != nil {
		return err
	}

	if latestImageID == nil {
		// No images in collection, clear cover
		collection, err := s.repo.FindByID(ctx, collectionID)
		if err != nil {
			return err
		}
		collection.CoverImageID = nil
		return s.repo.Update(ctx, collection)
	}

	return s.repo.UpdateCover(ctx, collectionID, *latestImageID)
}

// GetCollectionImages retrieves images in a collection with pagination
func (s *CollectionService) GetCollectionImages(ctx context.Context, collectionID int64, limit, offset int) ([]domain.Image, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.FindImagesByCollection(ctx, collectionID, limit, offset)
}

// CountCollections returns the total number of collections
func (s *CollectionService) CountCollections(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
