# SQLitREST - Recommandations sur Architecture V2 Autoclaude

**Date** : 2025-11-29
**Auteur** : Claude (analyse critique)
**Source** : Architecture V2 zombiezen-centric (autoclaude)

---

## 🎯 Synthèse Architecture V2

### Points Forts Majeurs ✅

1. **zombiezen au centre** : Choix cohérent, pool natif, UDF ergonomique
2. **Pattern 1 writer + N readers** : Excellente solution contention WAL
3. **Auth fluide** : Approche "context enrichment" très élégante (jamais bloquant sauf token invalide)
4. **Policies par filtrage** : WHERE injection vs blocage brutal → meilleure UX
5. **Modularité par interfaces** : Architecture propre, testable, découplée
6. **Multi-DB natif** : Gestion pool par DB, isolation physique

### Architecture Globale

```
Request
   ↓
[Middleware Stack]
   ├─ Logger
   ├─ Recovery
   ├─ CORS
   ├─ RateLimit
   ├─ Auth (enrichit context: Anonymous|User)
   ├─ Policy (injecte WHERE clauses)
   └─ DBResolver
   ↓
Handler
   ↓
DBManager
   ├─ Pool A (1 writer + 5 readers)
   ├─ Pool B (1 writer + 5 readers)
   └─ Pool C (1 writer + 5 readers)
   ↓
zombiezen/sqlite → modernc
```

---

## 📊 Analyse Critique Détaillée

### 1. Pattern Auth Fluide ✅ Excellent

**Concept** :
```
Pas de header JWT → Anonymous context → Accès selon policies
JWT invalide      → 401 Unauthorized (seul cas de rejet)
JWT valide        → User context → Accès selon policies
```

