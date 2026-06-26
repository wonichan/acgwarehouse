package search

import (
	"context"
	"os"
	"path/filepath"

	bleve "github.com/blevesearch/bleve/v2"
	_ "github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	pkgerrors "github.com/pkg/errors"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
)

// Query 定义全文检索请求。
type Query struct {
	Text string
	Page int
	Size int
}

// Searcher 定义服务层使用的搜索接口适配器。
type Searcher struct {
	index *Index
}

// NewSearcher 创建服务层搜索接口适配器。
func NewSearcher(index *Index) Searcher {
	return Searcher{index: index}
}

// Search 执行全文检索并返回图片 ID 列表。
func (s Searcher) Search(ctx context.Context, query do.ImageSearchQuery) (do.ImageSearchResult, error) {
	return s.index.Search(ctx, Query{Text: query.Text, Page: query.Page, Size: query.Size})
}

// Index 写入或更新图片索引文档。
func (s Searcher) Index(ctx context.Context, image do.Image) error {
	return s.index.Index(ctx, image)
}

// Delete 移除指定图片的搜索文档。
func (s Searcher) Delete(ctx context.Context, imageID int64) error {
	return s.index.Delete(ctx, imageID)
}

// Index 封装 bleve 图片索引。
type Index struct {
	index bleve.Index
	path  string
}

// Open 打开或创建 bleve 图片索引。
func Open(path string) (*Index, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, pkgerrors.WithMessage(err, "ensure bleve parent dir")
	}
	index, err := bleve.Open(path)
	if err == nil {
		return &Index{index: index, path: path}, nil
	}
	if err != bleve.ErrorIndexPathDoesNotExist {
		return nil, pkgerrors.WithMessage(err, "open bleve index")
	}
	created, err := bleve.New(path, NewMapping())
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "create bleve index")
	}
	return &Index{index: created, path: path}, nil
}

// Index 写入或更新图片索引文档。
func (i *Index) Index(ctx context.Context, image do.Image) error {
	if err := ctx.Err(); err != nil {
		return pkgerrors.WithMessage(err, "index image context")
	}
	if i == nil || i.index == nil {
		return pkgerrors.New("bleve index is nil")
	}
	if !image.IsActive() {
		return i.Delete(ctx, image.ID)
	}
	if err := i.index.Index(documentID(image.ID), imageDocument(image)); err != nil {
		return pkgerrors.WithMessage(err, "index image")
	}
	return nil
}

// Delete 删除图片索引文档。
func (i *Index) Delete(ctx context.Context, imageID int64) error {
	if err := ctx.Err(); err != nil {
		return pkgerrors.WithMessage(err, "delete image context")
	}
	if i == nil || i.index == nil {
		return pkgerrors.New("bleve index is nil")
	}
	if err := i.index.Delete(documentID(imageID)); err != nil {
		return pkgerrors.WithMessage(err, "delete image index")
	}
	return nil
}

// Search 执行全文检索并返回图片 ID 列表。
func (i *Index) Search(ctx context.Context, query Query) (do.ImageSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return do.ImageSearchResult{}, pkgerrors.WithMessage(err, "search image context")
	}
	if i == nil || i.index == nil {
		return do.ImageSearchResult{}, pkgerrors.New("bleve index is nil")
	}
	request := bleve.NewSearchRequestOptions(newQuery(query.Text), searchSize(query), searchOffset(query), false)
	result, err := i.index.Search(request)
	if err != nil {
		return do.ImageSearchResult{}, pkgerrors.WithMessage(err, "search image index")
	}
	return do.ImageSearchResult{IDs: hitsToIDs(result.Hits), Total: int64(result.Total)}, nil
}

// Count 返回当前索引文档总数。
func (i *Index) Count(ctx context.Context) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, pkgerrors.WithMessage(err, "count image context")
	}
	count, err := i.index.DocCount()
	if err != nil {
		return 0, pkgerrors.WithMessage(err, "count image index")
	}
	return count, nil
}

// Close 关闭 bleve 索引文件。
func (i *Index) Close() error {
	if i == nil || i.index == nil {
		return nil
	}
	if err := i.index.Close(); err != nil {
		return pkgerrors.WithMessage(err, "close bleve index")
	}
	return nil
}

// Reset 删除并重建当前索引。
func (i *Index) Reset() error {
	if i == nil {
		return pkgerrors.New("bleve index is nil")
	}
	if err := i.Close(); err != nil {
		return pkgerrors.WithMessage(err, "close bleve before reset")
	}
	if err := os.RemoveAll(i.path); err != nil {
		return pkgerrors.WithMessage(err, "remove bleve index")
	}
	if err := ensureParentDir(i.path); err != nil {
		return pkgerrors.WithMessage(err, "ensure bleve parent dir after reset")
	}
	created, err := bleve.New(i.path, NewMapping())
	if err != nil {
		return pkgerrors.WithMessage(err, "create bleve after reset")
	}
	i.index = created
	return nil
}

// Reindex 用 SQLite 真相源重建索引文档。
func (i *Index) Reindex(ctx context.Context, images []do.Image) error {
	if err := i.Reset(); err != nil {
		return pkgerrors.WithMessage(err, "reset image index")
	}
	batch := i.index.NewBatch()
	for _, image := range images {
		if err := ctx.Err(); err != nil {
			return pkgerrors.WithMessage(err, "reindex image context")
		}
		if image.IsActive() {
			if err := batch.Index(documentID(image.ID), imageDocument(image)); err != nil {
				return pkgerrors.WithMessage(err, "queue reindex image")
			}
		}
	}
	if err := i.index.Batch(batch); err != nil {
		return pkgerrors.WithMessage(err, "batch reindex images")
	}
	return nil
}

// ensureParentDir 确保索引父目录存在。
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return pkgerrors.WithMessage(err, "make bleve dir")
	}
	return nil
}
