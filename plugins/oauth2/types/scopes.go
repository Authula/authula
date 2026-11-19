package types

import "strings"

// MergeScopes merges two space-delimited scope strings, preserving the order of
// first appearance and removing duplicates. It is used when a user grants
// additional scopes to an existing consent.
func MergeScopes(existing string, requested string) string {
	seen := make(map[string]struct{})
	merged := make([]string, 0)

	for _, scope := range []string{existing, requested} {
		for _, value := range strings.Fields(scope) {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			merged = append(merged, value)
		}
	}

	return strings.Join(merged, " ")
}
