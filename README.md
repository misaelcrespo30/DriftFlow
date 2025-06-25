# DriftFlow

DriftFlow provides a lightweight CLI and library for managing database schema
    migrations. It works independently from your main application and can be reused
across different projects.

The tool supports PostgreSQL, MySQL, SQLite, and SQL Server.

## Usage

Build the CLI:

```bash
go build ./cmd/driftflow
```

Run commands:

```bash
driftflow generate   # generate migrations from models
driftflow migrate    # generate migrations and apply them
driftflow up         # apply pending migrations
driftflow down NAME  # rollback to a migration
driftflow undo [n]   # rollback the last n migrations (default 1)
driftflow seed       # execute JSON seed files
driftflow seedgen    # generate JSON seed templates
driftflow validate   # validate migration directory
```

The package also exposes a Go API for loading migration state and executing
migrations programmatically.

### Environment

`SEED_DIR` can be used to specify where JSON seed files are located (default `seeds`).

### Dynamic seed templates

You can provide generator functions to populate template values when using
`GenerateSeedTemplatesWithData`:

```go
gens := map[string]func() interface{}{
    "name": func() interface{} { return "Alice" },
    "age":  func() interface{} { return 30 },
}
driftflow.GenerateSeedTemplatesWithData([]interface{}{User{}}, "seeds", gens)
```
