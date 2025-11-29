-- Page de démonstration des fonctions SQL custom
-- Montre toutes les fonctions UDF disponibles dans GoPage

SELECT
    'shell' as component,
    'Fonctions SQL Custom' as title,
    'GoPage - Documentation des fonctions' as footer;

-- Navigation
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; Fonctions SQL</nav>' as html;

-- Introduction
SELECT
    'text' as component,
    'Fonctions disponibles' as title,
    'GoPage étend SQLite avec des fonctions personnalisées pour les opérations courantes.' as contents;

-- Section: Fonctions utilitaires
SELECT
    'text' as component,
    'Fonctions Utilitaires' as title;

SELECT
    'table' as component,
    'Exemples de fonctions utilitaires' as title;

SELECT
    'gopage_version()' as fonction,
    gopage_version() as resultat,
    'Version de GoPage' as description;

SELECT
    'uuid()' as fonction,
    uuid() as resultat,
    'Génère un UUID v4' as description;

SELECT
    'now_utc()' as fonction,
    now_utc() as resultat,
    'Date/heure UTC actuelle' as description;

-- Section: Fonctions de hashage
SELECT
    'text' as component,
    'Fonctions de Hashage' as title;

SELECT
    'table' as component,
    'Exemples de hashage' as title;

SELECT
    'sha256(''hello'')' as fonction,
    sha256('hello') as resultat,
    'Hash SHA256' as description;

SELECT
    'md5(''hello'')' as fonction,
    md5('hello') as resultat,
    'Hash MD5' as description;

-- Section: Encodage
SELECT
    'text' as component,
    'Fonctions d''Encodage' as title;

SELECT
    'table' as component,
    'Exemples d''encodage' as title;

SELECT
    'base64_encode(''GoPage'')' as fonction,
    base64_encode('GoPage') as resultat,
    'Encode en Base64' as description;

SELECT
    'base64_decode(''R29QYWdl'')' as fonction,
    base64_decode('R29QYWdl') as resultat,
    'Décode du Base64' as description;

SELECT
    'url_encode(''hello world!'')' as fonction,
    url_encode('hello world!') as resultat,
    'Encode pour URL' as description;

SELECT
    'url_decode(''hello%20world%21'')' as fonction,
    url_decode('hello%20world%21') as resultat,
    'Décode une URL' as description;

-- Section: Manipulation de texte
SELECT
    'text' as component,
    'Manipulation de Texte' as title;

SELECT
    'table' as component,
    'Exemples de manipulation de texte' as title;

SELECT
    'slugify(''Bonjour le Monde!'')' as fonction,
    slugify('Bonjour le Monde!') as resultat,
    'Crée un slug URL-friendly' as description;

SELECT
    'truncate(''Lorem ipsum dolor sit amet...'', 20)' as fonction,
    truncate('Lorem ipsum dolor sit amet consectetur adipiscing elit', 20) as resultat,
    'Tronque avec ellipse' as description;

SELECT
    'strip_html(''<p>Hello <b>World</b></p>'')' as fonction,
    strip_html('<p>Hello <b>World</b></p>') as resultat,
    'Supprime les balises HTML' as description;

-- Section: Formatage
SELECT
    'text' as component,
    'Fonctions de Formatage' as title;

SELECT
    'table' as component,
    'Exemples de formatage' as title;

SELECT
    'format_number(1234567)' as fonction,
    format_number(1234567) as resultat,
    'Formate avec séparateurs' as description;

SELECT
    'format_bytes(1536000)' as fonction,
    format_bytes(1536000) as resultat,
    'Formate en Ko/Mo/Go' as description;

SELECT
    'format_date(''2024-01-15'', ''long'')' as fonction,
    format_date('2024-01-15', 'long') as resultat,
    'Formate une date' as description;

-- Section: JSON
SELECT
    'text' as component,
    'Fonctions JSON' as title;

SELECT
    'table' as component,
    'Exemples JSON' as title;

SELECT
    'json_extract_path(json, ''name'')' as fonction,
    json_extract_path('{"name":"GoPage","version":"0.1"}', 'name') as resultat,
    'Extrait une valeur JSON' as description;

-- Section: Markdown
SELECT
    'text' as component,
    'Conversion Markdown' as title;

SELECT
    'card' as component,
    'markdown_to_html()' as title,
    'Convertit du Markdown basique en HTML' as description;

SELECT
    'text' as component,
    markdown_to_html('# Titre

Ceci est un **paragraphe** avec du texte en *italique*.

[Lien vers GoPage](/functions)') as html;

-- Section: HTTP (si disponible)
SELECT
    'text' as component,
    'Fonctions HTTP' as title,
    'Ces fonctions permettent de faire des requêtes HTTP depuis SQL.' as contents;

SELECT
    'card' as component,
    'http_get(url)' as title,
    'Fait une requête GET et retourne le body' as description;

SELECT
    'card' as component,
    'http_post_json(url, json)' as title,
    'Fait une requête POST avec un body JSON' as description;

SELECT
    'card' as component,
    'webhook(url, payload)' as title,
    'Envoie un webhook (fire and forget)' as description;

-- Section: LLM (si configuré)
SELECT
    'text' as component,
    'Fonctions LLM (IA)' as title,
    'Ces fonctions nécessitent une clé API LLM configurée.' as contents;

SELECT
    'card' as component,
    'llm_ask(prompt)' as title,
    'Pose une question à l''IA et retourne la réponse' as description;

SELECT
    'card' as component,
    'llm_summarize(text, max_words)' as title,
    'Résume un texte en N mots maximum' as description;

SELECT
    'card' as component,
    'llm_translate(text, lang)' as title,
    'Traduit un texte dans la langue spécifiée' as description;

SELECT
    'card' as component,
    'llm_extract_json(text, schema)' as title,
    'Extrait des données structurées d''un texte' as description;

-- Exemple d'utilisation
SELECT
    'text' as component,
    'Exemple d''utilisation' as title;

SELECT
    'debug' as component,
    'sql' as format,
    '-- Générer un ID unique pour un nouvel utilisateur
INSERT INTO users (id, name, email, created_at)
VALUES (
    uuid(),
    ''Jean Dupont'',
    ''jean@example.com'',
    now_utc()
);

-- Créer un slug pour un article
UPDATE articles
SET slug = slugify(title)
WHERE slug IS NULL;

-- Afficher le temps écoulé
SELECT
    title,
    time_ago(created_at) as publie
FROM articles;

-- Appeler une API externe
SELECT http_get_json(''https://api.example.com/data'');

-- Demander à l''IA de résumer un article
SELECT llm_summarize(content, 50) as resume
FROM articles
WHERE id = 1;' as data;
