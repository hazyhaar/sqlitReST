# SQLitREST - Pattern Auth Tierce par Découverte

**Principe** : Mode mono-user par défaut, auth activée si module tiers détecté.

---

## 🎯 Concept

```
Démarrage sqlitrest
        │
        ▼
Chercher fichier auth conventionnel
        │
    ┌───┴───┐
    │       │
Trouvé   Absent
    │       │
    ▼       ▼
Mode      Mode
Auth    Mono-user
 ON       (ouvert)
```

---

## 📐 Convention de Nommage

### Option 1 : DB sibling

```
/data/
├── app.db                  # Base données exposée
└── app.auth.db             # Base auth (convention : <nom>.auth.db)
```

**Détection** :
```go
// Au démarrage
dbPath := "app.db"
authPath := strings.TrimSuffix(dbPath, ".db") + ".auth.db"

if fileExists(authPath) {
    enableAuth(authPath)
} else {
    log.Info("Auth disabled: no auth DB found")
}
```

### Option 2 : DB dans répertoire `.auth/`

```
/data/
├── app.db
└── .auth/
    ├── users.db            # Utilisateurs
    ├── tokens.db           # Sessions/JWT
    └── policies.db         # Règles RLS-like
```

**Détection** :
```go
authDir := filepath.Join(filepath.Dir(dbPath), ".auth")

if dirExists(authDir) && fileExists(filepath.Join(authDir, "users.db")) {
    enableAuth(authDir)
}
```

### Option 3 : Table dédiée dans la DB principale

```sql
-- Dans app.db
CREATE TABLE IF NOT EXISTS _sqlitrest_auth (
    user_id      TEXT PRIMARY KEY,
    api_key_hash TEXT NOT NULL,
    role         TEXT DEFAULT 'reader',
    created_at   INTEGER DEFAULT (unixepoch())
);
```

**Détection** :
```go
// Vérifier présence table _sqlitrest_auth
conn := pool.Get()
defer pool.Put(conn)

var exists bool
sqlitex.Execute(conn, `
    SELECT 1 FROM sqlite_master
    WHERE type='table' AND name='_sqlitrest_auth'
`, &sqlitex.ExecOptions{
    ResultFunc: func(stmt *sqlite.Stmt) error {
        exists = true
        return nil
    },
})

if exists {
    enableAuth()
}
```

---

## 🔒 Schéma Auth DB

### Structure minimale

```sql
-- app.auth.db

-- Table des utilisateurs
CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    username   TEXT UNIQUE NOT NULL,
    api_key    TEXT UNIQUE NOT NULL,  -- SHA256(secret)
    role       TEXT DEFAULT 'reader', -- reader, writer, admin
    created_at INTEGER DEFAULT (unixepoch()),
    expires_at INTEGER                -- NULL = jamais
);

CREATE INDEX idx_users_api_key ON users(api_key);

-- Table des permissions (optionnel, pour RLS-like)
CREATE TABLE policies (
    id         INTEGER PRIMARY KEY,
    role       TEXT NOT NULL,
    table_name TEXT NOT NULL,
    operation  TEXT NOT NULL,         -- SELECT, INSERT, UPDATE, DELETE
    condition  TEXT,                  -- SQL WHERE clause (ex: "user_id = :current_user")
    UNIQUE(role, table_name, operation)
);

-- Exemple de policy
INSERT INTO policies (role, table_name, operation, condition) VALUES
('reader', '*', 'SELECT', NULL),                -- Lecture partout
('writer', 'posts', 'INSERT', 'author_id = :user_id'), -- Écriture si auteur
('admin', '*', '*', NULL);                      -- Tout
```

---

## 🔑 Mécanisme d'Authentification

### Workflow

