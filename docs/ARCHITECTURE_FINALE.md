# SQLitREST - Architecture Finale (Production-Ready)

**Positionnement** : Clone open-source de PostgREST pour SQLite
**Public cible** : Développeurs généralistes, startups, tooling interne
**Philosophie** : Fonctionnalité complète dès le départ, production-ready

---

## 🎯 Scope Fonctionnel Final

### Fonctionnalités Essentielles (v1.0)

| Feature | PostgREST | SQLitREST | Priorité |
|---------|-----------|-----------|----------|
| **CRUD automatique** | ✅ | ✅ | P0 |
| **Filtres riches** (eq, gt, like, in, fts) | ✅ | ✅ | P0 |
| **Pagination** (offset, keyset) | ✅ | ✅ | P0 |
| **Tri** (nulls first/last) | ✅ | ✅ | P0 |
| **Embedding FK** (relations) | ✅ | ✅ | P0 |
| **Operators logiques** (and, or, not) | ✅ | ✅ | P0 |
| **RLS-like policies** | ✅ (natif PG) | ✅ (app-side) | P0 |
| **Auth JWT** | ✅ | ✅ | P0 |
| **OpenAPI auto-gen** | ✅ | ✅ | P0 |
| **Multi-DB** | ✅ (schemas) | ✅ (ATTACH) | P0 |
| **UDF exposées** (/rpc/*) | ✅ | ✅ | P0 |
| **Endpoints debug LLM** | ❌ | ✅ | P0 |
| **gRPC** | ❌ | ❌ | Hors scope |
| **WebSocket/Streaming** | ❌ | ❌ | v2.0 |

---

## 🏗️ Stack Technique Finale

```go
// go.mod
module github.com/yourname/sqlitrest

go 1.23

require (
    // SQLite driver
    zombiezen.com/go/sqlite v1.4.0

    // HTTP framework
    github.com/go-chi/chi/v5 v5.1.0
    github.com/go-chi/cors v1.2.1
    github.com/go-chi/httprate v0.14.1

    // Auth
    github.com/golang-jwt/jwt/v5 v5.2.1

    // Config
    github.com/pelletier/go-toml/v2 v2.2.2

    // Logging
    github.com/rs/zerolog v1.33.0

    // OpenAPI
    github.com/getkin/kin-openapi/v3 v3.0.3
)
```

**Total** : 8 dépendances (minimaliste mais complet)

---

## 📐 Architecture Globale

```
┌──────────────────────────────────────────────────────────┐
│                    HTTP Request                          │
│         GET /posts?author=eq.john&select=*               │
└───────────────────────┬──────────────────────────────────┘
                        ▼
┌──────────────────────────────────────────────────────────┐
│                  Middleware Stack                         │
│  ┌────────────┐  ┌─────────┐  ┌──────┐  ┌──────────┐    │
│  │   Logger   │→│Recovery │→│ CORS │→│RateLimit │    │
│  └────────────┘  └─────────┘  └──────┘  └──────────┘    │
│  ┌────────────┐  ┌─────────────┐  ┌────────────────┐    │
│  │    Auth    │→│ DBResolver  │→│PolicyEnforcer  │    │
│  │  (fluide)  │  │(multi-DB)   │  │ (WHERE inject) │    │
│  └────────────┘  └─────────────┘  └────────────────┘    │
└───────────────────────┬──────────────────────────────────┘
                        ▼
┌──────────────────────────────────────────────────────────┐
│                   Query Builder                          │
│  ┌─────────┐  ┌────────┐  ┌────────┐  ┌──────────┐      │
│  │ SELECT  │  │ INSERT │  │ UPDATE │  │  DELETE  │      │
│  │ builder │  │builder │  │builder │  │ builder  │      │
│  └─────────┘  └────────┘  └────────┘  └──────────┘      │
│                                                           │
│  Parse URL → Build SQL + policies → Execute              │
└───────────────────────┬──────────────────────────────────┘
                        ▼
┌──────────────────────────────────────────────────────────┐
│                    DBManager                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Pool A     │  │   Pool B     │  │   Pool C     │   │
│  │              │  │              │  │              │   │
│  │ 1 writer     │  │ 1 writer     │  │ 1 writer     │   │
│  │ 5 readers    │  │ 5 readers    │  │ 5 readers    │   │
│  │              │  │              │  │              │   │
│  │ main.db      │  │ logs.db      │  │ analytics.db │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└───────────────────────┬──────────────────────────────────┘
                        ▼
┌──────────────────────────────────────────────────────────┐
│              zombiezen.com/go/sqlite                      │
│                   (modernc pure Go)                       │
└──────────────────────────────────────────────────────────┘
```

---

## 🔐 Auth & Policies (Production-Grade)

### Auth Chain (Fluide)

```go
// internal/auth/chain.go

type AuthChain struct {
    authenticators []Authenticator
    fallback       *Identity  // Anonymous par défaut
}

// Authenticators supportés (v1.0)
type Authenticator interface {
    Authenticate(r *http.Request) (*Identity, error)
}

// Implémentations :
// 1. JWTAuthenticator   (HS256, RS256, JWKS auto-refresh)
// 2. APIKeyAuthenticator (header X-API-Key ou query ?apikey=)
// 3. BasicAuthenticator (Authorization: Basic)

// Workflow
func (c *AuthChain) Authenticate(r *http.Request) (*Identity, error) {
    for _, auth := range c.authenticators {
        identity, err := auth.Authenticate(r)
        if err != nil {
            return nil, err  // Token invalide → 401
        }
        if identity != nil {
            return identity, nil  // Authentifié
        }
        // Pas applicable, essayer le suivant
    }
    return c.fallback, nil  // Anonymous
}
```

**Comportement** :
```
Pas de header        → Identity{id: "anon", role: "anon"}
JWT invalide         → 401 Unauthorized
JWT valide           → Identity{id: "user-123", role: "authenticated"}
API Key valide       → Identity{id: "api-xyz", role: "service"}
```

### Policies (RLS-like)

**Format TOML** :
```toml
# policies.toml

# Tout le monde peut lire les posts publiés
[[policy]]
name = "posts_public_read"
table = "posts"
operations = ["SELECT"]
roles = ["*"]
using = "status = 'published'"

# Auteurs peuvent modifier leurs propres posts
[[policy]]
name = "posts_owner_write"
table = "posts"
operations = ["UPDATE", "DELETE"]
roles = ["authenticated"]
using = "author_id = current_user_id()"

# Auteurs peuvent créer des posts (check à l'INSERT)
[[policy]]
name = "posts_create"
table = "posts"
operations = ["INSERT"]
roles = ["authenticated"]
check = "author_id = current_user_id()"

# Admins ont accès total
[[policy]]
name = "admin_full_access"
table = "*"
operations = ["SELECT", "INSERT", "UPDATE", "DELETE"]
roles = ["admin"]
using = "true"

# Masquer colonnes sensibles pour anonymes
[[policy]]
name = "users_anon_privacy"
table = "users"
operations = ["SELECT"]
roles = ["anon"]
columns.hidden = ["email", "password_hash"]
columns.masked = { phone = "'***-***-****'" }
```

**Mécanisme** :
```go
// internal/policy/engine.go

type PolicyEngine struct {
    policies []Policy
}

// Applicable retourne les policies qui s'appliquent
func (e *PolicyEngine) Applicable(
    identity *Identity,
    table string,
    op Operation,
) []Policy {
    var result []Policy
    for _, p := range e.policies {
        if !e.matches(p, identity, table, op) {
            continue
        }
        result = append(result, p)
    }
    return result
}

// BuildWhereClause combine les policies en OR
func (e *PolicyEngine) BuildWhereClause(policies []Policy) string {
    if len(policies) == 0 {
        return "false"  // Aucune policy = accès refusé
    }

    var conditions []string
    for _, p := range policies {
        if p.Using != "" {
            conditions = append(conditions, "("+p.Using+")")
        }
    }

    if len(conditions) == 0 {
        return "false"
    }

    // Policies combinées en OR (l'une suffit)
    return strings.Join(conditions, " OR ")
}
```

**Injection SQL** :
```sql
-- Requête originale
SELECT * FROM posts WHERE author = 'john'

-- Avec policies (role: anon)
SELECT * FROM posts
WHERE (status = 'published')  -- Policy appliquée
  AND (author = 'john')       -- Filtres utilisateur

-- Avec policies (role: authenticated, user-123)
SELECT * FROM posts
WHERE ((status = 'published') OR (author_id = 'user-123'))
  AND (author = 'john')
```

### UDF Context-Aware

```go
// internal/udf/builtin.go

func RegisterBuiltins(registry *UDFRegistry) {
    // Auth context
    registry.Register("current_user_id", func(ctx context.Context) string {
        return IdentityFrom(ctx).ID
    })

    registry.Register("current_role", func(ctx context.Context) string {
        return IdentityFrom(ctx).Role
    })

    registry.Register("current_claim", func(ctx context.Context, claim string) any {
        return IdentityFrom(ctx).Claims[claim]
    })

    registry.Register("has_role", func(ctx context.Context, role string) bool {
        return IdentityFrom(ctx).Role == role
    })

    // Utilitaires
    registry.Register("uuid_v4", func(ctx context.Context) string {
        return uuid.New().String()
    })

    registry.Register("sha256", func(ctx context.Context, data string) string {
        hash := sha256.Sum256([]byte(data))
        return hex.EncodeToString(hash[:])
    })

    registry.Register("now_unix", func(ctx context.Context) int64 {
        return time.Now().Unix()
    })
}
```

---

## 🛠️ Endpoints REST

### Routes Générées Automatiquement

**Mode Single-DB** :
```
GET    /tables                    # Liste des tables
GET    /{table}                   # SELECT avec filtres
POST   /{table}                   # INSERT
PATCH  /{table}                   # UPDATE (bulk ou WHERE)
DELETE /{table}                   # DELETE (bulk ou WHERE)

GET    /_schema                   # Schéma complet
GET    /_schema/{table}           # Schéma d'une table
POST   /rpc/{function}            # Appel UDF

GET    /_debug/schema             # Debug LLM : schéma complet
GET    /_debug/policies           # Debug LLM : policies
POST   /_debug/policies/eval      # Debug LLM : tester policies
GET    /_debug/pools              # Debug LLM : état pools
POST   /_debug/auth               # Debug LLM : tester auth
POST   /_debug/query              # Debug LLM : EXPLAIN
GET    /_debug/context            # Debug LLM : context actuel
GET    /_debug/health/verbose     # Debug LLM : healthcheck

GET    /health                    # Healthcheck simple
GET    /openapi.json              # Spec OpenAPI 3.0
```

**Mode Multi-DB** :
```
GET    /_databases                # Liste des DBs disponibles
GET    /{db}/tables               # Tables d'une DB
GET    /{db}/{table}              # SELECT avec filtres
POST   /{db}/{table}              # INSERT
PATCH  /{db}/{table}              # UPDATE
DELETE /{db}/{table}              # DELETE

GET    /{db}/_schema              # Schéma DB
POST   /{db}/rpc/{function}       # UDF dans contexte DB

GET    /_debug/*                  # Endpoints debug (cross-DB)
```

### Filtres PostgREST (Compatibilité)

```http
# Opérateurs de base
GET /posts?id=eq.123              # WHERE id = 123
GET /posts?views=gt.1000          # WHERE views > 1000
GET /posts?title=like.*SQLite*    # WHERE title LIKE '%SQLite%'
GET /posts?status=in.(draft,published)  # WHERE status IN ('draft', 'published')
GET /posts?deleted_at=is.null     # WHERE deleted_at IS NULL

# Logique combinée
GET /posts?or=(status.eq.published,author_id.eq.me)
# WHERE (status = 'published' OR author_id = current_user_id())

GET /posts?and=(views.gt.100,status.eq.published)
# WHERE (views > 100 AND status = 'published')

GET /posts?not.status=eq.draft
# WHERE NOT (status = 'draft')

# Tri et pagination
GET /posts?order=created_at.desc,views.desc&limit=10&offset=20
# ORDER BY created_at DESC, views DESC LIMIT 10 OFFSET 20

# Sélection colonnes
GET /posts?select=id,title,author(name,email)
# SELECT posts.id, posts.title, users.name, users.email
# FROM posts LEFT JOIN users ON posts.author_id = users.id

# Full-text search (FTS5)
GET /posts?content=fts.SQLite tutorial
# WHERE content MATCH 'SQLite tutorial'

# JSON operators
GET /posts?metadata->>tags=cs.["sqlite","database"]
# WHERE json_extract(metadata, '$.tags') @> '["sqlite","database"]'
```

### Headers HTTP

```http
# Pagination Range-based (PostgREST style)
GET /posts
Range: 0-24

# Response
HTTP/1.1 206 Partial Content
Content-Range: 0-24/150
[...]

# Préférences
POST /posts
Prefer: return=representation
# Retourne l'objet créé dans la réponse

PATCH /posts?id=eq.123
Prefer: return=minimal
# Retourne 204 No Content au lieu de l'objet

# Count exact
GET /posts
Prefer: count=exact
# Response header : Content-Range: 0-9/150
```

---

## 🧩 Structure du Projet

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
│   │   ├── manager.go                 # DBManager (registre pools)
│   │   ├── pool.go                    # ManagedPool (writer + readers)
│   │   ├── attach.go                  # Cross-DB queries
│   │   ├── introspect.go              # Lecture schéma SQLite
│   │   ├── schema.go                  # Types (Table, Column, FK)
│   │   └── pragmas.go                 # Configuration SQLite
│   │
│   ├── udf/
│   │   ├── registry.go                # Registre UDF
│   │   ├── builtin.go                 # UDF built-in (auth, utils)
│   │   └── types.go                   # Types UDF
│   │
│   ├── auth/
│   │   ├── identity.go                # Type Identity
│   │   ├── chain.go                   # AuthChain
│   │   ├── middleware.go              # HTTP middleware
│   │   ├── jwt.go                     # JWT authenticator
│   │   ├── apikey.go                  # API Key authenticator
│   │   └── basic.go                   # Basic auth
│   │
│   ├── policy/
│   │   ├── types.go                   # Types Policy
│   │   ├── engine.go                  # PolicyEngine
│   │   ├── parser.go                  # TOML → Policy
│   │   ├── middleware.go              # HTTP middleware
│   │   └── sql.go                     # WHERE injection
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
│       ├── Dockerfile                 # Image Alpine (15 MB)
│       └── docker-compose.yml         # Setup complet
│
├── docs/
│   ├── API.md                         # Documentation API REST
│   ├── FILTERS.md                     # Guide filtres PostgREST
│   ├── POLICIES.md                    # Guide policies RLS-like
│   ├── AUTH.md                        # Guide authentification
│   ├── UDF.md                         # Guide UDF
│   ├── DEPLOYMENT.md                  # Guide déploiement
│   └── COMPARISON.md                  # vs PostgREST, Datasette
│
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
│
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Tests + lint
│       ├── release.yml                # Binaires multi-plateformes
│       └── docker.yml                 # Image Docker
│
├── go.mod
├── go.sum
├── README.md
├── LICENSE                            # MIT ou Apache 2.0
├── CONTRIBUTING.md
├── CHANGELOG.md
└── Makefile
```

---

## 📋 Configuration

### Fichier principal (sqlitrest.toml)

```toml
# sqlitrest.toml

[server]
port = 8080
shutdown_timeout = "30s"

# Mode discovery automatique (optionnel)
[discovery]
enabled = false
path = "./data"
pattern = "*.db"

# Databases explicites
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

# Auth (fluide)
[auth]
enabled = true

[auth.jwt]
enabled = true
algorithm = "HS256"             # ou "RS256"
secret = "env:JWT_SECRET"       # HS256
# jwks_url = "https://..."      # RS256
audience = "sqlitrest"
issuer = "https://auth.example.com"
role_claim = "role"             # Claim pour extraire le rôle

[auth.apikey]
enabled = true
header = "X-API-Key"
# Stockage des clés :
# - file: ./apikeys.toml
# - db: main.api_keys (table)
source = "file:./apikeys.toml"

# Policies
[policies]
file = "./policies.toml"
default_action = "deny"         # ou "allow"

# Logging
[log]
level = "info"                  # debug, info, warn, error
format = "json"                 # json ou text

# Middleware
[middleware]
cors_origins = ["*"]
rate_limit_per_ip = 100         # req/s
request_timeout = "30s"

# Debug (production = false)
[debug]
enabled = true
require_admin = true            # Endpoints /_debug/* requièrent role admin
```

### API Keys (apikeys.toml)

```toml
# apikeys.toml

[[key]]
id = "service-analytics"
key_hash = "sha256:abcd1234..."  # SHA256 de la clé réelle
role = "service"
created_at = 2025-01-15T10:30:00Z

[[key]]
id = "frontend-app"
key_hash = "sha256:efgh5678..."
role = "authenticated"
created_at = 2025-01-20T14:00:00Z
```

---

## 🚀 Usage CLI

### Installation

```bash
# Depuis les releases GitHub
curl -L https://github.com/yourname/sqlitrest/releases/latest/download/sqlitrest-linux -o sqlitrest
chmod +x sqlitrest
sudo mv sqlitrest /usr/local/bin/

# Ou avec Go
go install github.com/yourname/sqlitrest/cmd/sqlitrest@latest

# Ou avec Docker
docker pull yourname/sqlitrest:latest
```

### Démarrage Simple

```bash
# Mode simple (1 commande)
sqlitrest my_app.db

# Avec port custom
sqlitrest my_app.db --port 3000

# Read-only
sqlitrest my_app.db --readonly

# Avec JWT auth
sqlitrest my_app.db --jwt-secret "$SECRET"
```

### Mode Avancé

```bash
# Multi-DB
sqlitrest \
  --db main=./main.db \
  --db logs=./logs.db:readonly \
  --db analytics=./analytics.db:readonly

# Avec config complète
sqlitrest --config ./sqlitrest.toml

# Discovery mode
sqlitrest --scan ./data/ --pattern "*.db"
```

---

## 🎯 Différenciation vs Concurrence

| Feature | PostgREST | Datasette | SQLitREST |
|---------|-----------|-----------|-----------|
| **Database** | PostgreSQL | SQLite | SQLite |
| **RLS** | ✅ Natif PG | ❌ | ✅ App-side |
| **Auth** | JWT | Plugin | ✅ JWT + API Key + Basic |
| **Policies** | ✅ Déclaratif | ❌ | ✅ TOML déclaratif |
| **Multi-DB** | ✅ Schemas | ⚠️ Limité | ✅ ATTACH natif |
| **Write** | ✅ | ❌ Read-only | ✅ CRUD complet |
| **OpenAPI** | ✅ | ⚠️ | ✅ Auto-gen |
| **Debug LLM** | ❌ | ⚠️ UI | ✅ `/_debug/*` |
| **Performance** | Excellent | Bon | Très bon |
| **Deployment** | Binaire | Python | Binaire |

**Positionnement** : "PostgREST pour SQLite, avec policies déclaratives et debug LLM"

---

## 📊 Métriques de Succès

### Performance Targets

```
Benchmark                   Target      Justification
──────────────────────────────────────────────────────
GET simple (no filter)      8k req/s    SQLite WAL read
GET filtered (3 conds)      5k req/s    WHERE clause overhead
GET embedded (1 FK JOIN)    2k req/s    JOIN overhead
POST insert                 3k req/s    Single writer limit
PATCH update                3k req/s    Single writer limit
DELETE                      3k req/s    Single writer limit

Latency (p95)
──────────────────────────────────────────────────────
GET simple                  <8ms        Direct SELECT
GET filtered                <12ms       WHERE parsing
GET embedded                <25ms       JOIN execution
Mutations                   <15ms       Write + RETURNING
```

### Adoption Targets (6 mois)

```
Métrique                    Target
────────────────────────────────────
GitHub stars                500+
Weekly downloads            1000+
Production deployments      50+
Contributors                10+
Documentation coverage      100%
Test coverage               80%+
```

---

## ✅ Checklist Implémentation

### Phase 1 : Core (Semaine 1-2)

- [ ] Setup projet Go (modules, structure)
- [ ] DBManager + ManagedPool (1 writer + N readers)
- [ ] Introspection schéma SQLite
- [ ] Query builder SELECT basique
- [ ] HTTP router Chi + handlers CRUD
- [ ] Tests unitaires core

### Phase 2 : Filtres & Pagination (Semaine 3)

- [ ] Parser filtres PostgREST (eq, gt, like, in, is)
- [ ] Opérateurs logiques (and, or, not)
- [ ] Pagination (limit/offset + Range header)
- [ ] Tri (order + nulls first/last)
- [ ] Tests filtres complets

### Phase 3 : Auth & Policies (Semaine 4)

- [ ] Auth chain (JWT + API Key + Basic)
- [ ] Identity + context propagation
- [ ] Policy engine (parser TOML)
- [ ] WHERE injection
- [ ] UDF context-aware
- [ ] Tests auth + policies

### Phase 4 : Features Avancées (Semaine 5)

- [ ] Embedding FK (LEFT JOIN)
- [ ] Sélection colonnes (?select=)
- [ ] UDF exposées (/rpc/*)
- [ ] Multi-DB (ATTACH/DETACH)
- [ ] Column-level masking
- [ ] Tests intégration

### Phase 5 : OpenAPI & Debug (Semaine 6)

- [ ] Génération OpenAPI 3.0
- [ ] Endpoints debug LLM (/_debug/*)
- [ ] Healthcheck verbose
- [ ] Métriques Prometheus
- [ ] Tests e2e

### Phase 6 : Polish & Release (Semaine 7-8)

- [ ] Documentation complète
- [ ] Exemples variés
- [ ] Docker image
- [ ] CI/CD GitHub Actions
- [ ] Release binaires multi-plateformes
- [ ] Website + landing page

---

## 🎯 Prêt à Implémenter

**Architecture validée** : ✅
- zombiezen + pools
- Auth fluide + policies
- REST uniquement (pas gRPC)
- Debug LLM natif

**Scope clair** : ✅
- Clone PostgREST pour SQLite
- Production-ready dès v1.0
- Open-source communautaire

**Timeline** : 8 semaines → v1.0 release

**Prochaine étape** : Démarrer Phase 1 (Core) ? 🚀
