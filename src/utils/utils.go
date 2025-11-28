package utils

import "strings"

func RemoveTrailingAnd(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(strings.ToLower(s), "and") {
		s = s[:len(s)-3]
	}
	return s
}
