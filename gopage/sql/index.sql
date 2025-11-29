-- Page d'accueil GoPage Forum
-- Démontre les composants de base

-- Shell (layout)
SELECT
    'shell' as component,
    'GoPage Forum' as title,
    'Propulsé par GoPage - SQL + Go + HTMX' as footer;

-- Hero section
SELECT
    'text' as component,
    'Bienvenue sur GoPage' as title,
    'Un forum entièrement propulsé par SQL. Chaque page est un fichier .sql qui définit les composants à afficher.' as contents;

-- Statistiques rapides
SELECT
    'cards' as component,
    'Forum en chiffres' as title;

SELECT
    'Topics' as title,
    '156' as description,
    'Discussions actives' as footer;

SELECT
    'Messages' as title,
    '2,847' as description,
    'Réponses postées' as footer;

SELECT
    'Membres' as title,
    '89' as description,
    'Utilisateurs inscrits' as footer;

SELECT
    'En ligne' as title,
    '12' as description,
    'Connectés maintenant' as footer;

-- Derniers topics
SELECT
    'text' as component,
    'Derniers Topics' as title;

SELECT
    'list' as component,
    'recent-topics' as id;

SELECT
    'Bienvenue sur GoPage Forum' as title,
    '/topic?id=1' as link,
    'admin - 42 réponses' as description;

SELECT
    'Comment utiliser les composants SQL' as title,
    '/topic?id=2' as link,
    'dev_john - 15 réponses' as description;

SELECT
    'HTMX et Alpine.js dans GoPage' as title,
    '/topic?id=3' as link,
    'frontend_marie - 28 réponses' as description;

-- Call to action
SELECT
    'card' as component,
    'Voir tous les topics' as title,
    'Parcourez l''ensemble des discussions du forum' as description,
    '/topics' as link;

-- Section fonctions SQL
SELECT
    'text' as component,
    'Fonctions SQL Custom' as title,
    'GoPage étend SQLite avec des fonctions puissantes.' as contents;

SELECT
    'cards' as component;

SELECT
    'Documentation' as title,
    'Toutes les fonctions disponibles' as description,
    '/functions' as link;

SELECT
    'Démo API' as title,
    'Test des fonctions HTTP' as description,
    '/api-demo' as link;

SELECT
    'Test LLM' as title,
    'Essayez les fonctions IA' as description,
    '/llm-test' as link;

-- Info version avec fonction custom
SELECT
    'text' as component,
    '<small>GoPage ' || gopage_version() || ' - Généré le ' || now_utc() || '</small>' as html;

-- Debug info (en mode développement)
SELECT
    'debug' as component,
    'Page: index' as page,
    now_utc() as rendered_at,
    gopage_version() as version;
