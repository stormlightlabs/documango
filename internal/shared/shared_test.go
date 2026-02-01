package shared

import (
	"bytes"
	"testing"
)

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "Hello"},
		{"HELLO", "Hello"},
		{"hELLO", "Hello"},
		{"hello world", "Hello World"},
		{"", ""},
		{"a", "A"},
		{"123abc", "123Abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Capitalize(tt.input)
			if result != tt.expected {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"first\nsecond\nthird", "first"},
		{"single line", "single line"},
		{"", ""},
		{"line1\n", "line1"},
		{"\nstarts with newline", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := FirstLine(tt.input)
			if result != tt.expected {
				t.Errorf("FirstLine(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompress(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"hello world"},
		{""},
		{"this is a longer string that should compress well"},
		{"1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			compressed := Compress(tt.input)
			if compressed == nil {
				t.Error("Compress returned nil")
			}

			if len(tt.input) > 0 && bytes.Equal(compressed, []byte(tt.input)) {
				t.Error("Compressed data should differ from original")
			}
		})
	}
}

func TestNormalizeLineEndings(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"line1\r\nline2", "line1\nline2"},
		{"line1\rline2", "line1\nline2"},
		{"line1\nline2", "line1\nline2"},
		{"mixed\r\n\r\n", "mixed\n\n"},
		{"", ""},
		{"no newlines", "no newlines"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeLineEndings(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeLineEndings(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		text   string
		maxLen int
		want   string
	}{
		{"hello world", 5, "hello..."},
		{"hello", 10, "hello"},
		{"", 5, ""},
		{"exact", 5, "exact"},
		{"short", 0, "..."},
		{"test", 3, "tes..."},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := TruncateText(tt.text, tt.maxLen)
			if got != tt.want {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.text, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{99, "99"},
		{100, "100"},
		{12345, "12345"},
		{-1, "-1"},
		{-9, "-9"},
		{-10, "-10"},
		{-12345, "-12345"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := Itoa(tt.input)
			if result != tt.expected {
				t.Errorf("Itoa(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringPtr(t *testing.T) {
	tests := []string{"hello", "", "test string", "123"}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			ptr := StringPtr(tt)
			if ptr == nil {
				t.Error("StringPtr returned nil")
			}
			if *ptr != tt {
				t.Errorf("StringPtr(%q) = %q", tt, *ptr)
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []bool{true, false}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ptr := BoolPtr(tt)
			if ptr == nil {
				t.Error("BoolPtr returned nil")
			}
			if *ptr != tt {
				t.Errorf("BoolPtr(%v) = %v", tt, *ptr)
			}
		})
	}
}
