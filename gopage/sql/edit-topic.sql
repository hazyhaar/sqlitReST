-- Formulaire d'édition d'un topic existant
-- Paramètre requis: :id

-- Shell (layout)
SELECT
    'shell' as component,
    'Modifier Topic #' || :id as title,
    'GoPage Forum Demo' as footer;

-- Breadcrumb
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/topics">Topics</a> &gt; <a href="/topic?id=' || :id || '">Topic #' || :id || '</a> &gt; Modifier</nav>' as html;

-- Formulaire d'édition avec valeurs pré-remplies
SELECT
    'form' as component,
    'edit-topic-form' as id,
    'POST' as method,
    '/edit-topic?id=' || :id as action,
    '/edit-topic?id=' || :id as hx_post,
    'Enregistrer les modifications' as submit_label,
    '/topic?id=' || :id as cancel_url,
    'Modifier le Topic' as title;

-- Champ caché pour l'ID
SELECT
    'id' as name,
    'hidden' as type,
    :id as value;

-- Champs pré-remplis avec les données existantes (simulées)
SELECT
    'title' as name,
    'text' as type,
    'Titre du topic' as label,
    CASE :id
        WHEN '1' THEN 'Bienvenue sur GoPage Forum'
        WHEN '2' THEN 'Comment utiliser les composants SQL'
        WHEN '3' THEN 'HTMX et Alpine.js dans GoPage'
        ELSE 'Topic #' || :id
    END as value,
    1 as required;

SELECT
    'category' as name,
    'select' as type,
    'Catégorie' as label,
    'general:Général,tech:Technique,help:Aide,off-topic:Hors-sujet' as options,
    'tech' as value;

SELECT
    'content' as name,
    'textarea' as type,
    'Contenu' as label,
    CASE :id
        WHEN '1' THEN 'Bienvenue sur notre nouveau forum propulsé par GoPage !'
        WHEN '2' THEN 'GoPage utilise un système de composants déclarés en SQL.'
        WHEN '3' THEN 'GoPage intègre HTMX pour les mises à jour partielles.'
        ELSE 'Contenu du topic...'
    END as value,
    1 as required;

-- Section de suppression (avec confirmation HTMX)
SELECT
    'text' as component,
    'Zone dangereuse' as title;

SELECT
    'card' as component,
    'Supprimer ce topic' as title,
    'Cette action est irréversible. Toutes les réponses seront également supprimées.' as description,
    '<button class="secondary"
             hx-delete="/delete-topic?id=' || :id || '"
             hx-confirm="Êtes-vous sûr de vouloir supprimer ce topic ?"
             hx-target="body">
        Supprimer définitivement
    </button>' as footer;
