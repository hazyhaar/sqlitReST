# AGENTS.md - Guide pour Agents SQLitREST

## 🚀 Commandes Build/Lint/Test

### Build (Go standard)
```bash
go build ./...                    # Compile tous les packages
go build -o bin/sqlitrest ./cmd/sqlitrest   # Build binaire principal
```

### Lint
```bash
golangci-lint run                 # Lint standard
golangci-lint run --fast          # Lint rapide
```

### Test
```bash
go test -v ./...                  # Tous les tests
go test -run TestSpecific ./path   # Test spécifique
go test ./pkg/db -v               # Tests package spécifique
```

## 📋 Guidelines de Code Style SQLitREST

### Imports et Dépendances
- **Driver SQLite UNIQUEMENT** : `zombiezen.com/go/sqlite` (surcouche modernc, CGO-free)
- **Pattern zombiezen** : `sqlite.OpenConn()` + pragmas avancés
- Imports groupés : stdlib → externes → internes pkg/
- Framework HTTP : `github.com/go-chi/chi/v5`

### Architecture et Structure
- **Pattern SQL-First** : Pages définies en SQL, rendu HTML
- **Reader/Writer pools** : Connexions DB séparées pour concurrence
- **HTMX natif** : UIs dynamiques sans JavaScript complexe
- **Alpine.js** : Interactivité légère côté client

### Conventions de Nommage
- Packages : `{db,engine,auth,policies,config,debug}`
- Handlers : `{page}{Handler}` (ex: `usersHandler`)
- Templates : `{component}.html` dans `templates/`

### Gestion des Erreurs
- Utiliser `fmt.Errorf` avec wrapping : `fmt.Errorf("operation failed: %w", err)`
- Logs structurés avec préfixes : `log.Printf("sqlitrest: %v", err)`
- Vérifier erreurs fermeture avec `defer`

### SQLite et Pragmas (zombiezen)
```go
conn, err := sqlite.OpenConn(path, sqlite.OpenReadWrite|sqlite.OpenCreate)
if err != nil {
    return err
}
defer conn.Close()

// Pragmas obligatoires
err = conn.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;")
```

### Attach/Dtach Dynamique
```go
// Attach avec defer cleanup
if err := dbManager.AttachDB(name, path, mode); err != nil {
    return err
}
defer dbManager.DetachDB(name)  // Cleanup automatique
```

### Auth et Sécurité
- JWT avec `github.com/golang-jwt/jwt/v5`
- RLS-like policies implémentées côté application
- CORS avec `github.com/go-chi/cors`

---

*Ce document guide les agents IA dans le développement respectant l'architecture SQLitREST*