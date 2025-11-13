package utils

import "strings"

func FirstLetterToUpper(s string) string {
	if len(s) > 0 {
		s = strings.ToUpper(s[:1]) + s[1:]
	}
	return s
}
