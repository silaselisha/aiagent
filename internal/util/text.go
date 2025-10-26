package util

import (
	"regexp"
	"strings"
)

var whitespace = regexp.MustCompile(`\s+`)

// NormalizeWhitespace trims and collapses whitespace to single spaces.
func NormalizeWhitespace(s string) string {
	return strings.TrimSpace(whitespace.ReplaceAllString(s, " "))
}

// ContainsAnyCaseInsensitive returns true if text contains any of the needles (case-insensitive).
func ContainsAnyCaseInsensitive(text string, needles []string) bool {
	lt := strings.ToLower(text)
	for _, n := range needles {
		if strings.Contains(lt, strings.ToLower(n)) {
			return true
		}
	}
	return false
}

// Tokenize splits on spaces and punctuation.
func Tokenize(s string) []string {
	s = strings.ToLower(s)
	repl := strings.NewReplacer(
		",", " ", ".", " ", "!", " ", "?", " ", ":", " ", ";", " ",
		"\n", " ", "\t", " ", "\r", " ", "(", " ", ")", " ", "[", " ", "]", " ",
	)
	s = repl.Replace(s)
	parts := strings.Fields(s)
	return parts
}
