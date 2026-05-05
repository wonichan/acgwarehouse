package service

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

func (s *TagAdminService) GetParentCandidates(ctx context.Context, targetLevel string) ([]*domain.Tag, error) {
	return s.tagRepo.FindValidParentCandidates(ctx, targetLevel)
}

func (s *TagAdminService) GetTagTree(ctx context.Context) ([]TagTreeNode, error) {
	total, err := s.tagRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	tags, err := s.tagRepo.FindAll(ctx, total, 0)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return []TagTreeNode{}, nil
	}

	tagIDs := make([]int64, len(tags))
	for i, t := range tags {
		tagIDs[i] = t.ID
	}

	descMap, err := s.batchResolveDescendants(ctx, tagIDs)
	if err != nil {
		return nil, err
	}

	statsMap, err := s.batchComputeHierarchyStats(ctx, descMap)
	if err != nil {
		return nil, err
	}

	nodes := make(map[int64]*TagTreeNode, len(tags))
	childrenByParent := make(map[int64][]*TagTreeNode)
	roots := make([]*TagTreeNode, 0)
	for _, tag := range tags {
		treeStats := statsMap[tag.ID]
		node := &TagTreeNode{
			TagID:          tag.ID,
			PreferredLabel: tag.PreferredLabel,
			Level:          tag.Level,
			ParentID:       tag.ParentID,
			UsageCount:     int64(tag.UsageCount),
			TreeUsageCount: treeStats.TreeUsageCount,
			Children:       []TagTreeNode{},
		}
		nodes[tag.ID] = node
		if tag.ParentID == nil {
			roots = append(roots, node)
			continue
		}
		childrenByParent[*tag.ParentID] = append(childrenByParent[*tag.ParentID], node)
	}

	for id := range nodes {
		children := childrenByParent[id]
		sort.Slice(children, func(i, j int) bool {
			if children[i].UsageCount == children[j].UsageCount {
				return children[i].TagID < children[j].TagID
			}
			return children[i].UsageCount > children[j].UsageCount
		})
	}

	for _, node := range nodes {
		if node.ParentID != nil {
			if _, ok := nodes[*node.ParentID]; !ok {
				roots = append(roots, node)
			}
		}
	}

	sort.Slice(roots, func(i, j int) bool {
		if roots[i].UsageCount == roots[j].UsageCount {
			return roots[i].TagID < roots[j].TagID
		}
		return roots[i].UsageCount > roots[j].UsageCount
	})

	result := make([]TagTreeNode, 0, len(roots))
	for _, root := range roots {
		result = append(result, buildTagTreeNode(root, childrenByParent))
	}
	return result, nil
}

func (s *TagAdminService) ChangeLevel(ctx context.Context, tagID int64, targetLevel string, parentID *int64) (*domain.Tag, error) {
	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}

	children, err := s.tagRepo.FindChildrenByParent(ctx, tag.ID)
	if err != nil {
		return nil, err
	}
	descendants, err := s.tagRepo.ResolveAllDescendantIDs(ctx, []int64{tag.ID})
	if err != nil {
		return nil, err
	}
	hasChildren := len(children) > 0
	hasDescendants := len(descendants) > 1

	switch {
	case tag.Level == domain.TagLevelParent && targetLevel == domain.TagLevelChild && hasChildren:
		return nil, ErrInvalidHierarchy
	case tag.Level == domain.TagLevelRoot && targetLevel != domain.TagLevelRoot && hasDescendants:
		return nil, ErrInvalidHierarchy
	}

	if err := s.validateHierarchyAssignment(ctx, tag.ID, targetLevel, parentID); err != nil {
		return nil, err
	}

	tag.Level = targetLevel
	tag.ParentID = parentID

	childrenToDetach := []*domain.Tag(nil)
	if tag.Level == domain.TagLevelRoot && len(children) > 0 {
		childrenToDetach = children
	}
	return s.adminStore.ChangeTagLevel(ctx, tag, childrenToDetach)
}

func (s *TagAdminService) ReparentTag(ctx context.Context, tagID int64, parentID *int64) (*domain.Tag, error) {
	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTagNotFound
		}
		return nil, err
	}
	if tag.Level == domain.TagLevelRoot {
		return nil, ErrInvalidHierarchy
	}
	if err := s.validateHierarchyAssignment(ctx, tag.ID, tag.Level, parentID); err != nil {
		return nil, err
	}
	if parentID != nil {
		isDesc, err := s.isDescendantOf(ctx, *parentID, tag.ID)
		if err != nil {
			return nil, err
		}
		if isDesc {
			return nil, ErrInvalidHierarchy
		}
	}
	tag.ParentID = parentID

	return s.adminStore.ReparentTag(ctx, tag)
}

func (s *TagAdminService) validateHierarchyAssignment(ctx context.Context, tagID int64, level string, parentID *int64) error {
	switch level {
	case domain.TagLevelRoot:
		if parentID != nil {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelParent:
		if parentID == nil || *parentID == tagID {
			return ErrInvalidHierarchy
		}
		parentTag, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidHierarchy
			}
			return err
		}
		if parentTag.Level != domain.TagLevelRoot {
			return ErrInvalidHierarchy
		}
		return nil
	case domain.TagLevelChild:
		if parentID == nil {
			return nil
		}
		if *parentID == tagID {
			return ErrInvalidHierarchy
		}
		parentTag, err := s.tagRepo.FindByID(ctx, *parentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrInvalidHierarchy
			}
			return err
		}
		if parentTag.Level != domain.TagLevelParent {
			return ErrInvalidHierarchy
		}
		return nil
	default:
		return ErrInvalidHierarchy
	}
}

func (s *TagAdminService) isDescendantOf(ctx context.Context, potentialDescendantID, ancestorID int64) (bool, error) {
	descendants, err := s.tagRepo.ResolveAllDescendantIDs(ctx, []int64{ancestorID})
	if err != nil {
		return false, err
	}
	for _, id := range descendants {
		if id == potentialDescendantID {
			return true, nil
		}
	}
	return false, nil
}

func buildTagTreeNode(node *TagTreeNode, childrenByParent map[int64][]*TagTreeNode) TagTreeNode {
	result := *node
	children := childrenByParent[node.TagID]
	result.Children = make([]TagTreeNode, 0, len(children))
	for _, child := range children {
		result.Children = append(result.Children, buildTagTreeNode(child, childrenByParent))
	}
	return result
}
