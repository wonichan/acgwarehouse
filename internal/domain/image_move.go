package domain

const (
	ImageMoveConflictSkip = "skip"
)

const (
	ImageMoveStatusMovable = "movable"
	ImageMoveStatusMoved   = "moved"
	ImageMoveStatusSkipped = "skipped"
	ImageMoveStatusFailed  = "failed"
)

const (
	ImageMoveReasonSourceMissing    = "source_missing"
	ImageMoveReasonTargetExists     = "target_exists"
	ImageMoveReasonPermissionDenied = "permission_denied"
	ImageMoveReasonInvalidSourceDir = "invalid_source_dir"
	ImageMoveReasonInvalidTargetDir = "invalid_target_dir"
	ImageMoveReasonDBUpdateFailed   = "db_update_failed"
	ImageMoveReasonRollbackFailed   = "rollback_failed"
	ImageMoveReasonMoveFailed       = "move_failed"
)

type ImageMoveRequest struct {
	SourceDirs []string `json:"source_dirs"`
	TagID      int64    `json:"tag_id"`
	TargetDir  string   `json:"target_dir"`
	Conflict   string   `json:"conflict"`
	Limit      int      `json:"limit"`
}

type ImageMoveItem struct {
	ImageID    int64  `json:"image_id"`
	Filename   string `json:"filename"`
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Status     string `json:"status"`
	Reason     string `json:"reason,omitempty"`
}

type ImageMovePreview struct {
	TotalMatched int64           `json:"total_matched"`
	Movable      int64           `json:"movable"`
	Skipped      int64           `json:"skipped"`
	Items        []ImageMoveItem `json:"items"`
}

type ImageMoveResult struct {
	TotalMatched int64           `json:"total_matched"`
	Moved        int64           `json:"moved"`
	Skipped      int64           `json:"skipped"`
	Failed       int64           `json:"failed"`
	Items        []ImageMoveItem `json:"items"`
}
