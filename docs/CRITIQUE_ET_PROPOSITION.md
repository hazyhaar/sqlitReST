# SQLitREST - Critique Autoclaude & Proposition Alternative

**Date** : 2025-11-29
**Auteur** : Claude (analyse critique)
**Context** : Réponse au plan développement autoclaude

---

## 📊 Analyse du Plan Autoclaude

### Forces ✅

1. **Analyse PostgREST complète**
   - Matrice de compatibilité PostgreSQL → SQLite rigoureuse
   - Identification claire des impossibilités (RLS natif, procédures stockées)
   - Compréhension fine des opérateurs de filtrage

2. **Pattern multi-DB + single-writer**
   - **Excellente** idée pour isolation physique vs RLS logique
   - Sécurité supérieure (fichiers isolés)
   - Scaling horizontal naturel
   - Backups triviaux (cp fichier)

3. **Stack technique cohérente**
   - zombiezen.com/go/sqlite (pool natif, WAL, CGO-free)
   - go-chi (router léger)
   - golang-jwt (auth standard)

### Faiblesses ⚠️

#### 1. Contradiction Scope

| Échange autoclaude | Instruction actuelle |
|--------------------|----------------------|
| "multiuser full auth" | "**monouser** centrée sur sqlite" |
| JWT RS256/HS256 + API Key + Basic | Auth nécessaire ? |
| Multi-tenant avec policies | Use case réel ? |

**Impact** : Risque de sur-ingénierie pour un besoin simple.

#### 2. Complexité fonctionnelle

**Inclus dans le plan** :
- ✅ REST API ← **Essentiel**
- ✅ gRPC ← **Utile pour quoi en mono-user ?**
- ✅ Policies RLS-like ← **Inutile si mono-user**
- ✅ Multi-DB routing ← **Nécessaire pour Horos ?**
- ✅ OpenAPI generation ← **Indispensable**
- ✅ UDF Go exposées ← **Très utile**
- ✅ Streaming gRPC ← **Over-kill ?**

**Estimation réaliste** :
- Plan autoclaude : "26 jours"
- Réalité pour scope complet : **8-12 semaines**
- MVP mono-user simple : **5-7 jours**

#### 3. Architecture complexe dès le départ

```
Plan autoclaude : 8 packages internes dès Phase 1
├── introspect/  ├── query/  ├── parser/  ├── api/
├── auth/        ├── db/      ├── aggregator/  └── router/
```

**Risque** : Complexité prématurée, time-to-market rallongé.

---

## 🎯 Proposition Alternative : Approche Progressive

### Philosophie : "Ship Fast, Iterate Smart"

```
Phase 0 : Clarifier le besoin réel (mono-user vs multi-user)
   ↓
Phase 1 : MVP mono-user (SQLite unique, local-first)
   ↓
Phase 2 : Support multi-DB (pattern Horos)
   ↓
Phase 3 : Auth basique (API key statique optionnelle)
   ↓
Phase 4 : Multi-user complet (si besoin validé)
```

---

## 🏗️ Architecture Phase 1 - MVP Mono-User

### Scope fonctionnel

