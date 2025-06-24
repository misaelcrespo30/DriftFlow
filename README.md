# DriftFlow

DriftFlow provides a lightweight CLI and library for managing database schema
    migrations. It works independently from your main application and can be reused
across different projects.

## Usage

Build the CLI:

```bash
go build ./cmd/driftflow
```

Run commands:

```bash
driftflow migrate    # generate migrations and apply them
driftflow up         # apply pending migrations
driftflow down NAME  # rollback to a migration
driftflow seed       # execute seed files
driftflow validate   # validate migration directory
```

The package also exposes a Go API for loading migration state and executing
migrations programmatically.
