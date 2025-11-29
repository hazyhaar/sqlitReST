-- Liste des topics du forum
-- Utilise les paramètres de pagination automatiques: :_limit, :_offset, :_search_like

-- Shell (layout)
SELECT
    'shell' as component,
    'Forum - Topics' as title,
    'GoPage Forum Demo' as footer;

-- Titre de la page
SELECT
    'text' as component,
    'Tous les Topics' as title,
    'Parcourez les discussions du forum. Utilisez la recherche pour filtrer.' as contents;

-- Barre de recherche
SELECT
    'form' as component,
    'search-form' as id,
    'GET' as method,
    '' as action,
    'true' as hx_get,
    '#topics-table' as hx_target,
    'outerHTML' as hx_swap;

-- Champ de recherche (sera rendu par le template form)

-- Table des topics avec pagination HTMX
SELECT
    'table' as component,
    'topics-table' as id,
    'id:ID:number, title:Titre:text:sortable, author:Auteur:text, replies:Réponses:number, created_at:Créé le:text' as columns,
    '?page=' || (CAST(:_page as INTEGER) + 1) as next_page,
    CASE WHEN CAST(:_page as INTEGER) > 1
         THEN '?page=' || (CAST(:_page as INTEGER) - 1)
         ELSE NULL
    END as prev_page;

-- Données de la table (simulées - en prod ce serait une vraie requête)
SELECT
    1 as id,
    'Bienvenue sur GoPage Forum' as title,
    'admin' as author,
    42 as replies,
    '2024-01-15' as created_at,
    '/topic?id=1' as link;

SELECT
    2 as id,
    'Comment utiliser les composants SQL' as title,
    'dev_john' as author,
    15 as replies,
    '2024-01-16' as created_at,
    '/topic?id=2' as link;

SELECT
    3 as id,
    'HTMX et Alpine.js dans GoPage' as title,
    'frontend_marie' as author,
    28 as replies,
    '2024-01-17' as created_at,
    '/topic?id=3' as link;

SELECT
    4 as id,
    'Optimisation des requêtes SQLite' as title,
    'dba_paul' as author,
    7 as replies,
    '2024-01-18' as created_at,
    '/topic?id=4' as link;

SELECT
    5 as id,
    'Déploiement en production' as title,
    'devops_lisa' as author,
    33 as replies,
    '2024-01-19' as created_at,
    '/topic?id=5' as link;
