# sqlitReST - Version Récupérée Complète

Version exhaustive du projet **sqlitReST** rassemblée depuis la corbeille le 2025-11-29.

## Vue d'ensemble

**sqlitReST** est composé de :

1. **GoPage** : Serveur web SQL-driven en Go
2. **Documentation Architecture** : Spécifications techniques complètes

## Concept

GoPage est un serveur d'application web piloté par SQL, inspiré de SQLPage avec :
- **SQLite pur Go** (zombiezen.com/go/sqlite, sans CGO)
- **HTMX natif** : UIs dynamiques sans JavaScript
- **Alpine.js** : Interactivité légère côté client
- **Pattern Reader/Writer** : Pools de connexions pour concurrence
- **SQL-First** : Définir des pages web directement en SQL

## Structure

```
sqlitReST/
├── docs/                        # Documentation architecture
│   ├── ARCHITECTURE_FINALE.md
│   ├── SQLITREST_BLUEPRINT_COMPLET.md
│   └── ...
├── gopage/                      # Code source
│   ├── cmd/gopage/
│   ├── pkg/{db,engine,funcs,render,server,sse}/
│   └── sql/                     # Pages SQL exemples
└── LICENSE
```

## Quick Start

```bash
cd gopage
go mod tidy
go build -o gopage ./cmd/gopage
./gopage -db myapp.db -sql ./sql -port 8080
```

Voir `gopage/README.md` pour détails complets.

## Documentation

- **ARCHITECTURE_FINALE.md** : Décisions validées
- **SQLITREST_BLUEPRINT_COMPLET.md** : Blueprint technique
- **AUTH_PATTERN.md** : Authentification/autorisation

## Origine

Récupéré depuis la corbeille (3 versions fusionnées) le 2025-11-29.

## License

MIT
