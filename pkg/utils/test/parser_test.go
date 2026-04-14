package test

import (
	"storage/pkg/utils"
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"4KB", 4096},
		{"1KB", 1024},
		{"2MB", 2 * 1024 * 1024},
		{"10", 10},
	}

	for _, tt := range tests {
		got := utils.ParseSize(tt.input)
		if got != tt.want {
			t.Errorf("parseSize(%s) = %d; want %d", tt.input, got, tt.want)
		}
	}
}
