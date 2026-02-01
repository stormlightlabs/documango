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
	before, _, ok := strings.Cut(s, "\n")
	if !ok {
		return s
	}
	return before
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

// Itoa converts an int to string without strconv.
func Itoa(i int) string {
	if i == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := i < 0
	if neg {
		i = -i
	}
	for i > 0 {
		buf = append(buf, byte('0'+i%10))
		i /= 10
	}
	if neg {
		buf = append(buf, '-')
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}
