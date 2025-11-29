# SQLitREST - Blueprint Complet & Audit Intégré

**Version** : 1.0-FINAL
**Date** : 2025-11-29
**Statut** : Prêt pour implémentation (avec révisions intégrées)

---

## 📋 EXECUTIVE SUMMARY

**SQLitREST** est un clone open-source de PostgREST pour SQLite, offrant une API REST automatique avec policies RLS-like, auth multi-méthode, et endpoints debug LLM.

**Verdict Audit Cerebras GLM-4.6** : Architecture solide mais nécessite révisions critiques sur :
1. **Sécurité** : Moteur de policies (injection SQL via AST, pas string concat)
2. **Timeline** : 12 semaines réalistes (vs 8 semaines initialement)
3. **Scope** : Approche MVP+ progressive (vs v1.0 complète immédiate)

**Différenciation clé** : "PostgREST for SQLite with declarative policies and LLM debugging"

---

## 🎯 POSITIONNEMENT

### Public Cible
- Développeurs fullstack (prototypage rapide)
- Startups (backend léger sans infra lourde)
- Tooling interne (dashboards, admin panels)
- Apps JAMstack (backend SQLite local-first)

### Concurrence & Différenciation

| Feature | PostgREST | Datasette | sqlite-http | **SQLitREST** |
|---------|-----------|-----------|-------------|---------------|
| Database | PostgreSQL | SQLite | SQLite | **SQLite** |
| RLS | ✅ Natif PG | ❌ | ❌ | ✅ **App-side** |
| Auth | JWT | Plugin | ❌ | ✅ **Multi** (JWT+APIKey+Basic) |
| Policies | ✅ Déclaratif | ❌ | ❌ | ✅ **TOML déclaratif** |
| Write | ✅ | ❌ Read-only | ⚠️ Limité | ✅ **CRUD complet** |
| OpenAPI | ✅ | ⚠️ | ❌ | ✅ **Auto-gen** |
| Debug LLM | ❌ | ⚠️ UI | ❌ | ✅ **`/_debug/*` natif** |
| Multi-DB | ✅ Schemas | ⚠️ | ❌ | ✅ **ATTACH natif** |
| Deployment | Binaire | Python | C lib | **Binaire CGO-free** |

**Positionnement** : Combler le gap entre Datasette (read-only, simple) et PostgREST (PostgreSQL).

---

## 🏗️ ARCHITECTURE TECHNIQUE

### Stack Validée

```go
// go.mod (8 dépendances minimalistes)
zombiezen.com/go/sqlite v1.4.0    // Driver (CGO-free via modernc)
go-chi/chi/v5 v5.1.0              // HTTP router
go-chi/cors v1.2.1                // CORS
go-chi/httprate v0.14.1           // Rate limiting
golang-jwt/jwt/v5 v5.2.1          // Auth JWT
pelletier/go-toml/v2 v2.2.2       // Config TOML
rs/zerolog v1.33.0                // Logging structuré
getkin/kin-openapi/v3 v3.0.3      // OpenAPI generation
```

**AJOUT POST-AUDIT** :
```go
vitess.io/vitess/go/vt/sqlparser  // SQL AST parser (sécurité policies)
```

### Architecture Globale (Révisée)

