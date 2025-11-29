-- Handler POST pour répondre à un topic
-- Traite la soumission du formulaire de réponse

-- Vérifier que les paramètres requis sont présents
-- En production, on ferait : INSERT INTO replies (topic_id, content, author) VALUES (:topic_id, :content, :author)

-- Simulation de l'insertion
-- SELECT 'insert' as action, :topic_id as topic_id, :content as content;

-- Retourner un fragment HTML pour HTMX (la nouvelle réponse)
SELECT
    'fragment' as component,
    '<article class="card fade-in">
        <h4>Nouvelle réponse</h4>
        <small>Vous - à l''instant</small>
        <p>' || COALESCE(:content, 'Contenu de la réponse...') || '</p>
        <footer><small>À l''instant</small></footer>
    </article>' as fragment;

-- Message de succès
SELECT
    'Votre réponse a été publiée !' as message,
    'success' as message_type;
