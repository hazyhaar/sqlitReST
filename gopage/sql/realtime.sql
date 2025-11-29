-- Démonstration SSE (Server-Sent Events)
-- Page de test temps réel

SELECT
    'shell' as component,
    'Temps Réel - SSE' as title,
    'GoPage - Server-Sent Events' as footer;

-- Navigation
SELECT
    'text' as component,
    '<nav><a href="/">Accueil</a> &gt; <a href="/functions">Fonctions</a> &gt; SSE</nav>' as html;

-- Introduction
SELECT
    'text' as component,
    'Server-Sent Events (SSE)' as title,
    'GoPage supporte les mises à jour en temps réel via SSE. Les fonctions SQL peuvent envoyer des notifications aux clients connectés.' as contents;

-- Statistiques en temps réel
SELECT
    'cards' as component,
    'Statistiques SSE' as title;

SELECT
    'Clients connectés' as title,
    sse_client_count() as description,
    'En ce moment' as footer;

-- Zone SSE
SELECT
    'text' as component,
    'Zone de réception SSE' as title,
    'Connectez-vous au flux SSE ci-dessous pour recevoir les messages.' as contents;

SELECT
    'card' as component,
    'Messages en temps réel' as title,
    '<div id="sse-messages"
          hx-ext="sse"
          sse-connect="/sse"
          sse-swap="message">
        <p><em>En attente de messages...</em></p>
    </div>' as footer;

-- Formulaire pour envoyer un message
SELECT
    'text' as component,
    'Envoyer un message' as title;

SELECT
    'form' as component,
    'sse-send-form' as id,
    'POST' as method,
    '/realtime-send' as action,
    'Envoyer' as submit_label,
    'Notification SSE' as title;

SELECT
    'message' as name,
    'text' as type,
    'Message' as label,
    'Hello from GoPage!' as placeholder,
    1 as required;

SELECT
    'event' as name,
    'text' as type,
    'Type d''événement' as label,
    'message' as value,
    'notification, alert, update...' as placeholder;

-- Documentation
SELECT
    'text' as component,
    'Fonctions SQL disponibles' as title;

SELECT
    'table' as component,
    'Fonctions SSE' as title;

SELECT
    'sse_notify(event, data)' as fonction,
    'Envoie à tous les clients' as description;

SELECT
    'sse_notify_channel(channel, event, data)' as fonction,
    'Envoie à un canal spécifique' as description;

SELECT
    'sse_broadcast(message)' as fonction,
    'Broadcast simple' as description;

SELECT
    'sse_client_count()' as fonction,
    'Nombre de clients connectés' as description;

SELECT
    'sse_channel_count(channel)' as fonction,
    'Clients dans un canal' as description;

-- Code exemple
SELECT
    'debug' as component,
    'sql' as format,
    '-- Envoyer une notification lors d''un INSERT
INSERT INTO messages (content, author) VALUES (:content, :author);

-- Notifier tous les clients
SELECT sse_notify(''new_message'',
    json_object(''content'', :content, ''author'', :author));

-- Notifier un canal spécifique
SELECT sse_notify_channel(''chat_room_1'', ''message'', :content);

-- Vérifier les connexions
SELECT sse_client_count() as connected_clients;' as data;

-- JavaScript client
SELECT
    'text' as component,
    'Code JavaScript client' as title;

SELECT
    'debug' as component,
    'javascript' as format,
    '// Connexion SSE avec HTMX (automatique avec hx-ext="sse")
// Ou manuellement:
const source = new EventSource(''/sse?channel=my_channel'');

source.addEventListener(''message'', (e) => {
    console.log(''Message:'', e.data);
});

source.addEventListener(''connected'', (e) => {
    const data = JSON.parse(e.data);
    console.log(''Client ID:'', data.clientId);
});

source.addEventListener(''error'', (e) => {
    console.error(''SSE error'', e);
});' as data;