```
┌────────────────────────────────────────────────────────┐
│                   HTTP Request                         │
│   GET /posts?author=eq.john&select=*,comments(*)       │
└─────────────────────┬──────────────────────────────────┘
                      ▼
┌────────────────────────────────────────────────────────┐
│               Middleware Stack                         │
│  ┌────────┐ ┌─────────┐ ┌──────┐ ┌──────────┐        │
│  │ Logger │→│Recovery │→│ CORS │→│RateLimit │        │
│  └────────┘ └─────────┘ └──────┘ └──────────┘        │
│  ┌────────┐ ┌─────────────┐ ┌───────────────┐        │
│  │  Auth  │→│ DBResolver  │→│PolicyEnforcer │        │
│  │(fluide)│  │(PoolManager)│  │ (AST-based)   │        │
│  └────────┘ └─────────────┘ └───────────────┘        │
└─────────────────────┬──────────────────────────────────┘
                      ▼
┌────────────────────────────────────────────────────────┐
│              Query Builder + AST Integration           │
│  ┌─────────┐ ┌────────┐ ┌────────┐ ┌──────────┐      │
│  │ SELECT  │ │ INSERT │ │ UPDATE │ │  DELETE  │      │
│  └─────────┘ └────────┘ └────────┘ └──────────┘      │
│                                                         │
│  Parse URL → Build SQL → Parse AST → Inject Policies  │
│  → Generate Final SQL → Execute                        │
└─────────────────────┬──────────────────────────────────┘
                      ▼
┌────────────────────────────────────────────────────────┐
│                PoolManager (Révisé)                     │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │
│  │   Pool A     │ │   Pool B     │ │   Pool C     │  │
│  │              │ │              │ │              │  │
│  │ 1 writer     │ │ 1 writer     │ │ 1 writer     │  │
│  │ 5 readers    │ │ 5 readers    │ │ 5 readers    │  │
│  │              │ │              │ │              │  │
│  │ main.db      │ │ logs.db      │ │ analytics.db │  │
│  └──────────────┘ └──────────────┘ └──────────────┘  │
│                                                         │
│  + AttachDB(name, path) / DetachDB(name) thread-safe  │
│  + Schema cache in-memory (refresh on demand)         │
└─────────────────────┬──────────────────────────────────┘
                      ▼
┌────────────────────────────────────────────────────────┐
│            zombiezen.com/go/sqlite                      │
│              (modernc pure Go)                          │
└────────────────────────────────────────────────────────┘
```

### Changements Majeurs Post-Audit

| Composant | Avant | Après (Révisé) |
|-----------|-------|----------------|
| **PolicyEnforcer** | String concat WHERE | **AST parser (vitess)** |
| **DBResolver** | Router statique | **PoolManager dynamique** |
| **Schema** | Query à chaque req | **Cache in-memory** |
| **UDF Registry** | Tout exposé | **Allow-list TOML** |
| **JWT Validation** | Basique | **Claims complets + rotation** |

---

## 🔐 SÉCURITÉ (CRITIQUE - POST-AUDIT)

### 1. Moteur de Policies (RÉVISION MAJEURE)

**❌ Approche Initiale (DANGEREUSE)** :
```go
// NE PAS FAIRE ÇA
func (e *PolicyEngine) BuildWhereClause(policies []Policy) string {
    var conditions []string
    for _, p := range policies {
        conditions = append(conditions, "("+p.Using+")")  // INJECTION !
    }
    return strings.Join(conditions, " OR ")
}
```

**✅ Approche Sécurisée (AST-based)** :
```go
// internal/policy/ast_injector.go
import "vitess.io/vitess/go/vt/sqlparser"

func (e *PolicyEngine) InjectPolicies(
    baseSQL string,
    policies []Policy,
) (string, error) {
    // 1. Parser la requête de base en AST
    stmt, err := sqlparser.Parse(baseSQL)
    if err != nil {
        return "", err
    }

    selectStmt, ok := stmt.(*sqlparser.Select)
    if !ok {
        return "", errors.New("not a SELECT statement")
    }

    // 2. Parser chaque policy en AST
    var policyExprs []sqlparser.Expr
    for _, p := range policies {
        policyAST, err := sqlparser.ParseExpr(p.Using)
        if err != nil {
            return "", fmt.Errorf("invalid policy %s: %w", p.Name, err)
        }
        policyExprs = append(policyExprs, policyAST)
    }

    // 3. Combiner en OR
    var combinedPolicy sqlparser.Expr
    if len(policyExprs) > 0 {
        combinedPolicy = policyExprs[0]
        for _, expr := range policyExprs[1:] {
            combinedPolicy = &sqlparser.OrExpr{
                Left:  combinedPolicy,
                Right: expr,
            }
        }
    }

    // 4. Intégrer dans l'AST de base
    if selectStmt.Where == nil {
        selectStmt.Where = &sqlparser.Where{
            Type: sqlparser.WhereClause,
            Expr: combinedPolicy,
        }
    } else {
        selectStmt.Where.Expr = &sqlparser.AndExpr{
            Left:  selectStmt.Where.Expr,
            Right: combinedPolicy,
        }
    }

    // 5. Régénérer SQL
    return sqlparser.String(selectStmt), nil
}
```

