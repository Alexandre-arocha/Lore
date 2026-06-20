-- name: GetDocument :one
SELECT id, source_id, slug, path, title, content_html, toc, position, word_count, created_at, updated_at
FROM documents
WHERE source_id = $1 AND slug = $2;

-- name: ListDocumentsBySource :many
SELECT id, source_id, slug, path, title, toc, position, word_count, created_at, updated_at
FROM documents
WHERE source_id = $1
ORDER BY position, title;

-- name: UpsertDocument :one
INSERT INTO documents (
    id, source_id, slug, path, title, content_html, content_text, toc, position, word_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (source_id, slug) DO UPDATE SET
    path         = EXCLUDED.path,
    title        = EXCLUDED.title,
    content_html = EXCLUDED.content_html,
    content_text = EXCLUDED.content_text,
    toc          = EXCLUDED.toc,
    position     = EXCLUDED.position,
    word_count   = EXCLUDED.word_count,
    updated_at   = now()
RETURNING id;

-- name: DeleteDocumentsBySource :exec
DELETE FROM documents WHERE source_id = $1;

-- name: PruneDocuments :exec
-- Removes documents of a source whose slug is no longer present in the latest sync.
DELETE FROM documents
WHERE source_id = $1 AND NOT (slug = ANY(@kept_slugs::text[]));

-- name: CountDocumentsBySource :one
SELECT count(*) FROM documents WHERE source_id = $1;
