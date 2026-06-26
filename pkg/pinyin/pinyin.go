package pinyin

import (
	"strings"
	"unicode"

	gopinyin "github.com/mozillazg/go-pinyin"
)

// TextTokens 返回文本的全拼与首字母检索字段。
func TextTokens(text string) (string, string) {
	args := gopinyin.NewArgs()
	args.Style = gopinyin.Normal
	args.Fallback = func(r rune, a gopinyin.Args) []string {
		return []string{string(r)}
	}

	var full strings.Builder
	var firstLetters strings.Builder
	for _, item := range text {
		writeRuneTokens(&full, &firstLetters, args, item)
	}
	return full.String(), firstLetters.String()
}

// writeRuneTokens 将单个字符转换并追加到拼音检索字段。
func writeRuneTokens(full *strings.Builder, firstLetters *strings.Builder, args gopinyin.Args, item rune) {
	values := gopinyin.SinglePinyin(item, args)
	if len(values) == 0 || values[0] == "" {
		lower := unicode.ToLower(item)
		full.WriteRune(lower)
		firstLetters.WriteRune(lower)
		return
	}

	value := strings.ToLower(values[0])
	full.WriteString(value)
	firstLetters.WriteByte(value[0])
}
