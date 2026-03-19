package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wonichan/acgwarehouse-backend/internal/service"
)

// ScanHandler 处理手动扫描任务
type ScanHandler struct {
	scannerSvc *service.ScannerService
	scanRoots  []string
}

// NewScanHandler 创建一个新的扫描任务处理器
func NewScanHandler(scannerSvc *service.ScannerService, scanRoots []string) *ScanHandler {
	return &ScanHandler{
		scannerSvc: scannerSvc,
		scanRoots:  scanRoots,
	}
}

// Handle 处理 manual_scan 类型的任务
func (h *ScanHandler) Handle(ctx context.Context, jobID int64, payload string) error {
	_ = jobID

	if h.scannerSvc == nil {
		return fmt.Errorf("scanner service is not initialized")
	}

	if len(h.scanRoots) == 0 {
		return fmt.Errorf("no scan roots configured")
	}

	// 解析 payload（如果有自定义配置）
	var p struct {
		Roots []string `json:"roots,omitempty"`
	}
	if payload != "" && payload != "{}" {
		if err := json.Unmarshal([]byte(payload), &p); err == nil && len(p.Roots) > 0 {
			// 使用 payload 中指定的扫描路径
			_, err := h.scannerSvc.Scan(ctx, p.Roots)
			return err
		}
	}

	// 使用配置的扫描路径
	result, err := h.scannerSvc.Scan(ctx, h.scanRoots)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// 记录扫描结果
	if result != nil {
		fmt.Printf("扫描完成: 总文件 %d, 导入 %d, 跳过 %d, 失败 %d, 耗时 %v\n",
			result.TotalFiles,
			result.Imported,
			result.Skipped,
			result.Failed,
			result.Duration,
		)
	}

	return nil
}
