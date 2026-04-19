package domain

import "testing"

func TestGovernanceTagFilterDefaults(t *testing.T) {
	t.Parallel()

	f := GovernanceTagFilter{}
	if f.HasFilters() {
		t.Error("empty filter should report HasFilters=false")
	}
}

func TestGovernanceTagFilterHasFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(*GovernanceTagFilter)
		want   bool
	}{
		{
			name:   "empty filter returns false",
			modify: func(f *GovernanceTagFilter) {},
			want:   false,
		},
		{
			name:   "levels set returns true",
			modify: func(f *GovernanceTagFilter) { f.Levels = []string{"root"} },
			want:   true,
		},
		{
			name:   "orphan only returns true",
			modify: func(f *GovernanceTagFilter) { f.OrphanOnly = true },
			want:   true,
		},
		{
			name:   "min usage returns true",
			modify: func(f *GovernanceTagFilter) { n := 5; f.MinUsageCount = &n },
			want:   true,
		},
		{
			name:   "max usage returns true",
			modify: func(f *GovernanceTagFilter) { n := 100; f.MaxUsageCount = &n },
			want:   true,
		},
		{
			name:   "source AI returns true",
			modify: func(f *GovernanceTagFilter) { f.SourceAI = true },
			want:   true,
		},
		{
			name:   "source manual returns true",
			modify: func(f *GovernanceTagFilter) { f.SourceManual = true },
			want:   true,
		},
		{
			name:   "search alone does not count as filter",
			modify: func(f *GovernanceTagFilter) { f.Search = "test" },
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := GovernanceTagFilter{}
			tt.modify(&f)
			if got := f.HasFilters(); got != tt.want {
				t.Errorf("HasFilters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGovernanceTagFilterValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(*GovernanceTagFilter)
		wantErr bool
	}{
		{
			name:    "valid empty filter",
			modify:  func(f *GovernanceTagFilter) {},
			wantErr: false,
		},
		{
			name:    "valid levels root",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"root"} },
			wantErr: false,
		},
		{
			name:    "valid levels parent and child",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"parent", "child"} },
			wantErr: false,
		},
		{
			name:    "valid all three levels",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"root", "parent", "child"} },
			wantErr: false,
		},
		{
			name:    "invalid level value grandpa",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"grandpa"} },
			wantErr: true,
		},
		{
			name:    "invalid level value empty string",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{""} },
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid levels",
			modify:  func(f *GovernanceTagFilter) { f.Levels = []string{"root", "invalid"} },
			wantErr: true,
		},
		{
			name:    "orphan only is valid",
			modify:  func(f *GovernanceTagFilter) { f.OrphanOnly = true },
			wantErr: false,
		},
		{
			name:    "negative min usage is invalid",
			modify:  func(f *GovernanceTagFilter) { n := -1; f.MinUsageCount = &n },
			wantErr: true,
		},
		{
			name:    "negative max usage is invalid",
			modify:  func(f *GovernanceTagFilter) { n := -5; f.MaxUsageCount = &n },
			wantErr: true,
		},
		{
			name:    "min greater than max is invalid",
			modify:  func(f *GovernanceTagFilter) { lo, hi := 100, 10; f.MinUsageCount = &lo; f.MaxUsageCount = &hi },
			wantErr: true,
		},
		{
			name:    "min equals max is valid",
			modify:  func(f *GovernanceTagFilter) { v := 50; f.MinUsageCount = &v; f.MaxUsageCount = &v },
			wantErr: false,
		},
		{
			name:    "valid usage range",
			modify:  func(f *GovernanceTagFilter) { lo, hi := 10, 100; f.MinUsageCount = &lo; f.MaxUsageCount = &hi },
			wantErr: false,
		},
		{
			name:    "zero min usage is valid",
			modify:  func(f *GovernanceTagFilter) { n := 0; f.MinUsageCount = &n },
			wantErr: false,
		},
		{
			name:    "source flags valid individually",
			modify:  func(f *GovernanceTagFilter) { f.SourceAI = true },
			wantErr: false,
		},
		{
			name:    "source flags valid together",
			modify:  func(f *GovernanceTagFilter) { f.SourceAI = true; f.SourceManual = true },
			wantErr: false,
		},
		{
			name:    "combined filters valid",
			modify: func(f *GovernanceTagFilter) {
				f.Levels = []string{"parent"}
				f.OrphanOnly = true
				lo := 5
				f.MinUsageCount = &lo
				f.SourceAI = true
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := GovernanceTagFilter{}
			tt.modify(&f)
			err := f.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
