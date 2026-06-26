package search

import (
	bleve "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

const cjkAnalyzer = "cjk"

// NewMapping 创建图片索引字段映射。
func NewMapping() *mapping.IndexMappingImpl {
	mapping := bleve.NewIndexMapping()
	mapping.DefaultAnalyzer = cjkAnalyzer
	doc := bleve.NewDocumentMapping()
	doc.AddFieldMappingsAt("id", bleve.NewNumericFieldMapping())
	doc.AddFieldMappingsAt("cos_key", bleve.NewKeywordFieldMapping())
	doc.AddFieldMappingsAt("filename", textField(cjkAnalyzer))
	doc.AddFieldMappingsAt("tags", textField(cjkAnalyzer))
	doc.AddFieldMappingsAt("pinyin", bleve.NewKeywordFieldMapping())
	doc.AddFieldMappingsAt("first_letter", bleve.NewKeywordFieldMapping())
	doc.AddFieldMappingsAt("size", bleve.NewNumericFieldMapping())
	doc.AddFieldMappingsAt("created_at", bleve.NewDateTimeFieldMapping())
	doc.AddFieldMappingsAt("status", bleve.NewKeywordFieldMapping())
	mapping.DefaultMapping = doc
	return mapping
}

// textField 创建文本字段映射。
func textField(analyzer string) *mapping.FieldMapping {
	field := bleve.NewTextFieldMapping()
	field.Analyzer = analyzer
	return field
}
