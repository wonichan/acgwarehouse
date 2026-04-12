package ai

import (
	"reflect"
	"testing"
)

func TestParseBatchTagsResponse_ParsesNumberedLines(t *testing.T) {
	input := `1: 泳装,黑丝,银发,B cup
2: 女仆装,白丝,短发
3: 泳装,比基尼,金发
4: 校服,黑丝,双马尾,粉发`

	results := ParseBatchTagsResponse(input, 4)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	expected := [][]string{
		{"泳装", "黑丝", "银发", "B cup"},
		{"女仆装", "白丝", "短发"},
		{"泳装", "比基尼", "金发"},
		{"校服", "黑丝", "双马尾", "粉发"},
	}

	for i, want := range expected {
		if !reflect.DeepEqual(results[i], want) {
			t.Errorf("group %d: expected %v, got %v", i+1, want, results[i])
		}
	}
}

func TestParseBatchTagsResponse_SplitsOnChineseAndEnglishCommas(t *testing.T) {
	input := `1: tag1,tag2，tag3,tag4`

	results := ParseBatchTagsResponse(input, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := []string{"tag1", "tag2", "tag3", "tag4"}
	if !reflect.DeepEqual(results[0], expected) {
		t.Errorf("expected %v, got %v", expected, results[0])
	}
}

func TestParseBatchTagsResponse_HandlesExtraWhitespace(t *testing.T) {
	input := `1:  tag1 ,  tag2  , tag3  `

	results := ParseBatchTagsResponse(input, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := []string{"tag1", "tag2", "tag3"}
	if !reflect.DeepEqual(results[0], expected) {
		t.Errorf("expected %v, got %v", expected, results[0])
	}
}

func TestParseBatchTagsResponse_SkipsEmptyTags(t *testing.T) {
	input := `1: tag1,,  ,tag2,`

	results := ParseBatchTagsResponse(input, 1)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	expected := []string{"tag1", "tag2"}
	if !reflect.DeepEqual(results[0], expected) {
		t.Errorf("expected %v, got %v", expected, results[0])
	}
}

func TestParseBatchTagsResponse_HandlesMissingGroups(t *testing.T) {
	input := `1: tag1,tag2
3: tag5,tag6`

	results := ParseBatchTagsResponse(input, 4)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	if len(results[0]) != 2 {
		t.Errorf("group 1 expected 2 tags, got %d", len(results[0]))
	}
	if len(results[1]) != 0 {
		t.Errorf("group 2 expected 0 tags (missing), got %d", len(results[1]))
	}
	if len(results[2]) != 2 {
		t.Errorf("group 3 expected 2 tags, got %d", len(results[2]))
	}
	if len(results[3]) != 0 {
		t.Errorf("group 4 expected 0 tags (missing), got %d", len(results[3]))
	}
}

func TestParseBatchTagsResponse_FallbackForNonNumberedOutput(t *testing.T) {
	input := `tag1,tag2,tag3`

	results := ParseBatchTagsResponse(input, 4)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	// 无编号行时，返回全空组（validateGeneratedTags 会拒绝空标签）
	for i := range results {
		if len(results[i]) != 0 {
			t.Errorf("group %d expected 0 tags, got %d: %v", i+1, len(results[i]), results[i])
		}
	}
}

func TestParseBatchTagsResponse_ZeroCount(t *testing.T) {
	input := `1: tag1,tag2`

	results := ParseBatchTagsResponse(input, 0)

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
