package postgres

import (
	"context"
	"errors"
	"testing"
)

// TestClassifyConnectionError tests error classification for various connection scenarios.
func TestClassifyConnectionError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			name:        "nil error",
			err:         nil,
			wantMessage: "",
		},
		{
			name:        "authentication failed - password",
			err:         errors.New("password authentication failed"),
			wantMessage: "authentication failed",
		},
		{
			name:        "authentication failed - role does not exist",
			err:         errors.New("role \"baduser\" does not exist"),
			wantMessage: "authentication failed",
		},
		{
			name:        "database does not exist",
			err:         errors.New("database \"notfound\" does not exist"),
			wantMessage: "database does not exist",
		},
		{
			name:        "permission denied",
			err:         errors.New("permission denied for schema pg_catalog"),
			wantMessage: "permission denied",
		},
		{
			name:        "connection refused",
			err:         errors.New("connection refused"),
			wantMessage: "cannot connect to database",
		},
		{
			name:        "no route to host",
			err:         errors.New("no route to host"),
			wantMessage: "cannot connect to database",
		},
		{
			name:        "timeout",
			err:         errors.New("context deadline exceeded: timeout"),
			wantMessage: "cannot connect to database",
		},
		{
			name:        "generic error",
			err:         errors.New("some other error"),
			wantMessage: "some other error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyConnectionError(tt.err)
			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil for nil input, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected error, got nil")
				return
			}

			if tt.wantMessage != "" && !containsAny(result.Error(), tt.wantMessage) {
				t.Errorf("expected error to contain %q, got %q", tt.wantMessage, result.Error())
			}
		})
	}
}

// TestValidateDSN tests DSN validation for various formats.
func TestValidateDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid postgresql URL",
			dsn:     "postgresql://user:pass@localhost:5432/dbname",
			wantErr: false,
		},
		{
			name:    "valid postgres URL",
			dsn:     "postgres://user:pass@localhost:5432/dbname",
			wantErr: false,
		},
		{
			name:    "empty DSN",
			dsn:     "",
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name:    "invalid scheme",
			dsn:     "mysql://localhost/db",
			wantErr: true,
			errMsg:  "invalid scheme",
		},
		{
			name:    "missing host",
			dsn:     "postgres://user:pass@/db",
			wantErr: true,
			errMsg:  "missing host",
		},
		{
			name:    "malformed URL",
			dsn:     "ht!tp://[invalid",
			wantErr: true,
			errMsg:  "malformed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDSN(tt.dsn)
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}
			if tt.errMsg != "" && err != nil && !containsAny(err.Error(), tt.errMsg) {
				t.Errorf("expected error to contain %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

// TestConnect_InvalidDSN tests connection with invalid DSNs.
func TestConnect_InvalidDSN(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "invalid connection string",
			dsn:     "invalid-dsn",
			wantErr: true,
		},
		{
			name:    "wrong scheme",
			dsn:     "mysql://localhost/db",
			wantErr: true,
		},
		{
			name:    "empty DSN",
			dsn:     "",
			wantErr: true,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := createPool(ctx, tt.dsn)
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

// TestContainsAny tests the containsAny utility function.
func TestContainsAny(t *testing.T) {
	tests := []struct {
		name       string
		s          string
		substrings []string
		want       bool
	}{
		{
			name:       "single match",
			s:          "hello world",
			substrings: []string{"hello"},
			want:       true,
		},
		{
			name:       "multiple options one match",
			s:          "database error",
			substrings: []string{"authentication", "database", "timeout"},
			want:       true,
		},
		{
			name:       "no match",
			s:          "error message",
			substrings: []string{"database", "timeout"},
			want:       false,
		},
		{
			name:       "empty substrings",
			s:          "test",
			substrings: []string{},
			want:       false,
		},
		{
			name:       "empty string",
			s:          "",
			substrings: []string{"test"},
			want:       false,
		},
		{
			name:       "substring longer than string",
			s:          "hi",
			substrings: []string{"hello"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAny(tt.s, tt.substrings...)
			if result != tt.want {
				t.Errorf("containsAny(%q, %v) = %v, want %v", tt.s, tt.substrings, result, tt.want)
			}
		})
	}
}
