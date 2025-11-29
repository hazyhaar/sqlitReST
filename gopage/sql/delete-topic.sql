-- Handler DELETE pour supprimer un topic
-- Paramètre requis: :id

-- En production:
-- DELETE FROM topics WHERE id = :id;
-- DELETE FROM replies WHERE topic_id = :id;

-- Redirection après suppression
SELECT '/topics' as redirect;

-- Message de succès (sera affiché via HX-Trigger sur la page de destination)
SELECT
    'Le topic a été supprimé' as message,
    'success' as message_type;
