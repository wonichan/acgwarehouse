package repository

import (
	"fmt"
	"time"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

// D1 mapper helpers convert map[string]any rows from the D1 HTTP API
// into domain structs. D1 JSON decodes numbers as float64 and nulls as nil.

func mapImageFromD1(row map[string]any) (*domain.Image, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, fmt.Errorf("image.id: %w", err)
	}
	fileSize, _ := toInt64(row["file_size"])
	width, _ := toInt(row["width"])
	height, _ := toInt(row["height"])
	phash, _ := toInt64(row["phash"])
	phashHex, _ := toString(row["phash_hex"])
	sha256, _ := toString(row["sha256"])
	sourceMTimeUnix, _ := toInt64(row["source_mtime_unix"])

	var collectionID *int64
	if cid, err := toInt64(row["collection_id"]); err == nil && cid != 0 {
		collectionID = &cid
	}

	thumbnailSmall, _ := toString(row["thumbnail_small_url"])
	thumbnailLarge, _ := toString(row["thumbnail_large_url"])

	createdAt, _ := toTime(row["created_at"])
	updatedAt, _ := toTime(row["updated_at"])

	return &domain.Image{
		ID:                id,
		CollectionID:      collectionID,
		Path:             toStringDefault(row["path"], ""),
		Filename:         toStringDefault(row["filename"], ""),
		SourceRoot:        toStringDefault(row["source_root"], ""),
		FileSize:          fileSize,
		Width:             width,
		Height:            height,
		Format:            toStringDefault(row["format"], ""),
		PHash:             phash,
		PHashHex:          phashHex,
		SHA256:            sha256,
		SourceMTimeUnix:   sourceMTimeUnix,
		ThumbnailSmallUrl: thumbnailSmall,
		ThumbnailLargeUrl: thumbnailLarge,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, nil
}

func mapTagFromD1(row map[string]any) (*domain.Tag, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, fmt.Errorf("tag.id: %w", err)
	}
	usageCount, _ := toInt(row["usage_count"])
	trustScore, _ := toFloat64(row["trust_score"])

	var parentID *int64
	if pid, err := toInt64(row["parent_id"]); err == nil && pid != 0 {
		parentID = &pid
	}

	primaryCategory, _ := toString(row["primary_category"])

	createdAt, _ := toTime(row["created_at"])

	return &domain.Tag{
		ID:              id,
		PreferredLabel:  toStringDefault(row["preferred_label"], ""),
		Slug:            toStringDefault(row["slug"], ""),
		Level:           toStringDefault(row["level"], ""),
		ParentID:        parentID,
		PrimaryCategory: primaryCategory,
		ReviewState:     toStringDefault(row["review_state"], ""),
		TrustScore:      trustScore,
		UsageCount:      usageCount,
		CreatedAt:       createdAt,
	}, nil
}

func mapTagAliasFromD1(row map[string]any) (*domain.TagAlias, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, fmt.Errorf("tag_alias.id: %w", err)
	}
	tagID, err := toInt64(row["tag_id"])
	if err != nil {
		return nil, fmt.Errorf("tag_alias.tag_id: %w", err)
	}
	isPreferred, _ := toBool(row["is_preferred"])

	return &domain.TagAlias{
		ID:              id,
		TagID:           tagID,
		Label:           toStringDefault(row["label"], ""),
		NormalizedLabel: toStringDefault(row["normalized_label"], ""),
		Locale:          toStringDefault(row["locale"], ""),
		AliasType:       toStringDefault(row["alias_type"], ""),
		IsPreferred:     isPreferred,
	}, nil
}

func mapImageTagFromD1(row map[string]any) (*domain.ImageTag, error) {
	imageID, err := toInt64(row["image_id"])
	if err != nil {
		return nil, fmt.Errorf("image_tag.image_id: %w", err)
	}
	tagID, err := toInt64(row["tag_id"])
	if err != nil {
		return nil, fmt.Errorf("image_tag.tag_id: %w", err)
	}
	confidence, _ := toFloat64(row["confidence"])

	var sourceObsID *int64
	if sid, err := toInt64(row["source_observation_id"]); err == nil && sid != 0 {
		sourceObsID = &sid
	}

	return &domain.ImageTag{
		ImageID:             imageID,
		TagID:               tagID,
		Source:              toStringDefault(row["source"], ""),
		SourceObservationID: sourceObsID,
		Confidence:          confidence,
		ReviewState:         toStringDefault(row["review_state"], ""),
	}, nil
}

func mapCollectionFromD1(row map[string]any) (*domain.Collection, error) {
	id, err := toInt64(row["id"])
	if err != nil {
		return nil, fmt.Errorf("collection.id: %w", err)
	}
	imageCount, _ := toInt(row["image_count"])

	var coverImageID *int64
	if cid, err := toInt64(row["cover_image_id"]); err == nil && cid != 0 {
		coverImageID = &cid
	}

	createdAt, _ := toTime(row["created_at"])
	updatedAt, _ := toTime(row["updated_at"])

	return &domain.Collection{
		ID:           id,
		Name:         toStringDefault(row["name"], ""),
		Description:  toStringDefault(row["description"], ""),
		CoverImageID: coverImageID,
		ImageCount:   imageCount,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}

func toInt64(v any) (int64, error) {
	if v == nil {
		return 0, fmt.Errorf("nil value")
	}
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toInt(v any) (int, error) {
	i, err := toInt64(v)
	if err != nil {
		return 0, err
	}
	return int(i), nil
}

func toFloat64(v any) (float64, error) {
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case int64:
		return float64(n), nil
	case int:
		return float64(n), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func toString(v any) (string, error) {
	if v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("cannot convert %T to string", v)
	}
	return s, nil
}

func toStringDefault(v any, def string) string {
	if v == nil {
		return def
	}
	s, ok := v.(string)
	if !ok {
		return def
	}
	return s
}

func toBool(v any) (bool, error) {
	if v == nil {
		return false, nil
	}
	switch b := v.(type) {
	case bool:
		return b, nil
	case float64:
		return b != 0, nil
	case int64:
		return b != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

func toTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	s, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("cannot convert %T to time", v)
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", s)
}