package cli

import (
	"context"
	"os"
	"testing"

	"github.com/phathdt/dryft/internal/config"
)

func TestInitCreatesValidConfig(t *testing.T) {
	tmpdir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpdir)

	// Simulate dryft init
	app := NewApp()
	err := app.Run(context.Background(), []string{"dryft", "init"})
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Check files were created
	if _, err := os.Stat(".dryft.yaml"); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	os.Setenv("DATABASE_URL", "postgresql://test@localhost/testdb")

	// Try to load the config
	cfg, err := config.Load(".dryft.yaml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify structure
	if cfg.Database.URL != "postgresql://test@localhost/testdb" {
		t.Errorf("database URL not expanded: got %s", cfg.Database.URL)
	}
	if cfg.Schema.File != "prisma/schema.prisma" {
		t.Errorf("schema file incorrect: got %s", cfg.Schema.File)
	}
	if cfg.Migration.Directory != "migrations" {
		t.Errorf("migration dir incorrect: got %s", cfg.Migration.Directory)
	}
}
