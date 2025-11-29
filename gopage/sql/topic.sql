-- Vue d'un topic individuel
-- Paramètre requis: :id

-- Shell (layout)
SELECT
    'shell' as component,
    'Forum - Topic #' || :id as title,
    'GoPage Forum Demo' as footer;

-- Breadcrumb
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/topics">Topics</a> &gt; Topic #' || :id || '</nav>' as html;

-- Détails du topic (simulé)
SELECT
    'card' as component,
    CASE :id
        WHEN '1' THEN 'Bienvenue sur GoPage Forum'
        WHEN '2' THEN 'Comment utiliser les composants SQL'
        WHEN '3' THEN 'HTMX et Alpine.js dans GoPage'
        WHEN '4' THEN 'Optimisation des requêtes SQLite'
        ELSE 'Topic #' || :id
    END as title,
    CASE :id
        WHEN '1' THEN 'admin'
        WHEN '2' THEN 'dev_john'
        WHEN '3' THEN 'frontend_marie'
        WHEN '4' THEN 'dba_paul'
        ELSE 'anonymous'
    END || ' - ' ||
    CASE :id
        WHEN '1' THEN '15 janvier 2024'
        WHEN '2' THEN '16 janvier 2024'
        WHEN '3' THEN '17 janvier 2024'
        WHEN '4' THEN '18 janvier 2024'
        ELSE 'Date inconnue'
    END as subtitle,
    CASE :id
        WHEN '1' THEN 'Bienvenue sur notre nouveau forum propulsé par GoPage ! Ce forum est entièrement généré à partir de fichiers SQL.'
        WHEN '2' THEN 'GoPage utilise un système de composants déclarés en SQL. Chaque SELECT avec une colonne "component" définit un composant à afficher.'
        WHEN '3' THEN 'GoPage intègre HTMX pour les mises à jour partielles et Alpine.js pour la réactivité côté client.'
        WHEN '4' THEN 'Pour optimiser vos requêtes SQLite dans GoPage, utilisez les index appropriés et limitez les résultats avec LIMIT/OFFSET.'
        ELSE 'Contenu du topic...'
    END as description;

-- Section réponses
SELECT
    'text' as component,
    'Réponses' as title;

-- Liste des réponses (simulées)
SELECT
    'cards' as component,
    'topic-replies' as id;

SELECT
    'Première réponse' as title,
    'user1' as subtitle,
    'Super topic, merci pour le partage !' as description,
    'Il y a 2 heures' as footer;

SELECT
    'Deuxième réponse' as title,
    'user2' as subtitle,
    'J''ai une question : comment personnaliser les templates ?' as description,
    'Il y a 1 heure' as footer;

SELECT
    'Troisième réponse' as title,
    'admin' as subtitle,
    '@user2 Les templates sont dans le dossier templates/. Tu peux créer des composants personnalisés en ajoutant des fichiers .html.' as description,
    'Il y a 30 minutes' as footer;

-- Formulaire de réponse
SELECT
    'text' as component,
    'Ajouter une réponse' as title;

SELECT
    'form' as component,
    'reply-form' as id,
    'POST' as method,
    '/topic/reply' as action,
    '/topic/reply' as hx_post,
    '#topic-replies' as hx_target,
    'beforeend' as hx_swap,
    'Envoyer' as submit_label;

-- Champs du formulaire seront rendus par le template form.html
