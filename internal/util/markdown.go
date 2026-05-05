package util

import (
	stripmd "github.com/writeas/go-strip-markdown"
)

// StripMarkdown 去除 Markdown 语法，返回纯文本
func StripMarkdown(content string) string {
	if content == "" {
		return ""
	}
	return stripmd.Strip(content)
}

// TruncateDescription 去除 Markdown 后截断到 maxLen 字符，超长追加 "..."
func TruncateDescription(content string, maxLen int) string {
	plain := StripMarkdown(content)
	if plain == "" {
		return ""
	}
	runes := []rune(plain)
	if len(runes) <= maxLen {
		return plain
	}
	return string(runes[:maxLen]) + "..."
}
