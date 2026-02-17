package migrations

import "strings"

// ProviderVariants maps a database provider to a lazy migration constructor.
type ProviderVariants map[string]func() []Migration

// ForProvider returns the migrations for the requested provider using the supplied variants.
func ForProvider(provider string, variants ProviderVariants) []Migration {
	if len(variants) == 0 {
		return nil
	}

	key := strings.ToLower(strings.TrimSpace(provider))
	if key == "" {
		return nil
	}

	if constructor, ok := variants[key]; ok && constructor != nil {
		return constructor()
	}

	return nil
}
