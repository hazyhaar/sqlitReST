# SQLitREST - Architecture Technique Détaillée

**Version** : MVP Phase 1
**Stack** : Go + zombiezen.com/go/sqlite + chi
**Philosophie** : Simple, rapide, évolutif

---

## 📐 Vue d'Ensemble

```
┌──────────────────────────────────────────────┐
│             HTTP Client                       │
│   (curl, browser, MCP, UI web)               │
└────────────────┬─────────────────────────────┘
                 │
                 ▼
┌──────────────────────────────────────────────┐
│          Chi HTTP Router                      │
│  GET /users?age=gt.18&select=name,email      │
└────────────────┬─────────────────────────────┘
                 │
         ┌───────┴────────┐
         ▼                ▼
┌─────────────┐    ┌─────────────┐
│   Query     │    │    UDF      │
│   Builder   │    │  Registry   │
└──────┬──────┘    └──────┬──────┘
       │                  │
       ▼                  ▼
┌──────────────────────────────────┐
│      SQLite Connection           │
│   (zombiezen Pool + WAL)         │
└──────────────┬───────────────────┘
               │
               ▼
        ┌──────────────┐
        │  my_app.db   │
        │   (WAL mode) │
        └──────────────┘
```

---

## 🗂️ Structure du Projet

```
sqlitrest/
├── cmd/
│   └── sqlitrest/
│       └── main.go                    # Entry point + CLI
│
├── internal/
│   ├── schema/
│   │   ├── introspector.go            # Lecture schéma SQLite
│   │   ├── types.go                   # Table, Column, FK structs
│   │   └── cache.go                   # Cache du schéma (refresh on-demand)
│   │
│   ├── query/
│   │   ├── builder.go                 # SQL query construction
│   │   ├── filter.go                  # Parsing ?col=op.value
│   │   ├── select.go                  # Parsing ?select=a,b,rel(*)
│   │   ├── order.go                   # Parsing ?order=col.desc
│   │   ├── pagination.go              # limit/offset handling
│   │   └── embed.go                   # FK joins via LEFT JOIN
│   │
│   ├── api/
│   │   ├── router.go                  # Chi router setup
│   │   ├── handlers.go                # HTTP handlers (CRUD)
│   │   ├── middleware.go              # CORS, logging, errors
│   │   ├── response.go                # JSON response formatting
│   │   └── openapi.go                 # OpenAPI 3.0 generation
│   │
│   ├── udf/
│   │   ├── registry.go                # Go function → SQL function
│   │   ├── builtin.go                 # Built-in UDFs (sha256, uuid, etc.)
│   │   └── rpc.go                     # /rpc/* endpoint handlers
│   │
│   └── db/
│       ├── conn.go                    # zombiezen pool wrapper
│       ├── pragmas.go                 # WAL + SYNCHRONOUS setup
│       └── stats.go                   # Connection pool metrics
│
├── pkg/                               # Public API (si bibliothèque)
│   └── sqlitrest/
│       ├── client.go                  # Go client pour l'API
│       └── types.go                   # Types publics
│
├── examples/
│   ├── basic/
│   │   ├── app.db                     # SQLite exemple
│   │   └── README.md                  # Quick start
│   ├── horos/
│   │   ├── expose-events.sh           # Exposer horos_events.db
│   │   └── custom-udfs.go             # UDFs Horos spécifiques
│   └── docker/
│       ├── Dockerfile                 # Image Alpine (15 MB)
│       └── docker-compose.yml         # Démo complète
│
├── docs/
│   ├── API.md                         # Documentation API REST
│   ├── FILTERS.md                     # Guide filtres PostgREST
│   ├── UDF.md                         # Guide UDF Go
│   └── DEPLOYMENT.md                  # Guide déploiement
│
├── tests/
│   ├── schema_test.go
│   ├── query_test.go
│   └── integration_test.go
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE                            # MIT
└── Makefile                           # build, test, lint, install
```

---

## 🧩 Composants Clés

### 1. Introspecteur de Schéma

```go
// internal/schema/introspector.go
package schema

import "zombiezen.com/go/sqlite"

type Introspector struct {
    conn *sqlite.Conn
}

type Table struct {
    Name        string
    Columns     []Column
    PrimaryKeys []string
    ForeignKeys []ForeignKey
    Indexes     []Index
}

type Column struct {
    Name         string
    Type         string
    NotNull      bool
    DefaultValue *string
    IsPrimaryKey bool
}

type ForeignKey struct {
    Column      string
    RefTable    string
    RefColumn   string
    OnDelete    string
    OnUpdate    string
}

func (i *Introspector) GetTables() ([]Table, error) {
    // 1. SELECT name FROM sqlite_master WHERE type='table'
    // 2. Pour chaque table : PRAGMA table_info(name)
    // 3. Pour chaque table : PRAGMA foreign_key_list(name)
    // 4. Pour chaque table : PRAGMA index_list(name)
}

func (i *Introspector) GetTable(name string) (*Table, error) {
    // Introspection d'une table spécifique
}
```

