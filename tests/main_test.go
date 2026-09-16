package tests

import (
	"context"
	"os"
	"testing"

	"github.com/phathdt/dryft/internal/testutil"
)

// TestMain sets up a shared PostgreSQL container for all tests in this package.
// This dramatically improves test performance by reusing a single container
// instead of creating one per test.
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start shared container (will be used by non-short tests)
	_, err := testutil.GetSharedContainer(ctx)
	if err != nil {
		// If container fails to start, it's likely Docker isn't available
		// Let individual tests handle the error
	}

	// Run all tests
	code := m.Run()

	// Cleanup shared container
	_ = testutil.CleanupSharedContainer(ctx)

	os.Exit(code)
}
