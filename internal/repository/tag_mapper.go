package repository

import (
	"github.com/yachiyo/acgwarehouse/internal/model/do"
	"github.com/yachiyo/acgwarehouse/internal/model/po"
)

// tagsToDO 将标签持久化对象列表转换为领域对象列表。
func tagsToDO(tags []po.Tag) []do.Tag {
	result := make([]do.Tag, 0, len(tags))
	for _, tag := range tags {
		result = append(result, tagToDO(tag))
	}
	return result
}

// tagToDO 将标签持久化对象转换为领域对象。
func tagToDO(tag po.Tag) do.Tag {
	return do.Tag{
		ID:         tag.ID,
		Name:       tag.Name,
		UsageCount: tag.UsageCount,
		CreatedAt:  tag.CreatedAt.UTC(),
		UpdatedAt:  tag.UpdatedAt.UTC(),
	}
}

// tagToPO 将标签领域对象转换为持久化对象。
func tagToPO(tag do.Tag) po.Tag {
	return po.Tag{
		ID:         tag.ID,
		Name:       tag.Name,
		UsageCount: tag.UsageCount,
		CreatedAt:  tag.CreatedAt.UTC(),
		UpdatedAt:  tag.UpdatedAt.UTC(),
	}
}
