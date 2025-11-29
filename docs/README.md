# SQLitREST - PostgREST pour SQLite

**Statut** : 🟡 En attente de validation avant implémentation

**Objectif** : API REST automatique pour SQLite, inspirée de PostgREST, adaptée au contexte mono-user et Horos.

---

## 📚 Documentation Produite

| Fichier | Contenu |
|---------|---------|
| **[CRITIQUE_ET_PROPOSITION.md](./CRITIQUE_ET_PROPOSITION.md)** | Analyse critique de l'approche autoclaude + proposition alternative progressive |
| **[ARCHITECTURE_TECHNIQUE.md](./ARCHITECTURE_TECHNIQUE.md)** | Architecture détaillée MVP Phase 1 avec exemples de code Go |
| **[AUTH_PATTERN.md](./AUTH_PATTERN.md)** | Pattern auth tierce par découverte automatique (`.auth.db`) |

---

## 🎯 Proposition Résumée

### Approche Progressive

```
Phase 1 (MVP)     : Mono-user, SQLite unique, CRUD + filtres PostgREST
  ↓ 5-7 jours
Phase 2           : Multi-DB support (pattern Horos 4-BDD)
  ↓ 1 semaine
Phase 3           : Auth par découverte (.auth.db)
  ↓ 3-4 jours
Phase 4 (optionnel): Multi-user complet + gRPC + policies
```

### Stack Technique (Phase 1)

```go
// 4 dépendances seulement
zombiezen.com/go/sqlite    // Driver SQLite (WAL, pool, CGO-free)
github.com/go-chi/chi/v5   // HTTP router
github.com/go-chi/cors     // CORS
github.com/swaggo/swag     // OpenAPI generation
```

### Fonctionnalités MVP

| Feature | Status |
|---------|--------|
| Introspection SQLite | ✅ Planifié |
| CRUD automatique | ✅ Planifié |
| Filtres PostgREST (?col=eq.X) | ✅ Planifié |
| Embedding FK | ✅ Planifié |
| Pagination | ✅ Planifié |
| UDF Go exposées | ✅ Planifié |
| OpenAPI auto-gen | ✅ Planifié |
| **Auth multi-user** | ⏸️ Phase 3 |
| **gRPC** | ⏸️ Phase 4 |
| **Policies RLS-like** | ⏸️ Phase 4 |

---

## ❓ Questions Avant Implémentation

### 1️⃣ Scope Fonctionnel

**Question** : Quel est le besoin PRIMAIRE ?

- [ ] **A** : Outil mono-user pour Horos (exposer `horos_events.db` en REST pour UI/MCP)
- [ ] **B** : Outil générique communautaire (comme Datasette, réutilisable partout)
- [ ] **C** : Plateforme multi-user SaaS (multi-tenant, auth complète, production-grade)

**Impact planning** :
- A → MVP en **5 jours**
- B → MVP en **10-14 jours** (doc + exemples variés)
- C → **8-12 semaines** (auth + policies + tests + sécurité)

---

### 2️⃣ Cas d'Usage Concret

**Question** : Quelle est la première utilisation RÉELLE prévue ?

Exemple :
- "Exposer `horos_events.db` en read-only pour qu'une UI web affiche les logs en temps réel"
- "Permettre au serveur MCP de lire `horos_meta.db.registry` via HTTP au lieu de SQLite direct"
- "Créer une API publique pour un projet perso basé sur SQLite"

**Pourquoi important** : Détermine les features prioritaires (read-only ? writes ? UDF custom ?)

---

### 3️⃣ Authentification

**Question** : Quelle sécurité pour Phase 1 MVP ?

- [ ] **Aucune** (localhost uniquement, réseau local de confiance)
- [ ] **API key statique** (variable env `SQLITREST_API_KEY`, protection basique)
- [ ] **Auth découverte** (`.auth.db` optionnelle, comme décrit dans AUTH_PATTERN.md)

