-- Page de test des fonctions LLM
-- Requiert une clé API LLM configurée

SELECT
    'shell' as component,
    'Test LLM' as title,
    'GoPage - Test des fonctions IA' as footer;

-- Navigation
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/functions">Fonctions</a> &gt; Test LLM</nav>' as html;

-- Vérifier si un prompt a été fourni
SELECT
    'text' as component,
    'Résultat de la requête LLM' as title
WHERE :prompt IS NOT NULL AND :prompt != '';

-- Afficher la question
SELECT
    'card' as component,
    'Votre question' as title,
    :prompt as description
WHERE :prompt IS NOT NULL AND :prompt != '';

-- Appeler le LLM et afficher la réponse
SELECT
    'card' as component,
    'Réponse de l''IA' as title,
    llm_ask(:prompt) as description
WHERE :prompt IS NOT NULL AND :prompt != '';

-- Formulaire si pas de prompt
SELECT
    'text' as component,
    'Posez une question à l''IA' as title,
    'Entrez votre question ci-dessous. La réponse sera générée par un modèle de langage.' as contents
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'form' as component,
    'llm-form' as id,
    'GET' as method,
    '/llm-test' as action,
    'Envoyer' as submit_label,
    'Question pour l''IA' as title
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'prompt' as name,
    'textarea' as type,
    'Votre question' as label,
    'Quelle est la capitale de la France?' as placeholder,
    1 as required
WHERE :prompt IS NULL OR :prompt = '';

-- Exemples de questions
SELECT
    'text' as component,
    'Exemples de questions' as title
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'list' as component
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'Quelle est la différence entre SQL et NoSQL?' as item,
    '/llm-test?prompt=Quelle%20est%20la%20diff%C3%A9rence%20entre%20SQL%20et%20NoSQL%3F' as link
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'Explique le concept de REST API en termes simples' as item,
    '/llm-test?prompt=Explique%20le%20concept%20de%20REST%20API%20en%20termes%20simples' as link
WHERE :prompt IS NULL OR :prompt = '';

SELECT
    'Donne 3 bonnes pratiques pour la sécurité des bases de données' as item,
    '/llm-test?prompt=Donne%203%20bonnes%20pratiques%20pour%20la%20s%C3%A9curit%C3%A9%20des%20bases%20de%20donn%C3%A9es' as link
WHERE :prompt IS NULL OR :prompt = '';

-- Autres fonctions LLM
SELECT
    'text' as component,
    'Autres fonctions LLM disponibles' as title;

SELECT
    'table' as component,
    'Liste des fonctions' as title;

SELECT
    'llm_ask(prompt)' as fonction,
    'Pose une question simple' as description;

SELECT
    'llm_ask_with_system(system, prompt)' as fonction,
    'Question avec contexte système' as description;

SELECT
    'llm_summarize(text, max_words)' as fonction,
    'Résume un texte' as description;

SELECT
    'llm_translate(text, language)' as fonction,
    'Traduit un texte' as description;

SELECT
    'llm_extract_json(text, schema)' as fonction,
    'Extrait du JSON structuré' as description;

SELECT
    'llm_classify(text, categories)' as fonction,
    'Classifie un texte' as description;

SELECT
    'llm_sentiment(text)' as fonction,
    'Analyse le sentiment' as description;

-- Retour
SELECT
    'text' as component,
    '<p><a href="/api-demo">← Retour à la démo API</a></p>' as html;
