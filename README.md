# dryft

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

# Development workflow (fmt + vet + test + build)
make dev
```

## Development Status

🚧 **v0.1 MVP in development** - Phase 1 complete

## License

MIT
