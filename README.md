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
driftflow initdb create   # crea una base (idempotente)
driftflow initdb drop     # elimina una base con confirmación
driftflow initdb backup   # genera backup completo (según driver)
driftflow initdb restore  # restaura desde backup (según driver)
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

Para administración segura de bases (`initdb`) con `postgres`, `mysql`, `sqlserver` y `mongodb`:

```bash
driftflow initdb create
driftflow initdb create --db my_tenant_db
driftflow initdb drop --db my_tenant_db
driftflow initdb drop --db my_tenant_db --yes
driftflow initdb backup --output backups/app.sql
driftflow initdb backup --db my_tenant_db --output backups/tenant.sql
driftflow initdb restore --file backups/app.sql
driftflow initdb restore --db my_tenant_db --file backups/tenant.sql --yes
```

### Especificaciones de `initdb` por motor

Ejemplo de `connection_string` compatible (PostgreSQL):

```text
postgres://postgres:postgres@postgres:5432/apexbuildr_auth_service?sslmode=disable
```

- Soporte de drivers: `postgres`, `mysql`, `sqlserver`, `mongodb` (usa el `--driver` global o `DB_TYPE` del `.env`).
- `create` es idempotente: si la base ya existe imprime `Database already exists` y termina en éxito.
- `drop` siempre requiere `--db`, y por defecto pide confirmación explícita escribiendo el nombre exacto de la base (se puede omitir con `--yes` o `--force`).
- `backup` requiere `--output`; `restore` requiere `--file` y confirmación por defecto (`y` o `--yes/--force`).
- Validación de `--db`: solo letras/números/underscore (`^[a-zA-Z0-9_]+$`) y bloqueo de nombres reservados (`postgres`, `template0`, `template1`).

Herramientas utilizadas por motor:

- PostgreSQL: `pg_dump` (backup) y `psql` (restore).
- MySQL: `mysqldump` (backup) y `mysql` (restore).
- SQL Server: backup/restore ejecutados por SQL (`BACKUP DATABASE` / `RESTORE DATABASE`) mediante la conexión configurada.
- MongoDB: `mongodump` (backup) y `mongorestore` (restore).


### Uso de `initdb` desde Go (microservicio de aprovisionamiento)

Además del CLI, puedes usar `DbAdminService` directamente desde tu código para aprovisionar múltiples bases (por ejemplo, una por tenant) y luego crear tablas.

```go
package provisioning

import (
    "context"
    "fmt"
    "log"
    "net/url"

    driftflow "github.com/misaelcrespo30/DriftFlow"
    "github.com/misaelcrespo30/DriftFlow/config"
)

// Ejemplo de modelo de dominio
type TenantUser struct {
    ID    uint   `gorm:"primaryKey"`
    Email string `gorm:"uniqueIndex"`
}

func ProvisionTenantDatabases(ctx context.Context, dbNames []string) error {
    cfg := config.Load()

    admin := driftflow.NewDbAdminService(cfg.DSN, cfg.Driver)

    for _, dbName := range dbNames {
        created, err := admin.Create(ctx, dbName)
        if err != nil {
            return fmt.Errorf("create %s: %w", dbName, err)
        }

        if !created {
            log.Printf("database %s already exists (idempotente)", dbName)
        }

        // Conectar a la DB recién creada (o existente)
        tenantDSN, err := dsnWithDatabase(cfg.Driver, cfg.DSN, dbName)
        if err != nil {
            return err
        }

        tenantDB, err := driftflow.ConnectToDB(tenantDSN, cfg.Driver)
        if err != nil {
            return fmt.Errorf("connect %s: %w", dbName, err)
        }

        // Crear tablas del tenant
        if err := tenantDB.AutoMigrate(&TenantUser{}); err != nil {
            return fmt.Errorf("migrate %s: %w", dbName, err)
        }
    }

    return nil
}

func dsnWithDatabase(driver, dsn, dbName string) (string, error) {
    u, err := url.Parse(dsn)
    if err != nil {
        return "", err
    }

    switch driver {
    case "postgres", "mysql":
        u.Path = "/" + dbName
    case "sqlserver":
        q := u.Query()
        q.Set("database", dbName)
        u.RawQuery = q.Encode()
    default:
        return "", fmt.Errorf("unsupported driver: %s", driver)
    }

    return u.String(), nil
}
```

Este flujo te permite orquestar aprovisionamiento desde un microservicio sin depender del comando `driftflow initdb` en shell.

## Uso como librería

### Patrón recomendado para usar DriftFlow desde otro microservicio

Si tu microservicio de aprovisionamiento no quiere invocar el binario CLI, puedes usar DriftFlow como librería y encadenar `generate` + `up` + `seed` en código Go:

```go
package provisioning

import (
    "fmt"

    driftflow "github.com/misaelcrespo30/DriftFlow"
    "github.com/misaelcrespo30/DriftFlow/config"
    "github.com/misaelcrespo30/DriftFlow/helpers"
)

// ProvisionSchemaAndData crea/actualiza esquema y datos iniciales de una base.
func ProvisionSchemaAndData() error {
    cfg := config.Load() // reutiliza .env y la misma resolución de DSN/driver

    db, err := driftflow.ConnectToDB(cfg.DSN, cfg.Driver)
    if err != nil {
        return fmt.Errorf("connect db: %w", err)
    }

    // 1) generate: construir migraciones desde modelos
    models, err := helpers.LoadModels()
    if err != nil {
        return fmt.Errorf("load models: %w", err)
    }

    if err := driftflow.GenerateModelMigrations(models, driftflow.GenerateOptions{
        Dir:          cfg.MigDir,
        ManifestMode: driftflow.ManifestStrict,
        Engine:       cfg.Driver,
    }); err != nil {
        return fmt.Errorf("generate migrations: %w", err)
    }

    // 2) up: aplicar migraciones pendientes
    if err := driftflow.Up(db, cfg.MigDir); err != nil {
        return fmt.Errorf("apply migrations: %w", err)
    }

    // 3) seed: poblar datos iniciales
    if err := driftflow.Seed(db, cfg.SeedRunDir); err != nil {
        return fmt.Errorf("run seeds: %w", err)
    }

    return nil
}
```

Este mecanismo permite que cualquier microservicio consumidor de DriftFlow haga provisioning completo sin depender del CLI.

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
