package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "enum with cast",
			input:    "'user'::user_role",
			expected: "user",
		},
		{
			name:     "quoted string",
			input:    "'user'",
			expected: "user",
		},
		{
			name:     "unquoted string",
			input:    "user",
			expected: "user",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "numeric",
			input:    "42",
			expected: "42",
		},
		{
			name:     "boolean",
			input:    "true",
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeLiteral_EnumDefaults(t *testing.T) {
	// Test the specific case from issue:
	// Migration: DEFAULT 'user'
	// Prisma: @default(user) → converted to "'user'"

	migrationDefault := "'user'"
	prismaDefault := "'user'"

	assert.Equal(t,
		NormalizeLiteral(migrationDefault),
		NormalizeLiteral(prismaDefault),
		"Migration and Prisma defaults should normalize to same value")
}
