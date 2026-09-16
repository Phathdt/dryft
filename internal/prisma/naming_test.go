package prisma

import (
	"testing"
)

func TestNamingConvention_TransformFieldName_CamelCase(t *testing.T) {
	nc := NamingConvention{Fields: "camelCase"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single word",
			input:    "email",
			expected: "email",
		},
		{
			name:     "snake_case two words",
			input:    "user_id",
			expected: "userId",
		},
		{
			name:     "snake_case three words",
			input:    "created_at_time",
			expected: "createdAtTime",
		},
		{
			name:     "multiple underscores",
			input:    "this_is_a_long_name",
			expected: "thisIsALongName",
		},
		{
			name:     "trailing underscore",
			input:    "field_name_",
			expected: "fieldName",
		},
		{
			name:     "leading underscore",
			input:    "_private_field",
			expected: "PrivateField",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nc.TransformFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("TransformFieldName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNamingConvention_TransformFieldName_SnakeCase(t *testing.T) {
	nc := NamingConvention{Fields: "snake_case"}

	input := "user_id"
	expected := "user_id"
	result := nc.TransformFieldName(input)

	if result != expected {
		t.Errorf("TransformFieldName(%q) = %q, want %q", input, result, expected)
	}
}

func TestNamingConvention_TransformModelName_PascalCase(t *testing.T) {
	nc := NamingConvention{Models: "PascalCase"}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single word",
			input:    "users",
			expected: "Users",
		},
		{
			name:     "snake_case two words",
			input:    "user_profiles",
			expected: "UserProfiles",
		},
		{
			name:     "snake_case three words",
			input:    "user_auth_tokens",
			expected: "UserAuthTokens",
		},
		{
			name:     "multiple underscores",
			input:    "very_long_table_name",
			expected: "VeryLongTableName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nc.TransformModelName(tt.input)
			if result != tt.expected {
				t.Errorf("TransformModelName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNamingConvention_TransformModelName_SnakeCase(t *testing.T) {
	nc := NamingConvention{Models: "snake_case"}

	input := "user_profiles"
	expected := "user_profiles"
	result := nc.TransformModelName(input)

	if result != expected {
		t.Errorf("TransformModelName(%q) = %q, want %q", input, result, expected)
	}
}

func TestDefaultNamingConvention(t *testing.T) {
	nc := DefaultNamingConvention()

	if nc.Fields != "camelCase" {
		t.Errorf("DefaultNamingConvention().Fields = %q, want %q", nc.Fields, "camelCase")
	}

	if nc.Models != "PascalCase" {
		t.Errorf("DefaultNamingConvention().Models = %q, want %q", nc.Models, "PascalCase")
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "two words",
			input:    "hello_world",
			expected: "helloWorld",
		},
		{
			name:     "three words",
			input:    "hello_world_test",
			expected: "helloWorldTest",
		},
		{
			name:     "starts with uppercase",
			input:    "Hello_world",
			expected: "HelloWorld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single word",
			input:    "user",
			expected: "User",
		},
		{
			name:     "two words",
			input:    "user_profile",
			expected: "UserProfile",
		},
		{
			name:     "three words",
			input:    "user_auth_token",
			expected: "UserAuthToken",
		},
		{
			name:     "already capitalized",
			input:    "User_Profile",
			expected: "UserProfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "lowercase word",
			input:    "hello",
			expected: "Hello",
		},
		{
			name:     "uppercase word",
			input:    "HELLO",
			expected: "HELLO",
		},
		{
			name:     "mixed case",
			input:    "hElLo",
			expected: "HElLo",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := capitalize(tt.input)
			if result != tt.expected {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