| Fonctionnalité | Inclus | Raison |
|----------------|--------|--------|
| Introspection SQLite | ✅ | Core |
| CRUD automatique (GET/POST/PUT/DELETE) | ✅ | Core |
| Filtres PostgREST (?col=eq.X) | ✅ | Core |
| Pagination (limit/offset) | ✅ | Core |
| Tri (?order=col.desc) | ✅ | Core |
| Sélection colonnes (?select=a,b) | ✅ | Core |
| Embedding FK basique | ✅ | Core |
| OpenAPI auto-gen | ✅ | Documentation |
| UDF Go exposées (/rpc/*) | ✅ | Extensibilité |
| **Auth multi-méthode** | ❌ Phase 3 | YAGNI |
| **gRPC** | ❌ Phase 4 | YAGNI |
| **Policies** | ❌ Phase 4 | YAGNI |
| **Multi-DB routing** | ❌ Phase 2 | YAGNI |

**Livrable** : Un binaire qui expose N'IMPORTE QUELLE base SQLite en REST en 1 commande.

```bash
sqlitrest my_app.db --port 8080
# API complète disponible sur http://localhost:8080
```

### Structure simplifiée

```
sqlitrest/
├── cmd/
│   └── sqlitrest/
│       └── main.go              # CLI + serveur HTTP
├── internal/
│   ├── schema/
│   │   ├── introspector.go      # sqlite_master + PRAGMA
│   │   └── types.go             # Table, Column, ForeignKey structs
│   ├── query/
│   │   ├── builder.go           # SQL query construction
│   │   ├── filter.go            # ?col=op.value parser
│   │   └── embed.go             # FK joins
│   ├── api/
│   │   ├── router.go            # Chi router setup
│   │   ├── handlers.go          # HTTP handlers
│   │   └── openapi.go           # OpenAPI spec generation
│   ├── udf/
│   │   ├── registry.go          # UDF Go → SQL functions
│   │   └── rpc.go               # /rpc/* endpoints
│   └── db/
│       └── conn.go              # zombiezen pool wrapper
├── go.mod
├── README.md
└── examples/
    └── horos-integration.sh     # Exemple Horos
```

**Packages total** : 5 (vs 8 dans plan autoclaude)

### Stack technique

```go
// go.mod
module github.com/yourname/sqlitrest

go 1.23

require (
    zombiezen.com/go/sqlite v1.4.0        // SQLite driver
    github.com/go-chi/chi/v5 v5.1.0       // HTTP router
    github.com/go-chi/cors v1.2.1         // CORS middleware
    github.com/swaggo/swag v1.16.3        // OpenAPI generation
)
```

**Total dépendances** : 4 (minimaliste)

### Exemple d'API générée

```http
# Introspection
GET /tables                              # Liste tables
GET /tables/users                        # Schéma table users

# CRUD
GET    /users                            # SELECT * FROM users
GET    /users?id=eq.123                  # WHERE id = 123
GET    /users?age=gt.18&order=name.asc   # WHERE age > 18 ORDER BY name
POST   /users                            # INSERT
PUT    /users?id=eq.123                  # UPDATE WHERE id = 123
DELETE /users?id=eq.123                  # DELETE WHERE id = 123

# Embedding (si FK users.company_id → companies.id)
GET /users?select=*,company(name,city)   # JOIN companies

# UDF (si fonction Go enregistrée)
POST /rpc/send_email                     # Exécute fonction Go
```

### Opérateurs de filtrage

```
eq     : égal (=)
neq    : différent (<>)
gt     : supérieur (>)
gte    : supérieur ou égal (>=)
lt     : inférieur (<)
lte    : inférieur ou égal (<=)
like   : LIKE '%pattern%'
ilike  : LIKE avec COLLATE NOCASE
in     : IN (val1, val2)
is     : IS NULL / IS NOT NULL
fts    : Full-text search (FTS5)
```

---

## 🔄 Intégration Horos (Phase 2)

### Pattern multi-DB respectueux

```bash
# Mode simple (Phase 1)
sqlitrest horos_events.db

# Mode Horos (Phase 2)
sqlitrest \
  --db events=horos_events.db:readonly \
  --db meta=horos_meta.db:readonly \
  --db tickets=ops_tickets.db

# Routes générées :
# /events/tasks, /events/logs, /events/heartbeats
# /meta/registry, /meta/config
# /tickets/bugs
```

### Pattern single-writer (Phase 2)

```go
// db/conn.go
type DBManager struct {
    readers  *sqlitex.Pool    // N connexions read-only (WAL)
    writer   *sqlite.Conn     // UNE seule connexion write
    writeCh  chan WriteOp     // Queue sérialisée
}

func (m *DBManager) Write(op WriteOp) error {
    m.writeCh <- op           // Sérialisé automatiquement
    return <-op.resultCh
}

func (m *DBManager) Read() *sqlite.Conn {
    return m.readers.Get()    // Pool illimité (WAL)
}
```

**Avantages** :
- ✅ Pas de SQLITE_BUSY
- ✅ Writes ordonnés garantis
- ✅ Reads parallèles illimités

---

## ❓ Questions de Clarification

### 1. Scope fonctionnel

**Question** : Quel est le besoin RÉEL ?

- [ ] **Option A** : Outil **mono-user local** pour Horos (exposer horos_events.db en REST pour UI/MCP)
- [ ] **Option B** : Outil **générique** réutilisable par la communauté (comme Datasette)
- [ ] **Option C** : Plateforme **multi-user SaaS** (multi-tenant, auth complète)

**Impact** :
- Option A → MVP en **5 jours**
- Option B → MVP en **2 semaines**
- Option C → Projet complet en **8-12 semaines**

### 2. Authentification

**Question** : Quelle sécurité ?

- [ ] **Aucune** (localhost uniquement, Horos interne)
- [ ] **API key statique** (protection basique)
- [ ] **JWT complet** (multi-user, RBAC)

### 3. Protocoles

**Question** : Pourquoi gRPC ?

- Cas d'usage concret : _______________
- Si pas de réponse claire → **Phase 4 optionnelle**

### 4. Multi-DB

**Question** : Pattern Horos (4-BDD) obligatoire dès Phase 1 ?

- [ ] **Oui** : Exposer input/lifecycle/output/metadata dès le départ
- [ ] **Non** : Commencer avec SQLite unique, ajouter multi-DB en Phase 2

### 5. UDF Priority

**Question** : Exemples d'UDF Go nécessaires ?

- [ ] Fonctions crypto (SHA256, HMAC)
- [ ] Appels HTTP externes (webhooks)
- [ ] Génération de données (UUID, timestamps)
- [ ] Autre : _______________

### 6. Délai attendu

**Question** : Time-to-market ?

- [ ] **Démo fonctionnelle** : Dans combien de temps ? (jours/semaines)
- [ ] **Production-ready** : Deadline ? (date)

---

## 📅 Planning Révisé

### Phase 1 : MVP Mono-User (5-7 jours)

**Jour 1-2** : Introspection + Query Builder
- [x] Lecture sqlite_master
- [x] PRAGMA table_info / foreign_key_list
- [x] SELECT builder avec WHERE/ORDER/LIMIT

**Jour 3-4** : API REST
- [x] Routes CRUD automatiques
- [x] Filtres PostgREST (?col=op.value)
- [x] Embedding FK basique

**Jour 5** : UDF + OpenAPI
- [x] Registre UDF Go
- [x] Endpoints /rpc/*
- [x] Génération OpenAPI

**Jour 6-7** : Polish
- [x] README killer
- [x] Exemples d'utilisation
- [x] Tests unitaires de base

**Livrable** : Binaire fonctionnel démontrable.

### Phase 2 : Multi-DB + Horos (1 semaine)

- [x] Support `--db name=path:mode`
- [x] Pattern single-writer
- [x] Namespace par DB (/dbname/table)

### Phase 3 : Auth Basique (3-4 jours)

- [x] API key statique (header `X-API-Key`)
- [x] Mode read-only optionnel

### Phase 4 : Multi-User Complet (SI BESOIN)

- [x] JWT RS256/HS256
- [x] Policies row-level
- [x] gRPC
- [x] Streaming

---

## 🎯 Recommandation Finale

**Je propose** :

1. **Commencer par Phase 1 MVP** (mono-user, SQLite unique)
2. **Valider l'utilité** avec un vrai cas d'usage Horos
3. **Itérer** vers Phase 2-3 selon besoin réel
4. **Ne pas coder Phase 4** sans demande explicite

**Pourquoi** :
- Time-to-market rapide (1 semaine vs 3 mois)
- Validation du concept avant sur-investissement
- Respect du principe YAGNI (You Aren't Gonna Need It)
- Architecture évolutive (pas de refacto si ajout features)

**Langage recommandé** : **Go**
- Cohérence Horos
- zombiezen/modernc natifs
- Simplicité vs Rust
- Communauté large

---

## 🚦 Prochaine Étape

**Avant tout code** :

1. **Répondre aux 6 questions** ci-dessus
2. **Valider le scope** (Phase 1 MVP vs Full)
3. **Confirmer le langage** (Go recommandé)
4. **Donner le feu vert** pour implémentation

Je suis prêt à démarrer dès validation. 🚀
