package ai

import (
	"regexp"
	"strconv"
	"strings"
)

// TagRequest 单个图片标签请求
type TagRequest struct {
	ImageID int64
	Path    string
	Prompt  string
}

// BatchTagResult 批量标签结果
type BatchTagResult struct {
	Groups      [][]string // groups[0] 对应第1张图片的标签
	ModelName   string
	RawResponse string
	Confidence  float64
}

func ParseBatchTagsResponse(content string, groupCount int) [][]string {
	if groupCount <= 0 {
		return nil
	}

	results := make([][]string, groupCount)
	lineRE := regexp.MustCompile(`^(\d+)\s*[:：]\s*(.+)$`)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		matches := lineRE.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		idx, err := strconv.Atoi(matches[1])
		if err != nil || idx < 1 || idx > groupCount {
			continue
		}
		results[idx-1] = parseBatchLineTags(matches[2])
	}

	if hasAnyGroup(results) {
		return results
	}

	return make([][]string, groupCount)
}

func hasAnyGroup(groups [][]string) bool {
	for _, g := range groups {
		if len(g) > 0 {
			return true
		}
	}
	return false
}

func parseBatchLineTags(content string) []string {
	content = strings.ReplaceAll(content, "，", ",")
	parts := strings.Split(content, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}
