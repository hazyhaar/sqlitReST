# AGENTS.md - Guide pour Agents sqlitReST

## 🚀 Commandes Build/Lint/Test

### Build (Go standard)
```bash
cd gopage
go build ./...                    # Compile tous les packages
go build -o gopage ./cmd/gopage   # Build binaire principal
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
go test ./pkg/engine -v           # Tests package spécifique
```

## 📋 Guidelines de Code Style sqlitReST

### Imports et Dépendances
- **Driver SQLite UNIQUEMENT** : `zombiezen.com/go/sqlite` (sans CGO)
- Imports groupés : stdlib → externes → internes pkg/
- Framework HTTP : `github.com/go-chi/chi/v5`

### Architecture et Structure
- **Pattern SQL-First** : Pages définies en SQL, rendu HTML
- **Reader/Writer pools** : Connexions DB séparées pour concurrence
- **HTMX natif** : UIs dynamiques sans JavaScript complexe
- **Alpine.js** : Interactivité légère côté client

### Conventions de Nommage
- Packages : `{engine,funcs,render,server,sse,db}`
- Handlers : `{page}{Handler}` (ex: `usersHandler`)
- Templates : `{component}.html` dans `templates/`

### Gestion des Erreurs
- Utiliser `fmt.Errorf` avec wrapping : `fmt.Errorf("operation failed: %w", err)`
- Logs structurés avec préfixes : `log.Printf("sqlitrest: %v", err)`
- Vérifier erreurs fermeture avec `defer`

### SQLite et Pragmas
```go
db.Exec("PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA foreign_keys=ON;")
```

### Auth et Sécurité
- JWT avec `github.com/golang-jwt/jwt/v5`
- RLS-like policies implémentées côté application
- CORS avec `github.com/go-chi/cors`

---

*Ce document guide les agents IA dans le développement respectant l'architecture sqlitReST*