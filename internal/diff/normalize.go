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

// NormalizeLiteral normalizes default literal values for comparison.
// Handles enum casts, quotes, and common variations.
func NormalizeLiteral(literal string) string {
	if literal == "" {
		return ""
	}

	normalized := literal

	// Strip type casts (e.g., 'user'::user_role → 'user')
	if idx := strings.Index(normalized, "::"); idx != -1 {
		normalized = normalized[:idx]
	}

	// Trim whitespace
	normalized = strings.TrimSpace(normalized)

	// Remove surrounding single quotes
	normalized = strings.Trim(normalized, "'")

	// Convert to lowercase for case-insensitive comparison
	normalized = strings.ToLower(normalized)

	return normalized
}
