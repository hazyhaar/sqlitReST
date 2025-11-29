-- Démonstration des fonctions HTTP et API
-- Page interactive pour tester les appels externes

SELECT
    'shell' as component,
    'Démo API & HTTP' as title,
    'GoPage - Test des fonctions HTTP' as footer;

-- Navigation
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/functions">Fonctions</a> &gt; Démo API</nav>' as html;

-- Introduction
SELECT
    'text' as component,
    'Test des fonctions HTTP' as title,
    'Cette page montre comment utiliser les fonctions HTTP pour appeler des API externes.' as contents;

-- Exemple: API publique JSONPlaceholder
SELECT
    'text' as component,
    'Appel API REST' as title;

SELECT
    'card' as component,
    'JSONPlaceholder API' as title,
    'Récupération d''un post depuis une API publique de test' as description;

-- Note: Cet appel sera fait à chaque chargement de page
-- En production, vous utiliseriez probablement du caching
SELECT
    'debug' as component,
    'json' as format,
    http_get('https://jsonplaceholder.typicode.com/posts/1') as data;

-- Exemple: Extraction de données JSON
SELECT
    'text' as component,
    'Extraction de données JSON' as title;

SELECT
    'table' as component,
    'Données extraites' as title;

WITH api_response AS (
    SELECT http_get('https://jsonplaceholder.typicode.com/posts/1') as json_data
)
SELECT
    json_extract_path(json_data, 'id') as id,
    json_extract_path(json_data, 'title') as titre,
    truncate(json_extract_path(json_data, 'body'), 50) as extrait
FROM api_response;

-- Formulaire de test HTTP
SELECT
    'text' as component,
    'Testeur HTTP interactif' as title;

SELECT
    'form' as component,
    'http-test-form' as id,
    'GET' as method,
    '/api-test' as action,
    'Tester' as submit_label,
    'Test d''URL' as title;

SELECT
    'url' as name,
    'text' as type,
    'URL à tester' as label,
    'https://jsonplaceholder.typicode.com/posts/1' as value,
    'https://api.example.com/endpoint' as placeholder,
    1 as required;

-- Résultat du test (si paramètre fourni)
SELECT
    'text' as component,
    'Résultat' as title
WHERE :url IS NOT NULL AND :url != '';

SELECT
    'debug' as component,
    'json' as format,
    http_get(:url) as data
WHERE :url IS NOT NULL AND :url != '';

-- Section LLM
SELECT
    'text' as component,
    'Fonctions LLM (Intelligence Artificielle)' as title,
    'Les fonctions LLM nécessitent une clé API configurée (LLM_API_KEY).' as contents;

SELECT
    'form' as component,
    'llm-test-form' as id,
    'GET' as method,
    '/llm-test' as action,
    'Demander' as submit_label,
    'Test LLM' as title;

SELECT
    'prompt' as name,
    'textarea' as type,
    'Votre question' as label,
    'Explique en une phrase ce qu''est SQLite.' as placeholder;

-- Documentation
SELECT
    'text' as component,
    'Documentation' as title;

SELECT
    'card' as component,
    'Fonctions disponibles' as title,
    '
<ul>
<li><code>http_get(url)</code> - Requête GET, retourne le body</li>
<li><code>http_get_json(url)</code> - GET avec validation JSON</li>
<li><code>http_post(url, content_type, body)</code> - POST générique</li>
<li><code>http_post_json(url, json)</code> - POST avec body JSON</li>
<li><code>http_post_form(url, data)</code> - POST form-urlencoded</li>
<li><code>http_head(url)</code> - HEAD, retourne les headers en JSON</li>
<li><code>webhook(url, payload)</code> - POST async (fire & forget)</li>
</ul>
' as footer;

SELECT
    'card' as component,
    'Considérations de sécurité' as title,
    '
<ul>
<li>Les requêtes HTTP ont un timeout configurable (défaut: 30s)</li>
<li>Seuls HTTP et HTTPS sont autorisés</li>
<li>Le body des réponses est limité à 10 Mo</li>
<li>Les webhooks sont asynchrones et ne bloquent pas</li>
</ul>
' as footer;
