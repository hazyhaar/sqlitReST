# sqlitREST

Serveur HTTP léger qui génère des pages web dynamiques à partir de fichiers SQL, utilisant SQLite avec le driver Go pur zombiezen.

## Caractéristiques

- **SQL-first** : Définissez vos pages et composants directement en SQL
- **SQLite pure Go** : Utilise zombiezen.com/go/sqlite (pas de CGO)
- **HTMX + Alpine.js** : Interactivité moderne sans JavaScript complexe
- **Pico.css** : Design responsive minimaliste
- **Fonctions SQL étendues** : UDF pour HTTP, LLM, SSE, crypto, dates...
- **SSE** : Server-Sent Events pour mises à jour temps réel

## Installation

```bash
git clone https://github.com/hazyhaar/sqlitReST.git
cd sqlitReST
go mod tidy
go build -o sqlitrest ./cmd/sqlitrest/
```

## Utilisation

```bash
./sqlitrest --help

# Options:
#   -port        Port HTTP (défaut: 8080)
#   -db          Chemin base SQLite (défaut: ./data.db)
#   -sql         Dossier fichiers SQL (défaut: ./sql)
#   -templates   Dossier templates (défaut: ./templates)
#   -debug       Mode debug
#   -llm-api-key Clé API pour fonctions LLM
```

### Démarrage rapide

```bash
./sqlitrest --debug
# Ouvrir http://localhost:8080/
```

## Structure du projet

```
sqlitREST/
├── cmd/sqlitrest/      # Point d'entrée
├── pkg/
│   ├── engine/         # Parser SQL + Executor
│   ├── funcs/          # Fonctions SQL personnalisées
│   ├── render/         # Rendu composants HTML
│   ├── server/         # Serveur HTTP (chi)
│   └── sse/            # Hub Server-Sent Events
├── sql/                # Pages SQL de démo
└── templates/          # Templates HTML
```

## Exemple de page SQL

```sql
-- sql/index.sql
SELECT 'shell' as component,
    'Ma Page' as title;

SELECT 'text' as component,
    'Bienvenue' as title,
    'Contenu de la page...' as contents;

SELECT 'table' as component,
    'Utilisateurs' as title;
SELECT id, name, email FROM users LIMIT 10;
```

## Fonctions SQL disponibles

### Utilitaires
- `gopage_version()` - Version de sqlitREST
- `uuid()` - Génère UUID v4
- `now_utc()` - Timestamp UTC actuel

### Crypto
- `sha256(text)` - Hash SHA256
- `md5(text)` - Hash MD5
- `base64_encode(text)` / `base64_decode(text)`

### Texte
- `slugify(text)` - Convertit en slug URL
- `truncate(text, length)` - Tronque avec ...
- `markdown_to_html(md)` - Convertit markdown

### HTTP
- `http_get(url)` - Requête GET
- `http_post(url, body, content_type)` - Requête POST
- `http_header(name, value)` - Définit header

### LLM (si configuré)
- `llm_ask(prompt)` - Question simple
- `llm_complete(prompt, system)` - Avec contexte système

### SSE (Server-Sent Events)
- `sse_notify(event, data)` - Envoie notification
- `sse_broadcast(message)` - Broadcast à tous
- `sse_client_count()` - Nombre de clients connectés

## Licence

MIT
