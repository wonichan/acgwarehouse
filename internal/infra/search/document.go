package search

import (
	"strconv"
	"strings"

	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"

	"github.com/yachiyo/acgwarehouse/internal/model/do"
	pinyinpkg "github.com/yachiyo/acgwarehouse/pkg/pinyin"
)

// Document 表示写入 bleve 的图片文档。
type Document struct {
	ID          int64    `json:"id"`
	COSKey      string   `json:"cos_key"`
	Filename    string   `json:"filename"`
	Tags        []string `json:"tags"`
	Pinyin      string   `json:"pinyin"`
	FirstLetter string   `json:"first_letter"`
	Size        int64    `json:"size"`
	CreatedAt   string   `json:"created_at"`
	Status      string   `json:"status"`
}

// imageDocument 将图片领域对象转换为索引文档。
func imageDocument(image do.Image) Document {
	full, firstLetters := pinyinpkg.TextTokens(strings.Join(append([]string{image.Filename}, image.Tags...), " "))
	return Document{
		ID:          image.ID,
		COSKey:      image.COSKey,
		Filename:    image.Filename,
		Tags:        image.Tags,
		Pinyin:      full,
		FirstLetter: firstLetters,
		Size:        image.Size,
		CreatedAt:   image.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:      string(image.Status),
	}
}

// newQuery 根据输入创建多字段检索条件。
func newQuery(text string) query.Query {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return bleve.NewMatchAllQuery()
	}
	fields := []string{"filename", "tags"}
	queries := make([]query.Query, 0, len(fields)+2)
	for _, field := range fields {
		match := bleve.NewMatchQuery(trimmed)
		match.SetField(field)
		queries = append(queries, match)
	}
	for _, field := range []string{"pinyin", "first_letter"} {
		prefix := bleve.NewPrefixQuery(strings.ToLower(trimmed))
		prefix.SetField(field)
		queries = append(queries, prefix)
	}
	return bleve.NewDisjunctionQuery(queries...)
}

// searchSize 返回检索分页大小。
func searchSize(query Query) int {
	if query.Size < 1 {
		return 20
	}
	return query.Size
}

// searchOffset 计算检索分页偏移量。
func searchOffset(query Query) int {
	if query.Page < 1 || query.Size < 1 {
		return 0
	}
	return (query.Page - 1) * query.Size
}

// hitsToIDs 将检索命中转换为图片 ID。
func hitsToIDs(hits search.DocumentMatchCollection) []int64 {
	ids := make([]int64, 0, len(hits))
	for _, hit := range hits {
		id, err := strconv.ParseInt(hit.ID, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// documentID 将图片 ID 转为 bleve 文档 ID。
func documentID(imageID int64) string {
	return strconv.FormatInt(imageID, 10)
}