**Avantages** :
- ✅ Zéro risque d'injection SQL
- ✅ Validation syntaxique automatique
- ✅ Gestion propre des parenthèses
- ✅ Support sous-requêtes sécurisé

### 2. UDF Registry (Allow-list Obligatoire)

**Configuration TOML** :
```toml
# sqlitrest.toml

[[udf]]
name = "current_user_id"
expose = true
readonly = true
description = "Returns current authenticated user ID"

[[udf]]
name = "current_role"
expose = true
readonly = true

[[udf]]
name = "sha256"
expose = true
readonly = true
deterministic = true

[[udf]]
name = "dangerous_mutation"
expose = false  # Pas exposé en /rpc/*
readonly = false
```

**Implémentation** :
```go
// internal/udf/registry.go
type UDFConfig struct {
    Name          string
    Expose        bool  // Si false, pas d'endpoint /rpc/{name}
    Readonly      bool  // Si false, requiert policy mutation
    Deterministic bool
}

func (r *Registry) LoadFromConfig(cfg []UDFConfig) {
    for _, udfCfg := range cfg {
        r.configs[udfCfg.Name] = udfCfg
    }
}

func (r *Registry) IsExposed(name string) bool {
    cfg, exists := r.configs[name]
    return exists && cfg.Expose
}

// Handler RPC
func (h *RPCHandler) Handle(w http.ResponseWriter, r *http.Request) {
    funcName := chi.URLParam(r, "function")

    if !h.udfRegistry.IsExposed(funcName) {
        http.Error(w, "function not found or not exposed", 404)
        return
    }
    // ...
}
```

### 3. JWT Validation Complète

**Claims standards validés** :
```go
// internal/auth/jwt.go
type JWTValidator struct {
    config JWTConfig
    keySet *jwk.AutoRefresh  // Pour RS256 JWKS rotation
}

func (v *JWTValidator) Validate(tokenString string) (*Claims, error) {
    token, err := jwt.Parse(tokenString, v.keyFunc)
    if err != nil {
        return nil, err
    }

    claims := token.Claims.(jwt.MapClaims)

    // Valider exp (expiration)
    if exp, ok := claims["exp"].(float64); ok {
        if time.Now().Unix() > int64(exp) {
            return nil, errors.New("token expired")
        }
    } else {
        return nil, errors.New("missing exp claim")
    }

    // Valider nbf (not before)
    if nbf, ok := claims["nbf"].(float64); ok {
        if time.Now().Unix() < int64(nbf) {
            return nil, errors.New("token not yet valid")
        }
    }

    // Valider iss (issuer)
    if v.config.Issuer != "" {
        if iss, ok := claims["iss"].(string); !ok || iss != v.config.Issuer {
            return nil, errors.New("invalid issuer")
        }
    }

    // Valider aud (audience)
    if v.config.Audience != "" {
        if aud, ok := claims["aud"].(string); !ok || aud != v.config.Audience {
            return nil, errors.New("invalid audience")
        }
    }

    return &Claims{
        Subject: claims["sub"].(string),
        Role:    claims[v.config.RoleClaim].(string),
        All:     claims,
    }, nil
}
```

### 4. Column Masking Sécurisé

**Mécanisme** :
```go
// internal/policy/column_filter.go
func (e *PolicyEngine) FilterColumns(
    table string,
    requestedCols []string,
    identity *Identity,
) (allowedCols []string, masked map[string]string) {
    policies := e.Applicable(identity, table, OpSelect)

    allowed := make(map[string]bool)
    masked = make(map[string]string)

    for _, p := range policies {
        if p.Columns == nil {
            // Pas de restriction colonnes
            for _, col := range requestedCols {
                allowed[col] = true
            }
            continue
        }

        // Whitelist
        for _, col := range p.Columns.Visible {
            allowed[col] = true
        }

        // Blacklist
        for _, col := range p.Columns.Hidden {
            delete(allowed, col)
        }

        // Masking
        for col, maskExpr := range p.Columns.Masked {
            masked[col] = maskExpr
            allowed[col] = true // Visible mais masqué
        }
    }

    for col := range allowed {
        allowedCols = append(allowedCols, col)
    }

    return
}

// Dans le Query Builder
func (b *SelectBuilder) Build() (string, error) {
    // ...
    allowedCols, masked := b.policyEngine.FilterColumns(
        b.table,
        b.requestedCols,
        b.identity,
    )

    var selectCols []string
    for _, col := range allowedCols {
        if maskExpr, ok := masked[col]; ok {
            selectCols = append(selectCols, fmt.Sprintf("%s AS %s", maskExpr, col))
        } else {
            selectCols = append(selectCols, col)
        }
    }
    // ...
}
```

