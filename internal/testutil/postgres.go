package testutil

import (
	"context"
	"fmt"
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

// StartPostgres starts a PostgreSQL container for testing.
// The container uses PostgreSQL 16-alpine by default and is configured to
// auto-remove after the test completes via t.Cleanup().
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
		_ = pgContainer.Terminate(ctx)
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
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute sql: %w", err)
	}

	return nil
}