### 2. Query Builder

```go
// internal/query/builder.go
package query

type SelectQuery struct {
    Table       string
    Columns     []string       // SELECT cols
    Filters     []Filter       // WHERE conditions
    Ordering    []OrderBy      // ORDER BY
    Limit       int            // LIMIT
    Offset      int            // OFFSET
    Embeds      []Embed        // FK joins
}

type Filter struct {
    Column   string
    Operator string  // eq, gt, like, in, is, etc.
    Value    any
}

type OrderBy struct {
    Column    string
    Direction string  // asc, desc
    NullsPos  string  // first, last
}

type Embed struct {
    Relation   string   // Table à joindre
    Columns    []string // Colonnes à sélectionner
    ForeignKey ForeignKey
}

func (q *SelectQuery) Build() (sql string, args []any, err error) {
    // Génération SQL avec prepared statements
    // SELECT ... FROM ... LEFT JOIN ... WHERE ... ORDER BY ... LIMIT ... OFFSET ...
}

// Parsing URL → SelectQuery
func ParseURL(r *http.Request, table string) (*SelectQuery, error) {
    q := &SelectQuery{Table: table}

    // ?select=name,email,company(name)
    if sel := r.URL.Query().Get("select"); sel != "" {
        q.Columns, q.Embeds = parseSelect(sel)
    }

    // ?age=gt.18&status=eq.active
    for key, vals := range r.URL.Query() {
        if isFilterKey(key) {
            q.Filters = append(q.Filters, parseFilter(key, vals[0]))
        }
    }

    // ?order=created_at.desc,name.asc.nullslast
    if ord := r.URL.Query().Get("order"); ord != "" {
        q.Ordering = parseOrder(ord)
    }

    // ?limit=10&offset=20
    q.Limit = parseIntParam(r, "limit", 100)  // default 100
    q.Offset = parseIntParam(r, "offset", 0)

    return q, nil
}
```

### 3. Gestionnaire de Connexions

```go
// internal/db/conn.go
package db

import (
    "zombiezen.com/go/sqlite"
    "zombiezen.com/go/sqlite/sqlitex"
)

type Manager struct {
    pool     *sqlitex.Pool
    dbPath   string
    readOnly bool
}

func NewManager(dbPath string, readOnly bool) (*Manager, error) {
    flags := sqlite.OpenReadWrite
    if readOnly {
        flags = sqlite.OpenReadOnly
    }

    pool, err := sqlitex.NewPool(dbPath, sqlitex.PoolOptions{
        Flags:    flags,
        PoolSize: 10,  // 10 connexions read en parallèle
        PrepareConn: func(conn *sqlite.Conn) error {
            // Pragmas obligatoires
            return sqlitex.ExecuteTransient(conn, `
                PRAGMA journal_mode=WAL;
                PRAGMA synchronous=NORMAL;
                PRAGMA foreign_keys=ON;
                PRAGMA busy_timeout=5000;
            `, nil)
        },
    })

    return &Manager{pool: pool, dbPath: dbPath, readOnly: readOnly}, err
}

func (m *Manager) Get() *sqlite.Conn {
    return m.pool.Get(context.Background())
}

func (m *Manager) Put(conn *sqlite.Conn) {
    m.pool.Put(conn)
}

func (m *Manager) Close() error {
    return m.pool.Close()
}
```

### 4. HTTP Handlers

```go
// internal/api/handlers.go
package api

import (
    "net/http"
    "github.com/go-chi/chi/v5"
)

type API struct {
    db          *db.Manager
    schema      *schema.Introspector
    udfRegistry *udf.Registry
}

func (api *API) HandleGetRows(w http.ResponseWriter, r *http.Request) {
    tableName := chi.URLParam(r, "table")

    // 1. Valider que la table existe
    table, err := api.schema.GetTable(tableName)
    if err != nil {
        http.Error(w, "Table not found", 404)
        return
    }

    // 2. Parser URL → SelectQuery
    selectQuery, err := query.ParseURL(r, tableName)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }

    // 3. Build SQL
    sql, args, err := selectQuery.Build()
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }

    // 4. Exécuter
    conn := api.db.Get()
    defer api.db.Put(conn)

    rows := []map[string]any{}
    err = sqlitex.Execute(conn, sql, &sqlitex.ExecOptions{
        Args: args,
        ResultFunc: func(stmt *sqlite.Stmt) error {
            row := map[string]any{}
            for i := 0; i < stmt.ColumnCount(); i++ {
                row[stmt.ColumnName(i)] = stmt.ColumnValue(i)
            }
            rows = append(rows, row)
            return nil
        },
    })

    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // 5. Retourner JSON
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rows)
}

func (api *API) HandleInsertRow(w http.ResponseWriter, r *http.Request) {
    // POST /table → INSERT INTO table
}

func (api *API) HandleUpdateRow(w http.ResponseWriter, r *http.Request) {
    // PUT /table?id=eq.123 → UPDATE table SET ... WHERE id = 123
}

func (api *API) HandleDeleteRow(w http.ResponseWriter, r *http.Request) {
    // DELETE /table?id=eq.123 → DELETE FROM table WHERE id = 123
}
```