---

## ⚡ PERFORMANCE (POST-AUDIT)

### Targets Révisés (Réalistes)

| Benchmark | Target Initial | **Target Révisé** | Justification |
|-----------|----------------|-------------------|---------------|
| GET simple | 8k req/s, <8ms | **5k req/s, <12ms** | Overhead AST + cache |
| GET filtered | 5k req/s, <12ms | **3k req/s, <18ms** | Policies injection |
| GET embedded | 2k req/s, <25ms | **1.5k req/s, <30ms** | JOIN complexity |
| Mutations | 3k req/s, <15ms | **1.5k req/s, <20ms** | Single writer bottleneck |

**Mesures** : Benchmarks à réaliser sur hardware de référence (ex: AWS c5.large).

### Cache de Schéma (CRITIQUE)

```go
// internal/db/schema_cache.go
type SchemaCache struct {
    mu     sync.RWMutex
    tables map[string]*TableSchema
}

type TableSchema struct {
    Name        string
    Columns     []ColumnInfo
    PrimaryKeys []string
    ForeignKeys []ForeignKeyInfo
    Indexes     []IndexInfo
}

func (c *SchemaCache) Refresh(conn *sqlite.Conn) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 1. Lire sqlite_master
    tables := make(map[string]*TableSchema)

    err := sqlitex.Execute(conn,
        "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'",
        &sqlitex.ExecOptions{
            ResultFunc: func(stmt *sqlite.Stmt) error {
                tableName := stmt.ColumnText(0)
                tables[tableName] = &TableSchema{Name: tableName}
                return nil
            },
        },
    )
    if err != nil {
        return err
    }

    // 2. Pour chaque table : PRAGMA table_info
    for tableName, schema := range tables {
        err := sqlitex.Execute(conn,
            fmt.Sprintf("PRAGMA table_info(%s)", tableName),
            &sqlitex.ExecOptions{
                ResultFunc: func(stmt *sqlite.Stmt) error {
                    schema.Columns = append(schema.Columns, ColumnInfo{
                        Name:    stmt.ColumnText(1),
                        Type:    stmt.ColumnText(2),
                        NotNull: stmt.ColumnInt(3) == 1,
                        PK:      stmt.ColumnInt(5) == 1,
                    })
                    return nil
                },
            },
        )
        if err != nil {
            return err
        }

        // 3. PRAGMA foreign_key_list
        // ... (similaire)
    }

    c.tables = tables
    return nil
}

func (c *SchemaCache) GetTable(name string) (*TableSchema, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    t, ok := c.tables[name]
    return t, ok
}
```

**Utilisation** :
- Refresh au démarrage du DBManager
- Endpoint `POST /_debug/reload-schema` (requiert admin)
- Signal `SIGHUP` pour reload à chaud

### Gestion WAL Checkpointing

```go
// internal/db/wal_manager.go
type WALManager struct {
    interval time.Duration
    done     chan struct{}
}

func (m *WALManager) Start(pool *ManagedPool) {
    ticker := time.NewTicker(m.interval) // ex: 5 minutes
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            _ = pool.Write(context.Background(), func(conn *sqlite.Conn) error {
                // Checkpoint passif (ne bloque pas les readers)
                return sqlitex.ExecuteTransient(conn, "PRAGMA wal_checkpoint(PASSIVE)", nil)
            })
        case <-m.done:
            return
        }
    }
}
```

---

## 📋 TIMELINE RÉVISÉE (12 SEMAINES)

### Phase 0 : Setup & Prototyping (Semaine 1)
- [ ] Init projet Go (modules, structure)
- [ ] Setup CI/CD (GitHub Actions)
- [ ] Prototyper AST policy injection (POC sécurité)
- [ ] Valider zombiezen pools (benchmarks basiques)

