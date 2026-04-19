package domain

import (
	"errors"
	"fmt"
)

var validLevels = map[string]bool{
	TagLevelRoot:   true,
	TagLevelParent: true,
	TagLevelChild:  true,
}

// GovernanceTagFilter carries all filter parameters for the governance tag list query.
// All conditions are AND-combined; levels within the Levels field are OR-combined.
type GovernanceTagFilter struct {
	Search        string
	Levels        []string
	OrphanOnly    bool
	MinUsageCount *int
	MaxUsageCount *int
	SourceAI      bool
	SourceManual  bool
	Limit         int
	Offset        int
}

// HasFilters returns true if any filter beyond search/pagination is active.
func (f *GovernanceTagFilter) HasFilters() bool {
	return len(f.Levels) > 0 ||
		f.OrphanOnly ||
		f.MinUsageCount != nil ||
		f.MaxUsageCount != nil ||
		f.SourceAI ||
		f.SourceManual
}

func (f *GovernanceTagFilter) Validate() error {
	for _, l := range f.Levels {
		if !validLevels[l] {
			return fmt.Errorf("invalid level %q: must be root, parent, or child", l)
		}
	}
	if f.MinUsageCount != nil && *f.MinUsageCount < 0 {
		return errors.New("min_usage_count must be non-negative")
	}
	if f.MaxUsageCount != nil && *f.MaxUsageCount < 0 {
		return errors.New("max_usage_count must be non-negative")
	}
	if f.MinUsageCount != nil && f.MaxUsageCount != nil && *f.MinUsageCount > *f.MaxUsageCount {
		return errors.New("min_usage_count must not exceed max_usage_count")
	}
	return nil
}