### 5. UDF Registry

```go
// internal/udf/registry.go
package udf

import "zombiezen.com/go/sqlite"

type Registry struct {
    functions map[string]UDF
}

type UDF struct {
    Name        string
    NumArgs     int
    GoFunc      func(args ...any) (any, error)
    Deterministic bool
}

func NewRegistry() *Registry {
    r := &Registry{functions: make(map[string]UDF)}

    // UDFs built-in
    r.Register(UDF{
        Name:    "uuid_v4",
        NumArgs: 0,
        GoFunc: func(...any) (any, error) {
            return uuid.New().String(), nil
        },
        Deterministic: false,
    })

    r.Register(UDF{
        Name:    "sha256",
        NumArgs: 1,
        GoFunc: func(args ...any) (any, error) {
            data := args[0].(string)
            hash := sha256.Sum256([]byte(data))
            return hex.EncodeToString(hash[:]), nil
        },
        Deterministic: true,
    })

    return r
}

func (r *Registry) Register(udf UDF) {
    r.functions[udf.Name] = udf
}

func (r *Registry) InstallInConnection(conn *sqlite.Conn) error {
    for name, udf := range r.functions {
        err := conn.CreateFunction(name, &sqlite.FunctionImpl{
            NArgs:         udf.NumArgs,
            Deterministic: udf.Deterministic,
            Scalar: func(ctx sqlite.Context, args []sqlite.Value) (sqlite.Value, error) {
                goArgs := make([]any, len(args))
                for i, arg := range args {
                    goArgs[i] = arg.Text()  // Simplification
                }
                result, err := udf.GoFunc(goArgs...)
                if err != nil {
                    return sqlite.Value{}, err
                }
                return sqlite.TextValue(result.(string)), nil
            },
        })
        if err != nil {
            return err
        }
    }
    return nil
}

// Endpoint RPC pour appeler UDF via HTTP
func (r *Registry) HandleRPC(w http.ResponseWriter, req *http.Request) {
    funcName := chi.URLParam(req, "function")

    udf, exists := r.functions[funcName]
    if !exists {
        http.Error(w, "Function not found", 404)
        return
    }

    // Parser body JSON → args
    var body struct {
        Args []any `json:"args"`
    }
    json.NewDecoder(req.Body).Decode(&body)

    // Exécuter UDF
    result, err := udf.GoFunc(body.Args...)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    json.NewEncoder(w).Encode(map[string]any{"result": result})
}
```

### 6. Router Setup

```go
// internal/api/router.go
package api

import (
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
)

func (api *API) SetupRouter() *chi.Mux {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: []string{"*"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
    }))

    // Meta endpoints
    r.Get("/", api.HandleRoot)                    // Infos serveur
    r.Get("/tables", api.HandleListTables)        // Liste tables
    r.Get("/tables/{table}", api.HandleTableInfo) // Schéma table
    r.Get("/openapi.json", api.HandleOpenAPI)     // OpenAPI spec

    // CRUD endpoints (générés dynamiquement)
    r.Route("/{table}", func(r chi.Router) {
        r.Get("/", api.HandleGetRows)
        r.Post("/", api.HandleInsertRow)
        r.Put("/", api.HandleUpdateRow)
        r.Delete("/", api.HandleDeleteRow)
    })

    // RPC endpoints
    r.Post("/rpc/{function}", api.udfRegistry.HandleRPC)

    return r
}
```

---

## 🔍 Exemples d'Utilisation

### Cas 1 : Exposer une base SQLite existante

```bash
# Télécharger le binaire
curl -L https://github.com/yourname/sqlitrest/releases/latest/download/sqlitrest-linux -o sqlitrest
chmod +x sqlitrest

# Lancer le serveur
./sqlitrest my_app.db --port 8080

# Tester
curl http://localhost:8080/tables
curl http://localhost:8080/users?age=gt.18&select=name,email
```

### Cas 2 : Intégration Horos

