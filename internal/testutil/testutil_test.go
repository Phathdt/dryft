package testutil

import (
	"context"
	"testing"
	"time"
)

// TestPostgresContainerStarts verifies the PostgreSQL container can start and connect
func TestPostgresContainerStarts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pc, err := StartPostgres(ctx, t)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}

	if pc.ConnString == "" {
		t.Fatal("connection string should not be empty")
	}

	// Verify we can execute SQL
	err = pc.ExecuteSQL(ctx, "SELECT 1")
	AssertNoError(t, err)
}

// TestHelpers verifies test utility functions work correctly
func TestHelpers(t *testing.T) {
	t.Run("AssertNoError", func(t *testing.T) {
		AssertNoError(t, nil) // Should not fail
	})

	t.Run("CompareStringsIgnoreWhitespace", func(t *testing.T) {
		tests := []struct {
			a, b string
			want bool
		}{
			{"hello world", "hello world", true},
			{"hello  world", "hello world", true},
			{"  hello world  ", "hello world", true},
			{"hello\nworld", "hello world", true},
			{"hello world", "goodbye world", false},
		}

		for _, tt := range tests {
			got := CompareStringsIgnoreWhitespace(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareStringsIgnoreWhitespace(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("CreateTempDir", func(t *testing.T) {
		dir := CreateTempDir(t)
		if dir == "" {
			t.Fatal("temp dir should not be empty")
		}
	})

	t.Run("WriteFixture", func(t *testing.T) {
		dir := CreateTempDir(t)
		path := WriteFixture(t, dir, "test.txt", "content")

		content := ReadFixture(t, path)
		if content != "content" {
			t.Errorf("got %q, want %q", content, "content")
		}
	})
}
