package service

import "github.com/wonichan/acgwarehouse-backend/internal/config"

// ResolveAITagQueueLimit returns the in-memory AI queue cap.
// Reuse auto_scan_batch_size so scan batch and queued AI window stay aligned.
func ResolveAITagQueueLimit(cfg *config.Config) int {
	if cfg != nil && cfg.AI.AutoScanBatchSize > 0 {
		return cfg.AI.AutoScanBatchSize
	}
	return 100
}
