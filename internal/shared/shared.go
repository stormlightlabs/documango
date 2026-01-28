package shared

import (
	"strings"

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
