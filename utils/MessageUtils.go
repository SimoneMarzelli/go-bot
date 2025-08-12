package utils

import "unicode"

func CapitalizeFirstLetter(s string) string {
	if s == "" {
		return ""
	}

	st := []rune(s)

	if unicode.IsLetter(st[0]) {
		st[0] = unicode.ToUpper(st[0])
	}

	return string(st)
}