### Phase 1 : Core v0.1 (Semaines 2-3)
- [ ] DBManager + PoolManager (create/destroy pools)
- [ ] Schema cache in-memory
- [ ] Introspection complète (tables, columns, FK)
- [ ] Query builder SELECT basique (sans policies)
- [ ] HTTP router Chi + handlers CRUD
- [ ] Tests unitaires (coverage >70%)

**Livrable** : CRUD simple, pas d'auth, pas de policies.

### Phase 2 : Filtres & Pagination v0.1 (Semaine 4)
- [ ] Parser filtres PostgREST (eq, gt, lt, like, in, is)
- [ ] Opérateurs logiques (and, or, not)
- [ ] Pagination (limit/offset + Range header)
- [ ] Tri (order + nulls first/last)
- [ ] Tests intégration filtres

**Livrable** : API compatible PostgREST (filtres).

### Phase 3 : Auth v0.2 (Semaine 5)
- [ ] Auth chain fluide (JWT + API Key + Basic)
- [ ] Identity + context propagation
- [ ] Validation JWT complète (exp, nbf, iss, aud)
- [ ] JWKS auto-refresh (RS256)
- [ ] Tests auth (unit + e2e)

**Livrable** : Auth fonctionnelle, identity dans context.

### Phase 4-5 : Policies v0.3 (Semaines 6-9) **CRITIQUE**
- [ ] Parser TOML policies
- [ ] AST injection (vitess sqlparser)
- [ ] UDF context-aware (current_user_id, etc.)
- [ ] UDF allow-list (config TOML)
- [ ] Column-level masking/hiding
- [ ] Tests sécurité exhaustifs (injection SQL, bypass)
- [ ] Audit sécurité externe (bug bounty private)

**Livrable** : Moteur de policies robuste, testé, audité.

