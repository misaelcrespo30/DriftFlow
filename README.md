# DriftFlow

DriftFlow provides a lightweight CLI and library for managing database schema migrations.
It works independently from your main application and can be reused across different projects.

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
driftflow rollback [n] # alias of undo
driftflow seed       # execute JSON seed files
driftflow seedgen    # generate JSON seed templates
                     # existing seed files are skipped
driftflow validate   # validate migration directory
```

### Migration file naming

Generated migrations use a timestamp prefix similar to other frameworks. Files
are created as `YYYYMMDDHHMMSS_table.sql` and include both directions using
section markers:

```sql
-- +migrate Up
CREATE TABLE example (id int);

-- +migrate Down
DROP TABLE example;
```

This keeps migrations ordered chronologically and simplifies rollbacks.

### Using the CLI programmatically

Besides calling the binary directly you can execute DriftFlow commands from
your own Go code. Import the `cli` package and run the root command with any
arguments you need:

```go
//package main

//import (
//    "log"

//    "github.com/misaelcrespo30/DriftFlow/cli"
//)

//func main() {
 //   cmd := cli.NewRootCommand()
 //   cmd.SetArgs([]string{"up"})
 //   if err := cmd.Execute(); err != nil {
 //       log.Fatal(err)
 //   }
}
```

### Environment

DriftFlow loads configuration from environment variables or a `.env` file.
It searches the working directory and its parents for `.env`, falling back to
the default file bundled with the library if none is found:

- `DB_TYPE` sets the database driver (`postgres`, `mysql`, `sqlserver`). Defaults to `postgres`.
- `DSN` provides the full database connection string. When not set, a DSN is assembled from `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` and `DB_SSLMODE`.
- `MIG_DIR` (or `MIGRATIONS_PATH`) specifies where `.sql` migration files live (default `migrations`). When both are set, `MIGRATIONS_PATH` takes precedence.
- `SEED_DIR` specifies where JSON seed files live (default `seeds`). Both directories must exist when running migrations or seeds.
- `MODELS_DIR` sets the directory containing Go model definitions used to generate migrations (default `models`).

If no `.env` file exists, `config.EnsureEnvFile` will create one using the
defaults in `config.defaultEnv`. When a file is present but missing any of these
keys, they are appended automatically with their default values.

### Dynamic seed templates

You can provide generator functions to populate template values when using
`GenerateSeedTemplatesWithData`:

```go
//gens := map[string]func() interface{}{
//    "name": func() interface{} { return "Alice" },
//    "age":  func() interface{} { return 30 },
//}
//driftflow.GenerateSeedTemplatesWithData([]interface{}{User{}}, "seeds", gens)
```
