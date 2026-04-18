package parser

import (
	"testing"
	"time"
)

func TestParseJapaneseDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"S49.9.24", time.Date(1974, 9, 24, 0, 0, 0, 0, time.Local)},
		{"H2.1.1", time.Date(1990, 1, 1, 0, 0, 0, 0, time.Local)},
		{"R6.4.11", time.Date(2024, 4, 11, 0, 0, 0, 0, time.Local)},
	}

	for _, tt := range tests {
		got, err := ParseJapaneseDate(tt.input)
		if err != nil {
			t.Errorf("ParseJapaneseDate(%s) error: %v", tt.input, err)
			continue
		}
		if !got.Equal(tt.expected) {
			t.Errorf("ParseJapaneseDate(%s) = %v; want %v", tt.input, got, tt.expected)
		}
	}
}