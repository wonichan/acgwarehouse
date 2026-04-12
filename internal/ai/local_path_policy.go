package ai

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

var localImageRootsState struct {
	sync.RWMutex
	roots []string
}

func SetAllowedLocalImageRoots(roots []string) []string {
	localImageRootsState.Lock()
	defer localImageRootsState.Unlock()
	previous := append([]string(nil), localImageRootsState.roots...)
	normalized := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		if abs, err := filepath.Abs(trimmed); err == nil {
			trimmed = abs
		}
		if resolved, err := filepath.EvalSymlinks(trimmed); err == nil {
			trimmed = resolved
		}
		normalized = append(normalized, filepath.Clean(trimmed))
	}
	localImageRootsState.roots = normalized
	return previous
}

func validateLocalImagePath(path string) error {
	cleanedPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve local image path: %w", err)
	}
	if resolvedPath, err := filepath.EvalSymlinks(cleanedPath); err == nil {
		cleanedPath = resolvedPath
	}
	cleanedPath = filepath.Clean(cleanedPath)

	localImageRootsState.RLock()
	roots := append([]string(nil), localImageRootsState.roots...)
	localImageRootsState.RUnlock()
	if len(roots) == 0 {
		return nil
	}

	for _, root := range roots {
		rel, err := filepath.Rel(root, cleanedPath)
		if err != nil {
			continue
		}
		if rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return nil
		}
	}

	return fmt.Errorf("local image path outside allowed scan roots: %s", cleanedPath)
}
