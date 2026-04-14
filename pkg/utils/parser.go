package utils

import (
	"strconv"
	"strings"
)

func ParseSize(s string) int {
	s = strings.ToUpper(strings.TrimSpace(s))

	multiplier := 1
	switch {
	case strings.HasSuffix(s, "KB"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "MB"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "MB")
	}

	n, _ := strconv.Atoi(s)
	return n * multiplier
}
