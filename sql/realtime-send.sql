-- Handler pour envoyer des messages SSE
-- Appelé par le formulaire de la page realtime

-- Envoyer la notification SSE
SELECT sse_notify(
    COALESCE(:event, 'message'),
    COALESCE(:message, 'Hello!')
);

-- Rediriger vers la page realtime avec un message de succès
SELECT
    'redirect' as component,
    '/realtime' as url,
    'Message envoyé!' as flash_success;