```
1. Client envoie requête avec header
   GET /users
   X-API-Key: sk_1234567890abcdef

2. Middleware auth vérifie
   - Auth DB existe ? → OUI
   - API key valide ? → OUI (lookup dans users table)
   - API key expirée ? → NON
   - Role extrait : "writer"

3. Middleware policies vérifie
   - Operation : SELECT sur table users
   - Policy pour role "writer" sur users : condition = "id = :current_user"
   - Injection WHERE : SELECT * FROM users WHERE id = 'user-123'

4. Query builder modifie requête
   - Requête originale : GET /users?age=gt.18
   - Requête finale : SELECT * FROM users WHERE age > 18 AND id = 'user-123'

5. Exécution + réponse
```

### Code Go

```go
// internal/auth/middleware.go
package auth

type Authenticator struct {
    authDB *sql.DB  // Connexion à app.auth.db
    mode   string   // "none", "basic", "jwt"
}

func NewAuthenticator(mainDBPath string) (*Authenticator, error) {
    authPath := strings.TrimSuffix(mainDBPath, ".db") + ".auth.db"

    if !fileExists(authPath) {
        return &Authenticator{mode: "none"}, nil
    }

    authDB, err := sql.Open("sqlite", authPath+"?mode=ro")
    if err != nil {
        return nil, err
    }

    log.Info("Auth enabled", "path", authPath)
    return &Authenticator{authDB: authDB, mode: "basic"}, nil
}

func (a *Authenticator) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if a.mode == "none" {
            // Pas d'auth, continuer
            next.ServeHTTP(w, r)
            return
        }

        // Extraire API key
        apiKey := r.Header.Get("X-API-Key")
        if apiKey == "" {
            http.Error(w, "Missing X-API-Key header", 401)
            return
        }

        // Vérifier dans auth DB
        user, err := a.validateAPIKey(apiKey)
        if err != nil {
            http.Error(w, "Invalid API key", 401)
            return
        }

        // Vérifier expiration
        if user.ExpiresAt != nil && time.Now().Unix() > *user.ExpiresAt {
            http.Error(w, "API key expired", 401)
            return
        }

        // Stocker user dans context
        ctx := context.WithValue(r.Context(), "current_user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (a *Authenticator) validateAPIKey(apiKey string) (*User, error) {
    var user User
    err := a.authDB.QueryRow(`
        SELECT id, username, role, expires_at
        FROM users
        WHERE api_key = ?
    `, hashAPIKey(apiKey)).Scan(&user.ID, &user.Username, &user.Role, &user.ExpiresAt)

    return &user, err
}

// Injecter policies dans requête
func (a *Authenticator) ApplyPolicies(query *query.SelectQuery, user *User) error {
    if user.Role == "admin" {
        return nil // Admin bypass policies
    }

    // Récupérer policies pour ce role + table + operation
    var condition string
    err := a.authDB.QueryRow(`
        SELECT condition
        FROM policies
        WHERE role = ? AND (table_name = ? OR table_name = '*') AND operation = 'SELECT'
        LIMIT 1
    `, user.Role, query.Table).Scan(&condition)

    if err == sql.ErrNoRows {
        return errors.New("Access denied: no policy found")
    }

    if condition != "" {
        // Parser condition et remplacer :user_id par valeur réelle
        condition = strings.ReplaceAll(condition, ":user_id", fmt.Sprintf("'%s'", user.ID))

        // Ajouter comme filtre
        query.Filters = append(query.Filters, query.Filter{
            RawSQL: condition,
        })
    }

    return nil
}
```

---

## 🛠️ Module Tiers (Exemple)

### Script de setup auth

