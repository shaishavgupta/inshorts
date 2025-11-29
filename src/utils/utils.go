package utils

import (
	"fmt"
	"strings"
)

func RemoveTrailingAnd(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(strings.ToLower(s), "and") {
		s = s[:len(s)-3]
	}
	return s
}

// QuoteAndEscapeStrings splits a comma-separated string, trims whitespace,
// escapes single quotes, and returns a slice of SQL-quoted strings.
func QuoteAndEscapeStrings(input string) []string {
	if input == "" {
		return []string{}
	}

	items := strings.Split(input, ",")
	quoted := make([]string, 0, len(items))

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// Escape single quotes by doubling them
		escaped := strings.ReplaceAll(item, "'", "''")
		quoted = append(quoted, fmt.Sprintf("'%s'", escaped))
	}

	return quoted
}

// FormatStringsForLikeQuery takes an array of strings and returns them formatted
// for SQL LIKE queries with wildcards. Example: ["abp", "bbc"] -> "'%abp%','%bbc%'"
func FormatStringsForLikeQuery(values []string) string {
	if len(values) == 0 {
		return ""
	}

	patterns := make([]string, 0, len(values))

	for _, s := range values {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		patterns = append(patterns, fmt.Sprintf("'%s'", "%"+s+"%"))
	}

	return strings.Join(patterns, ",")
}
