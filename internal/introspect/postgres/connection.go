package postgres

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultConnectTimeout = 10 * time.Second
	defaultQueryTimeout   = 30 * time.Second
)

// createPool creates and configures a pgxpool.Pool from a DSN.
// It validates the DSN, sets appropriate timeouts, and tests the connection.
func createPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	// Parse and validate DSN
	if err := validateDSN(dsn); err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	// Parse the DSN to create a config
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	// Set connection timeouts
	config.ConnConfig.ConnectTimeout = defaultConnectTimeout

	// Create context with timeout for initial connection
	connectCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()

	// Create the pool
	pool, err := pgxpool.NewWithConfig(connectCtx, config)
	if err != nil {
		return nil, classifyConnectionError(err)
	}

	// Test the connection with a simple query
	queryCtx, queryCancel := context.WithTimeout(ctx, defaultQueryTimeout)
	defer queryCancel()

	var version string
	err = pool.QueryRow(queryCtx, "SELECT version()").Scan(&version)
	if err != nil {
		pool.Close()
		return nil, classifyConnectionError(err)
	}

	return pool, nil
}

// validateDSN performs basic validation on the connection string.
func validateDSN(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("empty connection string")
	}

	parsedURL, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	if parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql" {
		return fmt.Errorf("invalid scheme '%s', expected 'postgres' or 'postgresql'", parsedURL.Scheme)
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("missing host")
	}

	return nil
}

// classifyConnectionError converts pgx errors into user-friendly error messages.
func classifyConnectionError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Authentication failures
	if containsAny(errMsg, "password authentication failed", "role", "does not exist") {
		return fmt.Errorf("authentication failed: check username/password")
	}

	// Database not found
	if containsAny(errMsg, "database", "does not exist") {
		return fmt.Errorf("database does not exist")
	}

	// Permission denied
	if containsAny(errMsg, "permission denied") {
		return fmt.Errorf("permission denied for system catalogs (need SELECT on pg_class)")
	}

	// Connection refused / timeout
	if containsAny(errMsg, "connection refused", "no route to host", "timeout") {
		return fmt.Errorf("cannot connect to database: %w", err)
	}

	// Return original error if not classified
	return err
}

// containsAny checks if the string contains any of the substrings.
func containsAny(s string, substrings ...string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