```bash
#!/bin/bash
# setup-auth.sh - Créer auth DB pour sqlitrest

DB_NAME="$1"
AUTH_DB="${DB_NAME%.db}.auth.db"

echo "Creating auth DB: $AUTH_DB"

sqlite3 "$AUTH_DB" <<EOF
CREATE TABLE users (
    id         TEXT PRIMARY KEY,
    username   TEXT UNIQUE NOT NULL,
    api_key    TEXT UNIQUE NOT NULL,
    role       TEXT DEFAULT 'reader',
    created_at INTEGER DEFAULT (unixepoch()),
    expires_at INTEGER
);

CREATE TABLE policies (
    id         INTEGER PRIMARY KEY,
    role       TEXT NOT NULL,
    table_name TEXT NOT NULL,
    operation  TEXT NOT NULL,
    condition  TEXT,
    UNIQUE(role, table_name, operation)
);

-- Admin par défaut
INSERT INTO users (id, username, api_key, role) VALUES
('admin', 'admin', '$(echo -n 'secret-key-123' | sha256sum | cut -d' ' -f1)', 'admin');

-- Policies par défaut
INSERT INTO policies (role, table_name, operation) VALUES
('admin', '*', '*'),
('reader', '*', 'SELECT'),
('writer', '*', 'SELECT'),
('writer', '*', 'INSERT'),
('writer', '*', 'UPDATE');

EOF

echo "Auth DB created with admin user"
echo "API Key: secret-key-123"
echo ""
echo "To use:"
echo "  curl -H 'X-API-Key: secret-key-123' http://localhost:8080/users"
```

### Usage

```bash
# Créer auth DB
./setup-auth.sh app.db

# Lancer sqlitrest (détection auto)
sqlitrest app.db --port 8080
# [INFO] Auth enabled: path=app.auth.db

# Tester sans auth → 401
curl http://localhost:8080/users
# Missing X-API-Key header

# Tester avec auth → 200
curl -H "X-API-Key: secret-key-123" http://localhost:8080/users
# [{"id": "admin", "username": "admin", ...}]
```

---

## 🔧 Outil de Gestion Auth (CLI)

```bash
# Ajouter un utilisateur
sqlitrest auth add-user \
  --db app.db \
  --username john \
  --role writer \
  --expires 2025-12-31

# Généré:
# User: john
# API Key: sk_a3f9d8e7c6b5a4f3e2d1c0b9a8f7e6d5
# Role: writer
# Expires: 2025-12-31

# Lister utilisateurs
sqlitrest auth list-users --db app.db

# Révoquer
sqlitrest auth revoke --db app.db --username john

# Ajouter policy
sqlitrest auth add-policy \
  --db app.db \
  --role writer \
  --table posts \
  --operation UPDATE \
  --condition "author_id = :user_id"
```

---

## 📊 Avantages du Pattern

| Aspect | Bénéfice |
|--------|----------|
| **Zéro config** | Pas d'auth par défaut = démarrage immédiat |
| **Découverte auto** | Détection convention de nommage |
| **Séparation concerns** | Auth DB isolée de data DB |
| **Opt-in progressif** | Ajouter auth quand nécessaire |
| **Backup indépendant** | Auth DB sauvegardable séparément |
| **Multi-instance** | Plusieurs apps peuvent partager auth DB |

---

## 🎨 Variantes

### Variante 1 : Auth via variable d'environnement

```bash
# Pas d'auth DB, mais API key statique
export SQLITREST_API_KEY="my-secret-key"
sqlitrest app.db

# Requiert header X-API-Key: my-secret-key
```

### Variante 2 : Auth DB distante (SQLite over HTTP)

```toml
# sqlitrest.toml
[auth]
type = "remote"
url = "https://auth-service.example.com/validate"

# Chaque requête envoie API key à service distant pour validation
```

### Variante 3 : Auth via JWT décodé localement

```bash
# Auth DB contient clés publiques RSA
sqlitrest app.db

# Client envoie JWT
curl -H "Authorization: Bearer eyJhbGc..." http://localhost:8080/users

# sqlitrest vérifie signature avec clé publique dans auth DB
```

---

## 🚦 Recommandation

**Pour Phase 1 MVP** :

1. **Pas d'auth du tout** (mono-user local)
2. **Documenter** la convention `<nom>.auth.db` pour Phase 3
3. **Implémenter détection** en Phase 3 (5 lignes de code au démarrage)

**Pour Phase 3** :

1. Implémenter Option 1 (DB sibling)
2. Fournir script `setup-auth.sh`
3. Ajouter commande CLI `sqlitrest auth`

**Résultat** :
- MVP reste ultra-simple
- Upgrade vers auth = copier un fichier `.auth.db`
- Pas de refactoring du code principal

C'est exactement l'approche "convention over configuration" qu'on cherche ! 🎯
