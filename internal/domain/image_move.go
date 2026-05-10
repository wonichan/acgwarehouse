package domain

const (
	ImageMoveConflictSkip      = "skip"
	ImageMoveConflictRename    = "rename"
	ImageMoveConflictOverwrite = "overwrite"
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
	ImageMoveReasonSystemTargetDir  = "system_target_dir"
	ImageMoveReasonDBUpdateFailed   = "db_update_failed"
	ImageMoveReasonRollbackFailed   = "rollback_failed"
	ImageMoveReasonMoveFailed       = "move_failed"
)

type ImageMoveRequest struct {
	SourceDirs              []string `json:"source_dirs"`
	TagID                   int64    `json:"tag_id"`
	TargetDir               string   `json:"target_dir"`
	Conflict                string   `json:"conflict"`
	Limit                   int      `json:"limit"`
	AllowTargetInsideSource bool     `json:"allow_target_inside_source,omitempty"`
}

type ImageMoveItem struct {
	ImageID     int64  `json:"image_id"`
	Filename    string `json:"filename"`
	SourcePath  string `json:"source_path"`
	TargetPath  string `json:"target_path"`
	Status      string `json:"status"`
	Reason      string `json:"reason,omitempty"`
	Retryable   bool   `json:"retryable"`
	Overwritten bool   `json:"overwritten,omitempty"`
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

const (
	ImageMoveBatchStatusQueued    = "queued"
	ImageMoveBatchStatusRunning   = "running"
	ImageMoveBatchStatusCompleted = "completed"
	ImageMoveBatchStatusFailed    = "failed"
	ImageMoveBatchStatusCancelled = "cancelled"
)

type ImageMoveBatch struct {
	ID               int64             `json:"id"`
	TagID            int64             `json:"tag_id"`
	SourceDirs       []string          `json:"source_dirs"`
	TargetDir        string            `json:"target_dir"`
	ConflictStrategy string            `json:"conflict_strategy"`
	TotalMatched     int64             `json:"total_matched"`
	Moved            int64             `json:"moved"`
	Skipped          int64             `json:"skipped"`
	Failed           int64             `json:"failed"`
	Status           string            `json:"status"`
	CreatedAt        string            `json:"created_at"`
	FinishedAt       *string           `json:"finished_at,omitempty"`
	Items            []ImageMoveItem   `json:"items,omitempty"`
	Progress         ImageMoveProgress `json:"progress"`
}

type ImageMoveProgress struct {
	Total       int64  `json:"total"`
	Processed   int64  `json:"processed"`
	Moved       int64  `json:"moved"`
	Skipped     int64  `json:"skipped"`
	Failed      int64  `json:"failed"`
	CurrentPath string `json:"current_path,omitempty"`
}

func ImageMoveReasonIsRetryable(reason string) bool {
	switch reason {
	case ImageMoveReasonPermissionDenied, ImageMoveReasonMoveFailed, ImageMoveReasonDBUpdateFailed:
		return true
	default:
		return false
	}
}
