# Schemaflow

Schemaflow provides a lightweight CLI and library for managing database schema
migrations. It works independently from the main `matters-service` project
and can be reused in other applications.

## Usage

Build the CLI:

```bash
go build ./cmd/schemaflow
```

Run commands:

```bash
schemaflow generate   # generate migrations
schemaflow up         # apply pending migrations
schemaflow down NAME  # rollback to a migration
schemaflow seed       # execute seed files
schemaflow validate   # validate migration directory
schemaflow diff       # print schema differences
```

The package also exposes a Go API for loading migration state and executing
migrations programmatically.
