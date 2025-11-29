-- Formulaire de création d'un nouveau topic
-- GET: Affiche le formulaire
-- POST: Crée le topic et redirige

-- Shell (layout)
SELECT
    'shell' as component,
    'Nouveau Topic' as title,
    'GoPage Forum Demo' as footer;

-- Breadcrumb
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/topics">Topics</a> &gt; Nouveau</nav>' as html;

-- Formulaire de création
SELECT
    'form' as component,
    'new-topic-form' as id,
    'POST' as method,
    '/new-topic' as action,
    '/new-topic' as hx_post,
    'Créer le topic' as submit_label,
    '/topics' as cancel_url,
    'Nouveau Topic' as title,
    'Partagez vos idées avec la communauté' as description;

-- Champs du formulaire
SELECT
    'title' as name,
    'text' as type,
    'Titre du topic' as label,
    'Un titre descriptif pour votre discussion' as placeholder,
    1 as required,
    'Le titre doit être clair et concis' as help;

SELECT
    'category' as name,
    'select' as type,
    'Catégorie' as label,
    'Choisissez une catégorie' as placeholder,
    'general:Général,tech:Technique,help:Aide,off-topic:Hors-sujet' as options,
    1 as required;

SELECT
    'content' as name,
    'textarea' as type,
    'Contenu' as label,
    'Décrivez votre sujet en détail...' as placeholder,
    1 as required,
    'Soyez précis et respectueux' as help;

SELECT
    'notify' as name,
    'checkbox' as type,
    'Me notifier des réponses' as label,
    '1' as value;