### Phase 6 : Features Avancées v0.4 (Semaine 10)
- [ ] Embedding FK (LEFT JOIN automatique)
- [ ] Sélection colonnes (?select=a,b,rel(*))
- [ ] UDF exposées (/rpc/*)
- [ ] Multi-DB dynamique (ATTACH/DETACH)
- [ ] Tests intégration multi-DB

**Livrable** : Toutes features fonctionnelles.

### Phase 7 : OpenAPI & Debug v0.5 (Semaine 11)
- [ ] Génération OpenAPI 3.0
- [ ] Endpoints debug LLM (/_debug/*)
- [ ] Healthcheck verbose
- [ ] Métriques pools
- [ ] WAL checkpointing automatique

**Livrable** : API complète + debug.

### Phase 8 : Polish & Release v1.0 (Semaine 12)
- [ ] Documentation complète (API.md, FILTERS.md, POLICIES.md, AUTH.md)
- [ ] Exemples variés (blog, todo, multi-tenant)
- [ ] Docker image Alpine (<20 MB)
- [ ] Benchmarks officiels (publier résultats)
- [ ] Release binaires multi-plateformes (Linux, macOS, Windows)
- [ ] Website + landing page
- [ ] Communication (Show HN, Reddit, Twitter)

**Livrable** : v1.0 production-ready.

---

## 🔧 STRUCTURE PROJET FINALE

```
sqlitrest/
├── cmd/
│   └── sqlitrest/
│       └── main.go                    # CLI + bootstrap
│
├── internal/
│   ├── config/
│   │   ├── config.go                  # Struct config
│   │   ├── loader.go                  # TOML + env + flags
│   │   └── defaults.go                # Valeurs par défaut
│   │
│   ├── db/
│   │   ├── pool_manager.go            # PoolManager (REVISED)
│   │   ├── pool.go                    # ManagedPool (1 writer + N readers)
│   │   ├── schema_cache.go            # Cache schéma in-memory (NEW)
│   │   ├── attach.go                  # Cross-DB queries (ATTACH/DETACH)
│   │   ├── introspect.go              # Lecture schéma SQLite
│   │   ├── schema.go                  # Types (Table, Column, FK)
│   │   ├── pragmas.go                 # Configuration SQLite
│   │   └── wal_manager.go             # WAL checkpointing (NEW)
│   │
│   ├── udf/
│   │   ├── registry.go                # Registre UDF + allow-list (REVISED)
│   │   ├── builtin.go                 # UDF built-in (auth, utils)
│   │   └── types.go                   # Types UDF
│   │
│   ├── auth/
│   │   ├── identity.go                # Type Identity
│   │   ├── chain.go                   # AuthChain
│   │   ├── middleware.go              # HTTP middleware
│   │   ├── jwt.go                     # JWT authenticator (REVISED: claims complets)
│   │   ├── apikey.go                  # API Key authenticator
│   │   └── basic.go                   # Basic auth
│   │
│   ├── policy/
│   │   ├── types.go                   # Types Policy
│   │   ├── engine.go                  # PolicyEngine
│   │   ├── parser.go                  # TOML → Policy
│   │   ├── ast_injector.go            # AST-based injection (NEW - CRITICAL)
│   │   ├── column_filter.go           # Column masking/hiding (NEW)
│   │   ├── middleware.go              # HTTP middleware
│   │   └── sql.go                     # WHERE injection (DEPRECATED → AST)
│   │
│   ├── query/
│   │   ├── builder.go                 # Interface QueryBuilder
│   │   ├── select.go                  # SELECT builder
│   │   ├── insert.go                  # INSERT builder
│   │   ├── update.go                  # UPDATE builder
│   │   ├── delete.go                  # DELETE builder
│   │   ├── embed.go                   # JOINs (FK embedding)
│   │   └── executor.go                # Exécution avec policies
│   │
│   ├── parser/
│   │   ├── request.go                 # Parse request complète
│   │   ├── select.go                  # ?select=...
│   │   ├── filter.go                  # ?col=op.value
│   │   ├── order.go                   # ?order=...
│   │   ├── pagination.go              # ?limit/offset + Range
│   │   └── operators.go               # eq, gt, like, in, fts...
│   │
│   ├── api/
│   │   ├── router.go                  # Chi router setup
│   │   ├── handlers.go                # CRUD handlers
│   │   ├── rpc.go                     # /rpc/* handlers
│   │   ├── schema.go                  # /_schema handlers
│   │   ├── debug.go                   # /_debug/* handlers
│   │   ├── health.go                  # /health
│   │   ├── errors.go                  # Error formatting
│   │   └── middleware/
│   │       ├── logger.go              # Request logging
│   │       ├── recovery.go            # Panic recovery
│   │       ├── cors.go                # CORS
│   │       ├── ratelimit.go           # Rate limiting
│   │       └── timeout.go             # Request timeout
│   │
│   └── openapi/
│       ├── generator.go               # Schema → OpenAPI
│       ├── types.go                   # SQLite → JSON Schema
│       └── serve.go                   # /openapi.json handler
│
├── examples/
│   ├── basic/
│   │   ├── blog.db                    # DB exemple blog
│   │   ├── README.md                  # Quick start
│   │   └── test.sh                    # Script de test
│   │
│   ├── policies/
│   │   ├── multi-tenant.toml          # Exemple multi-tenant
│   │   ├── rbac.toml                  # Exemple RBAC
│   │   └── privacy.toml               # Exemple masquage données
│   │
│   └── docker/
│       ├── Dockerfile                 # Image Alpine (~20 MB)
│       └── docker-compose.yml         # Setup complet
│
├── docs/
│   ├── API.md                         # Documentation API REST
│   ├── FILTERS.md                     # Guide filtres PostgREST
│   ├── POLICIES.md                    # Guide policies RLS-like
│   ├── AUTH.md                        # Guide authentification
│   ├── UDF.md                         # Guide UDF
│   ├── SECURITY.md                    # Security best practices (NEW)
│   ├── DEPLOYMENT.md                  # Guide déploiement
│   └── COMPARISON.md                  # vs PostgREST, Datasette
│
├── tests/
│   ├── unit/                          # Tests unitaires
│   ├── integration/                   # Tests intégration
│   ├── e2e/                           # Tests end-to-end
│   └── security/                      # Tests sécurité (injection, bypass) (NEW)
│
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Tests + lint
│       ├── security.yml               # Audit sécurité automatique (NEW)
│       ├── release.yml                # Binaires multi-plateformes
│       └── docker.yml                 # Image Docker
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE                            # MIT
├── CONTRIBUTING.md
├── SECURITY.md                        # Security policy (NEW)
├── CHANGELOG.md
└── Makefile
```

---

## 📝 CONFIGURATION FINALE

### sqlitrest.toml (Révisé)

```toml
[server]
port = 8080
shutdown_timeout = "30s"

# Databases
[databases.main]
path = "./main.db"
mode = "readwrite"

[databases.logs]
path = "./logs.db"
mode = "readonly"

# Pool configuration
[pool]
readers_per_db = 5
busy_timeout = "5s"
wal_checkpoint_interval = "5m"  # NEW

# Auth
[auth]
enabled = true

[auth.jwt]
enabled = true
algorithm = "HS256"
secret = "env:JWT_SECRET"
audience = "sqlitrest"
issuer = "https://auth.example.com"
role_claim = "role"
validate_exp = true   # NEW
validate_nbf = true   # NEW
validate_iss = true   # NEW
validate_aud = true   # NEW

# JWKS for RS256 (NEW)
# jwks_url = "https://auth.example.com/.well-known/jwks.json"
# jwks_refresh_interval = "1h"

[auth.apikey]
enabled = true
header = "X-API-Key"
source = "file:./apikeys.toml"

# Policies
[policies]
file = "./policies.toml"
default_action = "deny"

# UDF Allow-list (NEW)
[[udf]]
name = "current_user_id"
expose = true
readonly = true

[[udf]]
name = "current_role"
expose = true
readonly = true

[[udf]]
name = "sha256"
expose = true
readonly = true
deterministic = true

# Logging
[log]
level = "info"
format = "json"

# Middleware
[middleware]
cors_origins = ["*"]
rate_limit_per_ip = 100
request_timeout = "30s"

# Debug
[debug]
enabled = true
require_admin = true
```

---

## 🎯 MÉTRIQUES DE SUCCÈS

### Performance (Révisée)

```
Benchmark               Target      Mesure
────────────────────────────────────────────
GET simple              5k req/s    Benchmark avec hey/k6
GET filtered            3k req/s    Idem
GET embedded            1.5k req/s  Idem
Mutations               1.5k req/s  Idem

Latency (p95)
────────────────────────────────────────────
GET simple              <12ms       Idem
GET filtered            <18ms       Idem
GET embedded            <30ms       Idem
Mutations               <20ms       Idem
```

### Adoption (6 mois)

```
Métrique                Target
────────────────────────────────
GitHub stars            500+
Weekly downloads        1000+
Production deploys      50+
Contributors            10+
Doc coverage            100%
Test coverage           80%+
Security audits         2+ (interne + externe)
```

---

## 🚀 PRÊT POUR IMPLÉMENTATION

### Checklist Avant Démarrage

- [x] Architecture validée (audit Cerebras intégré)
- [x] Révisions sécurité critiques identifiées
- [x] Timeline réaliste (12 semaines)
- [x] Stack technique confirmée (+ vitess AST parser)
- [x] Structure projet définie
- [x] Métriques de succès établies

### Prochaines Étapes

1. **Créer le repo GitHub** : `yourname/sqlitrest`
2. **Init projet Go** : `go mod init github.com/yourname/sqlitrest`
3. **Setup CI/CD** : GitHub Actions (tests, lint, sécurité)
4. **Phase 0** : Prototyper AST injection (validation sécurité)
5. **Phase 1** : Démarrer implémentation Core v0.1

---

## 📚 SOURCES & RÉFÉRENCES

### Documentation Technique
- [PostgREST Documentation](https://postgrest.org/)
- [zombiezen go-sqlite](https://pkg.go.dev/zombiezen.com/go/sqlite)
- [Vitess SQL Parser](https://pkg.go.dev/vitess.io/vitess/go/vt/sqlparser)
- [Go Chi Router](https://github.com/go-chi/chi)

### Audit & Analyses
- Audit Cerebras GLM-4.6 (2025-11-29)
- Analyse architecture V2 autoclaude
- Recommandations Claude (moi)

### Inspiration Projets
- [PostgREST GitHub](https://github.com/PostgREST/postgrest)
- [Datasette](https://datasette.io/)
- [Soul SQLite REST](https://github.com/thevahidal/soul)

---

**Document produit par** : Claude (Anthropic)
**Audit réalisé par** : Cerebras GLM-4.6
**Date** : 2025-11-29
**Statut** : Production-ready blueprint

**Chemin du document** : `/home/cl-ment/horos_40/sqlitrest/SQLITREST_BLUEPRINT_COMPLET.md`
