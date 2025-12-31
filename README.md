# DriftFlow

DriftFlow proporciona un CLI liviano y una librería para gestionar migraciones de
esquema de base de datos. Funciona de forma independiente a tu aplicación y se
puede reutilizar en múltiples proyectos.

Soporta PostgreSQL, MySQL y SQL Server.

## Instalación

Para usar la librería en otro proyecto:

```bash
go get github.com/misaelcrespo30/DriftFlow
```

Para instalar el CLI:

```bash
go install github.com/misaelcrespo30/DriftFlow/cmd/driftflow@latest
```

## Configuración (.env y variables de entorno)

DriftFlow carga configuración desde variables de entorno o un archivo `.env`.
Busca un `.env` en el directorio actual y sus padres; si no existe, utiliza el
archivo `.env` incluido en el repositorio.

Variables soportadas:

- `DB_TYPE`: driver (`postgres`, `mysql`, `sqlserver`). Default: `postgres`.
- `DSN`: cadena de conexión completa. Si no se define, se arma con las variables
  siguientes.
- `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE`:
  parámetros para construir el DSN. `DB_SSLMODE` default: `disable`.
- `MIG_DIR`: ruta de migraciones `.sql`. Default: `migrations`.
- `SEED_GEN_DIR`: ruta de generación/lectura de seeds `.seed.json`.
  Default: `internal/database/data`.
- `SEED_RUN_DIR`: ruta adicional de seeds (configurable para proyectos que lo
  necesiten). Default: `internal/database/seed`.
- `MODELS_DIR`: carpeta de modelos Go para generación de migraciones.
  Default: `internal/models`.

## Uso del CLI

Compilar desde el repo:

```bash
go build ./cmd/driftflow
```

Comandos disponibles:

```bash
driftflow generate        # genera migraciones desde modelos
                          # (snapshot + incremental)
driftflow migrate         # genera y aplica migraciones
driftflow up              # aplica migraciones pendientes
driftflow down VERSION    # revierte migraciones posteriores a VERSION
driftflow undo [n]        # revierte las últimas n migraciones (default 1)
driftflow rollback [n]    # alias de undo
driftflow seed            # ejecuta seeders registrados
driftflow seedgen         # genera templates JSON de seeds desde modelos
driftflow validate        # valida el directorio de migraciones
driftflow audit list      # lista el log de auditoría
driftflow audit export    # exporta el log (usa --json para JSON)
driftflow compare         # compara dos bases de datos
```

Flags globales útiles:

- `--dsn`: DSN de la base de datos.
- `--driver`: driver (`postgres`, `mysql`, `sqlserver`).
- `--migrations`: ruta de migraciones.
- `--seeds`: ruta de seeds (configurable por proyecto).
- `--seed-gen-dir`: ruta para generar/leer seeds.
- `--models`: ruta de modelos Go.


Para `compare`:

```bash
driftflow compare --from postgres://... --to postgres://...
```

Para `audit export`:

```bash
driftflow audit export --json
```

## Uso como librería

### Conexión y migraciones

```go
package main

import (
    "log"

    driftflow "github.com/misaelcrespo30/DriftFlow"
)

func main() {
    db, err := driftflow.ConnectToDB("", "") // usa .env/variables si están disponibles
    if err != nil {
        log.Fatal(err)
    }

    // Aplica migraciones pendientes
    if err := driftflow.Up(db, "migrations"); err != nil {
        log.Fatal(err)
    }
}
```

También disponibles:

- `driftflow.Down(db, dir, version)`: revierte hasta una versión.
- `driftflow.DownSteps(db, dir, n)`: revierte las últimas `n` migraciones.
- `driftflow.MigrateTo(db, dir, version)`: migra hasta una versión específica.
- `driftflow.Migrate(db, dir, models)`: genera y aplica migraciones desde modelos.
- `driftflow.GenerateModelMigrations(models, opts)`: genera migraciones sin aplicar.
- `driftflow.Validate(dir)`: valida archivos de migración.

### Generación de migraciones desde modelos

```go
models := []interface{}{User{}, Product{}}
opts := driftflow.GenerateOptions{
    Dir:          "migrations",
    ManifestMode: driftflow.ManifestStrict,
}

if err := driftflow.GenerateModelMigrations(models, opts); err != nil {
    log.Fatal(err)
}
```

### Seeds y templates JSON

Generar templates `.seed.json`:

```go
models := []interface{}{User{}, Product{}}
if err := driftflow.GenerateSeedTemplates(models, "seeds"); err != nil {
    log.Fatal(err)
}
```

Usar generadores dinámicos:

```go
gens := map[string]func() interface{}{
    "name": func() interface{} { return "Alice" },
    "age":  func() interface{} { return 30 },
}

if err := driftflow.GenerateSeedTemplatesWithData(models, "seeds", gens); err != nil {
    log.Fatal(err)
}
```

Ejecutar seeds desde JSON:

```go
if err := driftflow.SeedFromJSON(db, "seeds", models); err != nil {
    log.Fatal(err)
}
```

Registrar seeders programáticos (opcional):

```go
type UserSeeder struct{}

func (s UserSeeder) Seed(db *gorm.DB, filePath string) error {
    // implementar lectura del JSON y creación de datos
    return nil
}

func init() {
    driftflow.SetSeederRegistry(func() []driftflow.Seeder {
        return []driftflow.Seeder{UserSeeder{}}
    })
}
```

Luego puedes ejecutar:

```go
if err := driftflow.Seed(db, "seeds"); err != nil {
    log.Fatal(err)
}
```

### Ejecutar el CLI desde Go

```go
package main

import (
    "log"

    "github.com/misaelcrespo30/DriftFlow/cli"
)

func main() {
    cmd := cli.NewRootCommand()
    cmd.SetArgs([]string{"up"})
    if err := cmd.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

## Formato de archivos de migración

Las migraciones generadas usan un prefijo de timestamp como `YYYYMMDDHHMMSS_table.sql`.
Incluyen las secciones `Up` y `Down`:

```sql
-- +migrate Up
CREATE TABLE example (id int);

-- +migrate Down
DROP TABLE example;
```

Esto mantiene el orden cronológico y simplifica los rollbacks.
