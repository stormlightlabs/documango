package shared

import (
	"strings"

	"github.com/stormlightlabs/documango/internal/codec"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func Capitalize(s string) string {
	return cases.Title(language.Und).String(s)
}

func FirstLine(s string) string {
	idx := strings.Index(s, "\n")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

func Compress(body string) []byte {
	compressed, _ := codec.Compress([]byte(body))
	return compressed
}

func NormalizeLineEndings(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// TruncateText truncates text to the specified length.
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
