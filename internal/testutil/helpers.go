package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertNoError checks that err is nil, failing the test if it is not.
// Uses t.Helper() to report the caller's location.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// CompareStringsIgnoreWhitespace compares two strings after normalizing whitespace.
// All sequences of whitespace characters are treated as a single space,
// and leading/trailing whitespace is trimmed.
func CompareStringsIgnoreWhitespace(a, b string) bool {
	normalize := func(s string) string {
		s = strings.TrimSpace(s)
		fields := strings.Fields(s)
		return strings.Join(fields, " ")
	}
	return normalize(a) == normalize(b)
}

// ReadFixture reads a test fixture file from the given path.
// Path is relative to the test file calling this function.
// Fails the test if the file cannot be read.
func ReadFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", path, err)
	}
	return string(data)
}

// CreateTempDir creates a temporary directory for testing.
// The directory is automatically cleaned up when the test finishes.
// Returns the absolute path to the created directory.
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "dryft-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Errorf("failed to cleanup temp dir: %v", err)
		}
	})

	return dir
}

// WriteFixture writes data to a file in a temporary directory.
// Returns the full path to the created file.
// Useful for creating test input files.
func WriteFixture(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write fixture %s: %v", path, err)
	}

	return path
}
