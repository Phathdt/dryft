# dryft

[![CI](https://github.com/phathdt/dryft/actions/workflows/pr-validation.yml/badge.svg)](https://github.com/phathdt/dryft/actions/workflows/pr-validation.yml)
[![codecov](https://codecov.io/gh/phathdt/dryft/branch/main/graph/badge.svg)](https://codecov.io/gh/phathdt/dryft)
[![Go Report Card](https://goreportcard.com/badge/github.com/phathdt/dryft)](https://goreportcard.com/report/github.com/phathdt/dryft)

> **Schema in. Migration out.**

Go CLI tool for PostgreSQL schema management via Prisma schemas and Goose migrations.

## Features

- **PostgreSQL introspection** → Generate Prisma schemas from existing databases
- **Prisma schema parsing** → Understand your data models
- **Schema diff engine** → Detect changes between schema versions
- **Goose migration generation** → Create SQL migrations automatically
- **Destructive change detection** → Warn about data loss risks

## Installation

Build from source:

```bash
git clone https://github.com/phathdt/dryft.git
cd dryft
make build
# or: go build -o dryft ./cmd/dryft
```

Or install to `$GOPATH/bin`:

```bash
make install
```

## Quick Start

Initialize a new dryft project:

```bash
dryft init
```

Pull schema from an existing PostgreSQL database:

```bash
dryft db pull
```

Create a migration from schema changes:

```bash
dryft migration create "add user table"
```

## Commands

- `dryft init` - Initialize configuration and state directory
- `dryft db pull` - Introspect PostgreSQL and generate Prisma schema
- `dryft migration create <name>` - Generate Goose migration from schema diff
- `dryft schema diff` - Preview changes between current and previous schemas
- `dryft validate` - Validate Prisma schema syntax and semantics

## Configuration

Create a `.dryft.yaml` file in your project root (or run `dryft init`):

```yaml
database:
  provider: postgresql
  url: ${DATABASE_URL}

schema:
  file: prisma/schema.prisma
  naming:
    fields: camelCase
    tables: PascalCase

migration:
  directory: migrations
  format: goose
  naming: timestamp

goose:
  version_table: goose_db_version
```

See `.dryft.yaml.example` for full configuration options.

## How It Works

dryft generates incremental migrations by comparing your Prisma schema against **migration history** (100% offline):

### Offline Migration Generation

Migration generation is **100% offline** and does NOT require database connection:

```bash
# First migration (empty migrations/ directory)
dryft migration create initial
# → Creates full schema (CREATE TABLE)

# Incremental migrations (existing migrations/ files)
dryft migration create add_user_bio
# → Creates incremental changes (ALTER TABLE)
```

**Workflow:**

1. **Parse existing migrations** in `migrations/` directory
2. Build virtual schema from migration history
3. Compare with current `schema.prisma`
4. Generate incremental SQL (ALTER instead of CREATE)

### When Database Connection is Needed

Database connection is ONLY needed for `db pull`:

```bash
# Introspect existing database → generate schema.prisma
dryft db pull
```

After `db pull`, all future migrations are generated offline from migration history.

## Development

```bash
# Run tests (skip integration tests)
make test-short

# Run all tests including integration
make test

# Format code
make fmt

# Run static analysis
make vet

# Run linter
make lint

# Development workflow (fmt + vet + test + build)
make dev
```

## Contributing

We welcome contributions! Before submitting a pull request, please ensure:

- All tests pass: `make test`
- Code is formatted: `make fmt`
- No lint errors: `make lint`
- Test coverage ≥85%

See [CONTRIBUTING.md](./docs/contributing.md) for detailed guidelines.

### Pull Request Requirements

All PRs must pass these automated checks:

✅ **Format** - Code properly formatted with `gofmt`  
✅ **Lint** - No linting errors (`golangci-lint`)  
✅ **Vet** - Pass `go vet` analysis  
✅ **Build** - Code compiles successfully  
✅ **Unit Tests** - All unit tests pass  
✅ **Integration Tests** - All integration tests pass  
✅ **Coverage** - Maintain ≥85% test coverage

## Development Status

✅ **v0.1 MVP Complete** - All 8 phases implemented and tested

- ✅ CLI framework and configuration
- ✅ Internal schema model (normalized IR)
- ✅ PostgreSQL introspection
- ✅ Prisma schema writer with relation fields
- ✅ Prisma schema parser
- ✅ Schema diff engine with dependency ordering
- ✅ PostgreSQL SQL generator
- ✅ Goose migration formatter
- ✅ 57 integration tests passing (testcontainers)

See [MVP_COMPLETE.md](./MVP_COMPLETE.md) for detailed summary.

## License

MIT
