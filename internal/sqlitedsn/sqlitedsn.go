// Package sqlitedsn builds SQLite connection strings carrying the defaults
// Authula relies on, so that every SQLite connection - application or test -
// behaves the same way.
package sqlitedsn

import (
	"net/url"
	"strings"
)

// Query parameters understood by the modernc.org/sqlite driver. Each default
// also lists the driver's alias for the same setting so a caller supplied
// alias is not silently duplicated.
const (
	busyTimeoutParam = "_busy_timeout"
	busyTimeoutAlias = "_timeout"
	timezoneParam    = "_timezone"
)

// The driver applies no busy timeout of its own, so a pooled writer that finds
// the database locked fails immediately with SQLITE_BUSY instead of waiting.
const defaultBusyTimeout = "5000"

// Timestamps are scanned with time.Parse, which resolves a zero UTC offset to
// the process local zone. Pinning the connection to UTC keeps scanned values
// in the zone they were written in, regardless of where the process runs.
const defaultTimezone = "UTC"

var defaults = []struct {
	param string
	alias string
	value string
}{
	{param: busyTimeoutParam, alias: busyTimeoutAlias, value: defaultBusyTimeout},
	{param: timezoneParam, value: defaultTimezone},
}

// Build turns a SQLite target - a file path, ":memory:", or an existing "file:"
// URI with or without query parameters - into a DSN with Authula's defaults
// applied. Parameters already present in the target are left untouched, so a
// caller can always override a default.
func Build(target string) string {
	base, query, _ := strings.Cut(target, "?")

	params, err := url.ParseQuery(query)
	if err != nil {
		// An unparsable query is the caller's to fix; rewriting it here would
		// only obscure the driver's error.
		return target
	}

	for _, d := range defaults {
		if params.Has(d.param) || (d.alias != "" && params.Has(d.alias)) {
			continue
		}
		params.Set(d.param, d.value)
	}

	return uri(base) + "?" + params.Encode()
}

// uri renders base as a SQLite "file:" URI. Only the characters that would
// otherwise terminate or misdelimit the path need escaping; escaping the rest
// would break URIs such as "file::memory:".
func uri(base string) string {
	if strings.HasPrefix(base, "file:") {
		return base
	}

	replacer := strings.NewReplacer("%", "%25", "#", "%23")

	return "file:" + replacer.Replace(base)
}
