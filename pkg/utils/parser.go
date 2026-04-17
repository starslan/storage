package utils

import (
	"fmt"
	"os"
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

func CheckDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("is not directory: %s", path)
	}

	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return err
}