**Ma recommandation** : Aucune en Phase 1, `.auth.db` en Phase 3.

---

### 4️⃣ Multi-DB dès le Départ ?

**Question** : Pattern Horos 4-BDD nécessaire en Phase 1 ?

- [ ] **Oui** : Exposer `input.db`, `lifecycle.db`, `output.db`, `metadata.db` dès MVP
- [ ] **Non** : Commencer avec SQLite unique, ajouter multi-DB en Phase 2

**Impact** :
- Oui → +3-4 jours de dev (routing, namespaces)
- Non → MVP plus rapide, ajout incrémental

---

### 5️⃣ gRPC Justification

**Question** : Pourquoi gRPC en plus de REST ?

Cas d'usage concrets : ______________________________

**Si pas de réponse claire** → Reporter en Phase 4 optionnelle.

**Note** : gRPC ajoute :
- Complexité (protobuf, code generation)
- Dépendances supplémentaires
- Temps de dev : +1-2 semaines

---

### 6️⃣ UDF Prioritaires

**Question** : Quelles UDF Go sont nécessaires en Phase 1 ?

Exemples possibles :
- [ ] Fonctions crypto (sha256, hmac, uuid)
- [ ] Appels HTTP externes (webhooks, fetch)
- [ ] Génération données (timestamps, slugs)
- [ ] Intégration Horos (lecture `horos_meta.db` depuis UDF)
- [ ] Autre : ______________________________

**Ma recommandation** : UDF basiques (sha256, uuid, now) suffisent pour MVP.

---

### 7️⃣ Driver SQLite

**Question** : Préférence entre drivers ?

| Driver | Avantages | Inconvénients |
|--------|-----------|---------------|
| **zombiezen** | Pool natif, API bas niveau, très rapide | API moins idiomatique que database/sql |
| **modernc** | database/sql standard, utilisé dans Horos | Pas de pool natif, API plus haut niveau |

**Mon analyse** :
- **zombiezen** si performance critique + contrôle fin
- **modernc** si cohérence Horos primordiale

**Recommandation** : **zombiezen** pour ce projet (performance API REST)

---

### 8️⃣ Timeline

**Question** : Délai souhaité ?

- [ ] Démo fonctionnelle : Dans **_____ jours/semaines**
- [ ] Production-ready : Date limite **_____**

**Planning réaliste** (si Phase 1 MVP validé) :
- Jour 1-2 : Introspection + Query Builder
- Jour 3-4 : HTTP handlers + filtres
- Jour 5 : UDF + OpenAPI
- Jour 6-7 : Tests + doc + polish

**Livrable J7** : Binaire déployable, README complet, exemples d'utilisation.

---

## 🚀 Prêt à Démarrer

**Ce qui est fait** :
- ✅ Étude PostgREST
- ✅ Critique approche autoclaude
- ✅ Architecture Phase 1 détaillée
- ✅ Pattern auth tierce
- ✅ Exemples de code Go

**Ce qu'il manque** :
- ⏳ Validation du scope (réponses aux 8 questions ci-dessus)
- ⏳ Feu vert pour implémentation

**Dès validation** :
1. Initialisation repo Git
2. Setup Go module
3. Implémentation Phase 1 (5-7 jours)
4. Livraison MVP fonctionnel

---

## 📖 Sources

- [PostgREST Documentation](https://postgrest.org/)
- [PostgREST GitHub](https://github.com/PostgREST/postgrest)
- [PostgREST API Reference](https://postgrest.org/en/stable/references/api/resource_embedding.html)
- [zombiezen SQLite Driver](https://pkg.go.dev/zombiezen.com/go/sqlite)
- [Go Chi Router](https://github.com/go-chi/chi)
- [Datasette](https://datasette.io/) (inspiration alternative)

---

## 💬 Contact

Questions, clarifications ou feu vert → Répondre aux 8 questions ci-dessus.

Prêt à coder dès validation. 🎯
