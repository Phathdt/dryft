// Package testutil provides testing utilities and helpers for dryft integration tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PostgresContainer wraps a testcontainers PostgreSQL instance
type PostgresContainer struct {
	Container  *postgres.PostgresContainer
	ConnString string
}

var (
	sharedContainer     *PostgresContainer
	sharedContainerOnce sync.Once
	sharedContainerErr  error
)

// GetSharedContainer returns a shared PostgreSQL container for all tests.
// The container is created once and reused across all test functions.
// Each test should create its own schema/tables and clean up after itself.
func GetSharedContainer(ctx context.Context) (*PostgresContainer, error) {
	sharedContainerOnce.Do(func() {
		pgContainer, err := postgres.Run(ctx,
			"postgres:16-alpine",
			postgres.WithDatabase("testdb"),
			postgres.WithUsername("testuser"),
			postgres.WithPassword("testpass"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			sharedContainerErr = fmt.Errorf("failed to start postgres container: %w", err)
			return
		}

		connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			if termErr := pgContainer.Terminate(ctx); termErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to terminate container: %v\n", termErr)
			}
			sharedContainerErr = fmt.Errorf("failed to get connection string: %w", err)
			return
		}

		sharedContainer = &PostgresContainer{
			Container:  pgContainer,
			ConnString: connString,
		}
	})

	return sharedContainer, sharedContainerErr
}

// CleanupSharedContainer terminates the shared container.
// Should be called in TestMain after all tests complete.
func CleanupSharedContainer(ctx context.Context) error {
	if sharedContainer != nil {
		return sharedContainer.Close(ctx)
	}
	return nil
}

// StartPostgres starts a PostgreSQL container for testing.
// The container uses PostgreSQL 16-alpine by default and is configured to
// auto-remove after the test completes via t.Cleanup().
//
// DEPRECATED: Use GetSharedContainer() instead to reuse a single container
// across all tests. This function creates a new container per test which is slow.
//
// Returns a PostgresContainer with a ready-to-use connection string, or an error
// if the container fails to start.
func StartPostgres(ctx context.Context, t *testing.T) (*PostgresContainer, error) {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connString, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		if termErr := pgContainer.Terminate(ctx); termErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to terminate container: %v\n", termErr)
		}
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	pc := &PostgresContainer{
		Container:  pgContainer,
		ConnString: connString,
	}

	t.Cleanup(func() {
		if err := pc.Close(context.Background()); err != nil {
			t.Errorf("failed to cleanup postgres container: %v", err)
		}
	})

	return pc, nil
}

// Close stops and removes the PostgreSQL container
func (pc *PostgresContainer) Close(ctx context.Context) error {
	if pc.Container == nil {
		return nil
	}
	return pc.Container.Terminate(ctx)
}

// ExecuteSQL runs SQL statements against the test database.
// Useful for seeding test data or running initialization scripts.
func (pc *PostgresContainer) ExecuteSQL(ctx context.Context, sql string) error {
	conn, err := pgx.Connect(ctx, pc.ConnString)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close connection: %v\n", closeErr)
		}
	}()

	_, err = conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute sql: %w", err)
	}

	return nil
}

// CleanupTables drops all user tables in the test database.
// Use this in test cleanup to isolate tests when using shared container.
func (pc *PostgresContainer) CleanupTables(ctx context.Context) error {
	conn, err := pgx.Connect(ctx, pc.ConnString)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close connection: %v\n", closeErr)
		}
	}()

	// Drop all tables in public schema
	_, err = conn.Exec(ctx, `
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = 'public') LOOP
				EXECUTE 'DROP TABLE IF EXISTS public.' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop tables: %w", err)
	}

	// Drop all types (enums) in public schema
	_, err = conn.Exec(ctx, `
		DO $$ DECLARE
			r RECORD;
		BEGIN
			FOR r IN (SELECT typname FROM pg_type WHERE typnamespace = 'public'::regnamespace AND typtype = 'e') LOOP
				EXECUTE 'DROP TYPE IF EXISTS public.' || quote_ident(r.typname) || ' CASCADE';
			END LOOP;
		END $$;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop types: %w", err)
	}

	return nil
}
