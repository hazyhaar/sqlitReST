-- Page d'accueil GoPage

-- Shell (layout)
SELECT
    'shell' as component,
    'GoPage - Forum' as title,
    'Un forum propulsé par SQL et Go' as footer;

-- Texte de bienvenue
SELECT
    'text' as component,
    'Bienvenue sur GoPage' as title,
    'Ce forum est entièrement généré à partir de fichiers SQL.
     Chaque page correspond à un fichier .sql qui définit les composants à afficher.' as contents;

-- Statistiques (simulées)
SELECT
    'card' as component,
    'Statistiques' as title,
    'Utilisateurs: 42 | Messages: 1337 | Topics: 256' as description;

-- Debug info
SELECT
    'debug' as component,
    'Index page loaded' as message,
    datetime('now') as timestamp;
