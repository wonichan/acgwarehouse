package pinyin

import "testing"

func Test_TextTokens_returns_full_pinyin_and_first_letters_when_text_has_chinese(t *testing.T) {
	// Given
	text := "初音未来"

	// When
	full, firstLetters := TextTokens(text)

	// Then
	if full != "chuyinweilai" {
		t.Fatalf("full = %q, want chuyinweilai", full)
	}
	if firstLetters != "cywl" {
		t.Fatalf("firstLetters = %q, want cywl", firstLetters)
	}
}

func Test_TextTokens_keeps_ascii_lowercase_when_text_has_mixed_filename(t *testing.T) {
	// Given
	text := "Miku-初音_01.png"

	// When
	full, firstLetters := TextTokens(text)

	// Then
	if full != "miku-chuyin_01.png" {
		t.Fatalf("full = %q, want miku-chuyin_01.png", full)
	}
	if firstLetters != "miku-cy_01.png" {
		t.Fatalf("firstLetters = %q, want miku-cy_01.png", firstLetters)
	}
}
