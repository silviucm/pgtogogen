package main

import (
	"bytes"
	"os"
	"regexp"
	"unicode"
	"unicode/utf8"
)

var camelCaseRegularExpression = regexp.MustCompile("[0-9A-Za-z]+")

func CamelCase(original string) string {

	if original == "" {
		return ""
	}

	sections := camelCaseRegularExpression.FindAll([]byte(original), -1)
	for i, v := range sections {
		sections[i] = bytes.Title(v)
	}

	// while returning, make sure to lower the first character
	return LowerFirstChar(string(bytes.Join(sections, nil)))

}

func LowerFirstChar(original string) string {

	if original == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(original)
	return string(unicode.ToLower(r)) + original[n:]
}

// Checks if the file with the given path exists, returns true if yes
func FileExists(name string) bool {

	if _, err := os.Stat(name); err == nil {
		return true
	}

	return false

}
