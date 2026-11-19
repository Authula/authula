package emailtemplate

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/Authula/authula/util"
)

func FormatDuration(d time.Duration) string {
	return util.FormatDuration(d)
}

func Plural(s string, count int) string {
	if count == 1 {
		return s
	}
	return s + "s"
}

func Upper(s string) string {
	return strings.ToUpper(s)
}

func Lower(s string) string {
	return strings.ToLower(s)
}

func Title(s string) string {
	if s == "" {
		return ""
	}

	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		// Leave invalid UTF-8 untouched instead of replacing its leading byte.
		return s
	}

	return string(unicode.ToUpper(r)) + s[size:]
}

func CurrentYear() int {
	return time.Now().Year()
}
