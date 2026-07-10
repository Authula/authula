package emailtemplate

import (
	"strings"
	"time"

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
	return strings.ToUpper(s[:1]) + s[1:]
}

func CurrentYear() int {
	return time.Now().Year()
}
