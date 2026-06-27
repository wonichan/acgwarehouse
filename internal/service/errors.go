package service

import (
	pkgerrors "github.com/pkg/errors"
	"github.com/yachiyo/acgwarehouse/internal/ports"
)

var (
	// ErrForbidden 表示当前用户无权执行该操作。
	ErrForbidden = pkgerrors.New("service: forbidden")
	// ErrCollectionNotFound 表示收藏夹不存在。
	ErrCollectionNotFound = ports.ErrCollectionNotFound
	// ErrImageNotFound 表示图片不存在。
	ErrImageNotFound = ports.ErrImageNotFound
	// ErrTagNotFound 表示标签不存在。
	ErrTagNotFound = ports.ErrTagNotFound
)