**Pourquoi c'est bon** :
- Flexibilité maximale (mono-user ET multi-user)
- UX progressive (pas besoin d'auth pour démarrer)
- Chain of Responsibility élégante (JWT → APIKey → Basic → Anonymous)

**Recommandation** : ✅ **Adopter tel quel**

---

### 2. Policies par Filtrage ✅ Excellent

**Mécanisme** :
```sql
-- Sans policy
SELECT * FROM posts

-- Avec policy (role anon)
SELECT * FROM posts WHERE published = true

-- Avec policy (role authenticated)
SELECT * FROM posts WHERE (published = true) OR (author_id = current_user_id())
```

**Pourquoi c'est bon** :
- Pas de 403 frustrants (seulement des résultats vides si pas d'accès)
- Combinaison OR des policies (approche permissive)
- UDF context-aware (`current_user_id()`, `current_role()`)

**Recommandation** : ✅ **Adopter avec ajouts**

**Ajouts proposés** :
1. **Policy par défaut explicite** : Si aucune policy → comportement par défaut configurable
   ```toml
   [policies]
   default_action = "allow"  # ou "deny"
   ```

2. **Policy debugging** : Endpoint pour visualiser policies applicables
   ```http
   GET /_debug/policies?db=main&table=posts&operation=SELECT&role=anon
   ```

---

### 3. Pattern 1 Writer + N Readers ✅ Excellent

**Code proposé** :
```go
type ManagedPool struct {
    writer   *sqlite.Conn      // Single writer
    readers  *sqlitex.Pool     // N readers
    writeMu  sync.Mutex        // Sérialise writes
}

func (p *ManagedPool) Write(ctx context.Context, fn func(*sqlite.Conn) error) error {
    p.writeMu.Lock()
    defer p.writeMu.Unlock()
    return fn(p.writer)
}
```

**Recommandation** : ✅ **Adopter avec optimisation**

**Optimisation proposée** : **Write queue avec batching optionnel**

```go
type ManagedPool struct {
    writer   *sqlite.Conn
    readers  *sqlitex.Pool
    writeQ   chan writeOp      // Queue async optionnelle
}

type writeOp struct {
    fn       func(*sqlite.Conn) error
    resultCh chan error
}

// Mode sync (par défaut)
func (p *ManagedPool) Write(ctx context.Context, fn func(*sqlite.Conn) error) error {
    if p.writeQ == nil {
        return p.writeSync(fn)
    }
    return p.writeAsync(ctx, fn)
}

// Mode async (optionnel, config)
func (p *ManagedPool) writeAsync(ctx context.Context, fn func(*sqlite.Conn) error) error {
    op := writeOp{fn: fn, resultCh: make(chan error, 1)}
    select {
    case p.writeQ <- op:
        return <-op.resultCh
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

**Justification** : Permet batching automatique en mode high-throughput (optionnel).

---

### 4. ATTACH/DETACH Cross-DB ⚠️ Bon mais incomplet

**Code proposé** :
```go
func (m *DBManager) CrossDBQuery(
    ctx context.Context,
    primaryDB string,
    attachDBs []string,
    fn func(conn *sqlite.Conn, aliases map[string]string) error,
) error {
    // ATTACH des DBs secondaires
    // defer DETACH
}
```

**Recommandation** : ⚠️ **Adopter avec garde-fous**

**Problèmes potentiels** :
1. **Limite SQLite** : 10 ATTACH max par défaut (SQLITE_MAX_ATTACHED)
2. **Contention** : ATTACH peut bloquer si write en cours sur DB attachée
3. **Pas de timeout** : Peut deadlock

**Améliorations proposées** :

```go
// 1. Vérifier limite ATTACH
const MaxAttach = 10

func (m *DBManager) CrossDBQuery(...) error {
    if len(attachDBs) > MaxAttach {
        return ErrTooManyAttach
    }

    // 2. Timeout sur ATTACH
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    // 3. Mode READ-ONLY pour ATTACH
    for _, name := range attachDBs {
        alias := "attached_" + name
        err := sqlitex.ExecuteTransient(conn,
            fmt.Sprintf("ATTACH DATABASE ? AS %s MODE ro", alias),
            &sqlitex.ExecOptions{Args: []interface{}{pool.Path}},
        )
        // ...
    }
}
```

---

### 5. UDF avec Context Propagation ✅ Excellent

**Pattern** :
```go
// UDF peut accéder au context Go (Identity, etc.)
registry.Register("current_user_id", func(ctx context.Context) (string, error) {
    return IdentityFrom(ctx).ID, nil
})
```

**Recommandation** : ✅ **Adopter avec extensions**

**Extensions proposées** :

```go
// 1. UDF de debugging
registry.Register("_debug_context", func(ctx context.Context) (string, error) {
    id := IdentityFrom(ctx)
    return fmt.Sprintf("User: %s, Role: %s, Method: %s", id.ID, id.Role, id.Method), nil
})

// 2. UDF de tracing
registry.Register("_trace", func(ctx context.Context, msg string) (string, error) {
    log.Debug("SQL trace", "message", msg, "user", IdentityFrom(ctx).ID)
    return msg, nil  // Passthrough
})

// Usage SQL :
-- SELECT _trace('Fetching posts'), * FROM posts WHERE author_id = current_user_id()
```

---

### 6. gRPC en Parallèle de REST ⚠️ Questionable

**Code proposé** :
```go
// REST server
router.Serve(cfg.HTTP.Port)

// gRPC server (parallèle)
grpcServer.Serve(cfg.GRPC.Port)
```

**Recommandation** : ⚠️ **Revoir priorités**

**Question** : **Quel besoin réel pour gRPC ?**

| Cas d'usage | REST | gRPC | Gagnant |
|-------------|------|------|---------|
| UI web | ✅ Parfait | ❌ Complexe | REST |
| CLI | ✅ curl | ⚠️ grpcurl | REST |
| Horos MCP | ✅ JSON | ❌ Protobuf | REST |
| Streaming | ⚠️ SSE/WebSocket | ✅ Bidirectionnel | gRPC |
| Perf extrême | ⚠️ JSON overhead | ✅ Protobuf compact | gRPC |

**Recommandations** :

1. **Phase 1** : REST uniquement
2. **Phase 2** : Si besoin streaming validé → gRPC
3. **Alternative** : Server-Sent Events (SSE) pour streaming léger

```go
// SSE endpoint (alternative à gRPC streaming)
GET /_stream/{table}?filters=...

// Client reçoit :
data: {"event": "insert", "table": "posts", "row": {...}}
data: {"event": "update", "table": "posts", "row": {...}}
```

**Verdict** : gRPC = **Phase 2 optionnelle**, pas MVP.

---

### 7. Configuration TOML ✅ Excellent

**Format proposé** :
```toml
[databases.main]
path = "./main.db"
mode = "readwrite"

[auth.jwt]
enabled = true
secret = "env:JWT_SECRET"
role_claim = "role"

[[policy]]
name = "posts_public_read"
table = "posts"
operations = ["SELECT"]
using = "published = true"
```

**Recommandation** : ✅ **Adopter avec validation**

**Améliorations** :

```toml
# 1. Validation au démarrage
[validation]
strict = true          # Fail si policy référence UDF inexistante
check_policies = true  # Vérifier syntaxe SQL des policies

# 2. Env var expansion
[databases.prod]
path = "env:DB_PATH"  # Expansion automatique

# 3. Includes
[policies]
include = ["./policies/*.toml"]  # Charger tous les fichiers
```

---

## 🚨 Points de Vigilance

### 1. Complexité vs Besoin Réel

**Observation** : Architecture très complète (REST + gRPC + Policies + Multi-DB + Auth chain)

**Question** : Quel est le **premier** cas d'usage concret ?

Si c'est "exposer `horos_events.db` en read-only pour une UI" :
- ✅ Besoin : REST, Multi-DB
- ❌ Pas besoin (Phase 1) : gRPC, Policies complexes, Auth multi-méthode

**Recommandation** : **Approche incrémentale validée**

```
Phase 1 (2 semaines)
├─ Core : DBManager + zombiezen pools
├─ REST : CRUD basique + filtres PostgREST
├─ Multi-DB : Pattern Horos (4-BDD)
└─ OpenAPI : Auto-generation

Phase 2 (1 semaine)
├─ Auth : Chain basique (JWT OU APIKey)
└─ Policies : Filtrage simple

Phase 3 (si besoin validé)
├─ gRPC : Streaming
└─ Policies : Column-level masking
```

### 2. Tests & Validation

**Manquant dans l'architecture V2** : Stratégie de tests

**Recommandation** : Ajouter section tests

```go
// Test strategy
tests/
├── unit/
│   ├── db/pool_test.go              # Tests pool isolation
│   ├── auth/chain_test.go           # Tests auth chain
│   └── policy/engine_test.go        # Tests policies
│
├── integration/
│   ├── crud_test.go                 # Tests CRUD complets
│   ├── filters_test.go              # Tests opérateurs PostgREST
│   └── multidb_test.go              # Tests ATTACH/DETACH
│
└── e2e/
    ├── horos_integration_test.go    # Test cas Horos réel
    └── policies_test.go             # Test RLS-like behavior
```

### 3. Performance Benchmarks

**Recommandation** : Ajouter critères de performance

```go
// Performance targets
Benchmark                   Target
─────────────────────────────────────
GET simple (no filter)      10k req/s
GET with filters (3 conds)  5k req/s
GET with embedding (1 FK)   2k req/s
POST insert                 4k req/s
ATTACH cross-DB query       500 req/s

// Latency targets (p95)
GET simple                  <5ms
GET filtered                <10ms
GET embedded                <20ms
```

---

## 🎯 Endpoints Debug LLM Systématiques

### Philosophie

Les LLM (Claude, GPT, Gemini) ont besoin de **visibilité structurée** sur :
1. Schéma des données (tables, colonnes, FK)
2. Policies appliquées (qui peut voir quoi)
3. État du système (connections, pools)
4. Trace des requêtes (debugging)

### Endpoints Proposés

```
/_debug/
├── schema                # Schéma complet toutes DBs
├── schema/{db}           # Schéma d'une DB
├── schema/{db}/{table}   # Schéma d'une table
├── policies              # Toutes les policies
├── policies/eval         # Évaluer policies pour un contexte
├── pools                 # État des pools de connexions
├── auth                  # Testeur d'authentification
├── query                 # Explainer SQL
├── context               # Context actuel (Identity, etc.)
└── health/verbose        # Healthcheck détaillé
```

### 1. GET /_debug/schema

**Description** : Schéma complet explorable par LLM

**Response** :
```json
{
  "databases": [
    {
      "name": "main",
      "path": "/data/main.db",
      "mode": "readwrite",
      "tables": [
        {
          "name": "users",
          "columns": [
            {"name": "id", "type": "INTEGER", "pk": true, "notnull": true},
            {"name": "email", "type": "TEXT", "unique": true, "notnull": true},
            {"name": "role", "type": "TEXT", "default": "'user'"}
          ],
          "foreign_keys": [],
          "indexes": [
            {"name": "idx_users_email", "columns": ["email"], "unique": true}
          ]
        },
        {
          "name": "posts",
          "columns": [
            {"name": "id", "type": "INTEGER", "pk": true},
            {"name": "author_id", "type": "INTEGER", "notnull": true},
            {"name": "title", "type": "TEXT"},
            {"name": "published", "type": "INTEGER", "default": "0"}
          ],
          "foreign_keys": [
            {
              "from": "author_id",
              "to_table": "users",
              "to_column": "id",
              "on_delete": "CASCADE"
            }
          ]
        }
      ]
    }
  ],
  "relationships": [
    {
      "from_table": "posts",
      "from_column": "author_id",
      "to_table": "users",
      "to_column": "id",
      "type": "many_to_one"
    }
  ],
  "stats": {
    "total_tables": 12,
    "total_columns": 87,
    "total_foreign_keys": 15
  }
}
```

**Usage LLM** :
```
User: "Quelles sont les tables disponibles ?"
LLM → GET /_debug/schema → Parse JSON → "Il y a 12 tables : users, posts, ..."
```

### 2. GET /_debug/policies

**Description** : Liste toutes les policies avec leur logique

**Response** :
```json
{
  "policies": [
    {
      "name": "posts_public_read",
      "database": "*",
      "table": "posts",
      "operations": ["SELECT"],
      "roles": ["anon", "authenticated"],
      "using": "published = true",
      "check": null,
      "columns": null
    },
    {
      "name": "posts_owner_all",
      "database": "*",
      "table": "posts",
      "operations": ["SELECT", "INSERT", "UPDATE", "DELETE"],
      "roles": ["authenticated"],
      "using": "author_id = current_user_id()",
      "check": "author_id = current_user_id()",
      "columns": null
    }
  ],
  "summary": {
    "total_policies": 8,
    "by_table": {
      "posts": 2,
      "users": 3,
      "comments": 3
    },
    "by_role": {
      "anon": 3,
      "authenticated": 4,
      "admin": 1
    }
  }
}
```

### 3. POST /_debug/policies/eval

**Description** : Évaluer quelles policies s'appliquent pour un contexte

**Request** :
```json
{
  "identity": {
    "id": "user-123",
    "role": "authenticated"
  },
  "database": "main",
  "table": "posts",
  "operation": "SELECT"
}
```

**Response** :
```json
{
  "applicable_policies": [
    {
      "name": "posts_public_read",
      "using": "published = true"
    },
    {
      "name": "posts_owner_all",
      "using": "author_id = current_user_id()"
    }
  ],
  "combined_where": "(published = true) OR (author_id = 'user-123')",
  "final_sql_example": "SELECT * FROM posts WHERE (published = true) OR (author_id = 'user-123')",
  "access_granted": true,
  "reason": "2 policies allow access"
}
```

**Usage LLM** :
```
User: "Puis-je voir tous les posts ?"
LLM → POST /_debug/policies/eval → Parse → "Oui, vous pouvez voir les posts publiés + les vôtres"
```

### 4. GET /_debug/pools

**Description** : État des pools de connexions (monitoring)

**Response** :
```json
{
  "pools": [
    {
      "database": "main",
      "writer": {
        "active": true,
        "in_transaction": false,
        "last_write": "2025-11-29T17:30:45Z"
      },
      "readers": {
        "total": 5,
        "active": 2,
        "idle": 3,
        "queue_depth": 0
      },
      "stats": {
        "total_reads": 12458,
        "total_writes": 1247,
        "avg_read_latency_ms": 3.2,
        "avg_write_latency_ms": 12.5
      }
    }
  ],
  "health": {
    "status": "healthy",
    "issues": []
  }
}
```

**Usage LLM** :
```
User: "Le serveur est lent, pourquoi ?"
LLM → GET /_debug/pools → "Queue depth = 0, latence normale → pas de problème DB"
```

### 5. POST /_debug/auth

**Description** : Tester l'authentification

**Request** :
```json
{
  "method": "jwt",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response** :
```json
{
  "valid": true,
  "identity": {
    "id": "user-123",
    "role": "authenticated",
    "method": "jwt",
    "claims": {
      "sub": "user-123",
      "email": "user@example.com",
      "role": "authenticated",
      "exp": 1735574400
    }
  },
  "expires_at": "2025-12-30T12:00:00Z",
  "time_until_expiry": "23h 30m"
}
```

**Usage LLM** :
```
User: "Mon token JWT fonctionne-t-il ?"
LLM → POST /_debug/auth → "Oui, valide jusqu'au 30/12/2025"
```

### 6. POST /_debug/query

**Description** : Expliquer une requête SQL (EXPLAIN QUERY PLAN)

**Request** :
```json
{
  "database": "main",
  "sql": "SELECT * FROM posts WHERE author_id = ? AND published = true",
  "params": [123]
}
```

**Response** :
```json
{
  "query": "SELECT * FROM posts WHERE author_id = ? AND published = true",
  "explain_plan": [
    "SEARCH posts USING INDEX idx_posts_author_id (author_id=?)",
    "Filter: published = true"
  ],
  "uses_index": true,
  "estimated_rows": 42,
  "warnings": [],
  "suggestions": []
}
```

**Usage LLM** :
```
User: "Cette requête est-elle optimisée ?"
LLM → POST /_debug/query → "Oui, utilise l'index idx_posts_author_id"
```

### 7. GET /_debug/context

**Description** : Context actuel de la requête (Identity, headers, etc.)

**Response** :
```json
{
  "identity": {
    "id": "user-123",
    "role": "authenticated",
    "method": "jwt"
  },
  "request": {
    "method": "GET",
    "path": "/_debug/context",
    "headers": {
      "Authorization": "Bearer eyJ...",
      "User-Agent": "curl/7.68.0"
    },
    "remote_addr": "192.168.1.100:54321"
  },
  "server": {
    "version": "1.0.0",
    "uptime": "2h 34m 12s",
    "databases": ["main", "logs", "analytics"]
  }
}
```

**Usage LLM** :
```
User: "Qui suis-je connecté en tant que ?"
LLM → GET /_debug/context → "Vous êtes user-123 (role: authenticated)"
```

### 8. GET /_debug/health/verbose

**Description** : Healthcheck détaillé (vs `/health` minimaliste)

**Response** :
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h 34m 12s",
  "checks": {
    "databases": {
      "status": "healthy",
      "details": [
        {"name": "main", "status": "ok", "size_mb": 245},
        {"name": "logs", "status": "ok", "size_mb": 1024}
      ]
    },
    "pools": {
      "status": "healthy",
      "active_connections": 7,
      "idle_connections": 8
    },
    "auth": {
      "status": "healthy",
      "providers": ["jwt", "apikey"]
    },
    "policies": {
      "status": "healthy",
      "loaded": 8
    }
  },
  "metrics": {
    "total_requests": 125847,
    "requests_per_sec": 42,
    "avg_latency_ms": 8.5,
    "error_rate": 0.002
  }
}
```

---

## 🔧 Implémentation Endpoints Debug

### Middleware de protection

```go
// internal/api/middleware/debug.go

func DebugOnly(allowInProduction bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Option 1 : Bloquer en production
            if !allowInProduction && os.Getenv("ENV") == "production" {
                http.Error(w, "Debug endpoints disabled in production", 403)
                return
            }

            // Option 2 : Requiert auth admin
            identity := auth.IdentityFrom(r.Context())
            if identity.Role != "admin" {
                http.Error(w, "Debug endpoints require admin role", 403)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

### Router debug

```go
// internal/api/rest/debug.go

func SetupDebugRoutes(r chi.Router, deps DebugDeps) {
    r.Route("/_debug", func(r chi.Router) {
        r.Use(middleware.DebugOnly(false))  // Désactivé en prod

        r.Get("/schema", handleDebugSchema(deps.DBManager))
        r.Get("/schema/{db}", handleDebugSchemaDB(deps.DBManager))
        r.Get("/schema/{db}/{table}", handleDebugSchemaTable(deps.DBManager))

        r.Get("/policies", handleDebugPolicies(deps.PolicyEngine))
        r.Post("/policies/eval", handleDebugPoliciesEval(deps.PolicyEngine))

        r.Get("/pools", handleDebugPools(deps.DBManager))
        r.Post("/auth", handleDebugAuth(deps.AuthChain))
        r.Post("/query", handleDebugQuery(deps.DBManager))
        r.Get("/context", handleDebugContext())
        r.Get("/health/verbose", handleHealthVerbose(deps))
    })
}
```

---

## 📋 Recommandations Finales

### Priorités Phase 1 (2 semaines)

1. ✅ **Core DBManager** : Pattern 1 writer + N readers
2. ✅ **REST CRUD** : Endpoints basiques + filtres PostgREST
3. ✅ **Multi-DB** : Support pattern Horos (4-BDD)
4. ✅ **Endpoints debug** : `/_debug/schema`, `/_debug/pools`, `/_debug/context`
5. ✅ **OpenAPI** : Auto-génération depuis schéma

### Reporter en Phase 2-3

1. ⏸️ **gRPC** : Attendre besoin streaming validé
2. ⏸️ **Auth chain complète** : Commencer JWT simple
3. ⏸️ **Policies column-level** : Commencer row-level simple

### Décisions à Valider

1. **gRPC ou pas** : Quel besoin concret ?
2. **Policies dès Phase 1** : Ou commencer sans (mode ouvert) ?
3. **Env de dev** : Production immédiate ou prototypage ?

---

## 🎯 Verdict Global

**Architecture V2 autoclaude** : ✅ **Excellente base technique**

**Points forts** :
- zombiezen bien utilisé
- Auth fluide innovante
- Policies par filtrage élégantes
- Modularité propre

**Points d'attention** :
- Scope large (gRPC optionnel ?)
- Besoin validation cas d'usage réel
- Tests à ajouter

**Recommandation** : ✅ **Adopter avec approche incrémentale**

Phase 1 (Core + REST + Debug) → Valider → Phase 2 (Auth + Policies) → Valider → Phase 3 (gRPC si besoin)

---

**Prêt à implémenter dès validation du scope Phase 1.** 🚀