```bash
# Exposer horos_events.db en read-only
./sqlitrest \
  /data/horos/system/horos_events.db \
  --port 8080 \
  --readonly

# Utilisation depuis MCP ou UI
curl http://localhost:8080/tasks?status=eq.pending&order=priority.desc&limit=10
curl http://localhost:8080/heartbeats?entity_id=eq.audit_manager
curl http://localhost:8080/logs?level=eq.ERROR&limit=50
```

### Cas 3 : UDF personnalisée

```go
// custom_udf.go
package main

import (
    "github.com/yourname/sqlitrest/pkg/sqlitrest"
)

func main() {
    api := sqlitrest.New("app.db")

    // Enregistrer UDF custom
    api.RegisterUDF(sqlitrest.UDF{
        Name: "send_email",
        NumArgs: 2,
        GoFunc: func(args ...any) (any, error) {
            to := args[0].(string)
            subject := args[1].(string)
            // ... logique envoi email
            return "Email sent", nil
        },
    })

    api.Start(":8080")
}
```

```bash
# Appel via RPC
curl -X POST http://localhost:8080/rpc/send_email \
  -H "Content-Type: application/json" \
  -d '{"args": ["user@example.com", "Hello"]}'
```

---

## 📊 Génération OpenAPI

```go
// internal/api/openapi.go
package api

func (api *API) GenerateOpenAPI() OpenAPISpec {
    tables, _ := api.schema.GetTables()

    spec := OpenAPISpec{
        OpenAPI: "3.0.0",
        Info: Info{
            Title:   "SQLitREST API",
            Version: "1.0.0",
        },
        Paths: make(map[string]PathItem),
    }

    for _, table := range tables {
        // GET /table
        spec.Paths["/"+table.Name] = PathItem{
            Get: Operation{
                Summary: "List " + table.Name,
                Parameters: []Parameter{
                    {Name: "select", In: "query", Schema: StringSchema},
                    {Name: "order", In: "query", Schema: StringSchema},
                    {Name: "limit", In: "query", Schema: IntSchema},
                    {Name: "offset", In: "query", Schema: IntSchema},
                },
                // Générer un paramètre par colonne pour filtrage
            },
            Post: Operation{
                Summary: "Insert " + table.Name,
                RequestBody: RequestBody{
                    Content: map[string]MediaType{
                        "application/json": {
                            Schema: generateSchemaFromTable(table),
                        },
                    },
                },
            },
        }
    }

    return spec
}
```

---

## 🚀 Commandes CLI

```bash
# Build
go build -o sqlitrest cmd/sqlitrest/main.go

# Usage basique
sqlitrest <database.db>                  # Port 8080 par défaut

# Options
sqlitrest app.db \
  --port 8080 \                          # Port HTTP
  --readonly \                           # Mode lecture seule
  --cors-origin "https://example.com" \  # CORS spécifique
  --log-level debug                      # Verbosité logs

# Multi-DB (Phase 2)
sqlitrest \
  --db events=horos_events.db:readonly \
  --db meta=horos_meta.db:readonly \
  --db tickets=ops_tickets.db

# Avec config file
sqlitrest --config sqlitrest.toml
```

---

## 📦 Dépendances Minimales

```go
// go.mod
module github.com/yourname/sqlitrest

go 1.23

require (
    zombiezen.com/go/sqlite v1.4.0      // SQLite driver
    github.com/go-chi/chi/v5 v5.1.0     // HTTP router
    github.com/go-chi/cors v1.2.1       // CORS
    github.com/google/uuid v1.6.0       // UDF uuid_v4
)
```

**Total** : 4 dépendances directes seulement.

---

## ⚡ Performance Attendue

### Benchmarks cibles (SQLite WAL, machine moderne)

| Opération | Req/sec | Latence p95 |
|-----------|---------|-------------|
| GET simple (SELECT *) | 5000-8000 | <10ms |
| GET avec filtres | 3000-5000 | <15ms |
| GET avec embedding | 1000-2000 | <30ms |
| POST (INSERT) | 2000-4000 | <20ms |
| PUT (UPDATE) | 2000-4000 | <20ms |

**Limitation** : SQLite lui-même (writes sérialisés), pas le serveur HTTP.

---

## 🎯 Points Décision

Avant implémentation, valider :

1. ✅ **Langage** : Go confirmé ?
2. ✅ **Scope Phase 1** : MVP mono-user validé ?
3. ✅ **Driver** : zombiezen OK ? (vs modernc)
4. ✅ **gRPC** : Reporter en Phase 4 ?
5. ✅ **Multi-DB** : Reporter en Phase 2 ?
6. ✅ **Auth** : Aucune en Phase 1, API key en Phase 3 ?

**Ready to code dès validation.** 🚀
