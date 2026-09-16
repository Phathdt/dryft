package diff

import (
	"strings"
)

// normalizeExpression normalizes SQL expressions for comparison.
// This helps avoid spurious diffs from whitespace or equivalent syntax variations.
func normalizeExpression(expr string) string {
	if expr == "" {
		return ""
	}

	// Convert to lowercase
	s := strings.ToLower(expr)

	// Remove extra whitespace
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")

	// Normalize common PostgreSQL function variations
	replacements := map[string]string{
		"current_timestamp": "now()",
		"gen_random_uuid()": "uuid_generate_v4()",
	}

	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}

	return s
}
