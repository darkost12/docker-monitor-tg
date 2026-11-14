package textutil

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

func StripANSI(s string) string { return ansiRe.ReplaceAllString(s, "") }

func Tail(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}

	s = string(r[len(r)-max:])
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return "…" + s[i:]
	}

	return "…\n" + s
}

func Head(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "\n…"
}
