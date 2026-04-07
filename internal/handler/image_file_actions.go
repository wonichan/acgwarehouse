package handler

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wonichan/acgwarehouse-backend/internal/domain"
)

type imageFileActionExecutor interface {
	OpenSource(path string) error
	DeleteSourceAndThumbnails(image domain.Image) error
}

type defaultImageFileActionExecutor struct{}

func newDefaultImageFileActionExecutor() imageFileActionExecutor {
	return defaultImageFileActionExecutor{}
}

func (defaultImageFileActionExecutor) OpenSource(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return fmt.Errorf("source file path is empty")
	}

	if _, err := os.Stat(trimmed); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", "start", "", trimmed)
	case "darwin":
		cmd = exec.Command("open", trimmed)
	default:
		cmd = exec.Command("xdg-open", trimmed)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

func (defaultImageFileActionExecutor) DeleteSourceAndThumbnails(image domain.Image) error {
	if err := removeIfExists(strings.TrimSpace(image.Path)); err != nil {
		return err
	}

	thumbnailCandidates := []string{image.ThumbnailSmallUrl, image.ThumbnailLargeUrl}
	resolved := make(map[string]struct{})
	for _, candidate := range thumbnailCandidates {
		path, ok := resolveThumbnailLocalPath(candidate)
		if !ok {
			continue
		}
		if _, exists := resolved[path]; exists {
			continue
		}
		resolved[path] = struct{}{}
		if err := removeIfExists(path); err != nil {
			return err
		}
	}

	return nil
}

func removeIfExists(path string) error {
	if path == "" {
		return nil
	}

	err := os.Remove(path)
	if err == nil || errors.Is(err, os.ErrNotExist) {
		return nil
	}

	return err
}

func resolveThumbnailLocalPath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	if strings.HasPrefix(strings.ToLower(trimmed), "file://") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return "", false
		}
		if parsed.Path == "" {
			return "", false
		}
		return filepath.FromSlash(strings.TrimPrefix(parsed.Path, "/")), true
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" {
		if filepath.IsAbs(trimmed) {
			return trimmed, true
		}
		return "", false
	}

	if parsed.Scheme == "http" || parsed.Scheme == "https" {
		if parsed.Host == "" {
			return "", false
		}
		if strings.EqualFold(parsed.Hostname(), "localhost") || parsed.Hostname() == "127.0.0.1" {
			candidate := parsed.Path
			if len(candidate) >= 3 && candidate[0] == '/' && candidate[2] == ':' {
				candidate = strings.TrimPrefix(candidate, "/")
			}
			candidate = filepath.FromSlash(candidate)
			if filepath.IsAbs(candidate) {
				return candidate, true
			}
		}
	}

	return "", false
}
