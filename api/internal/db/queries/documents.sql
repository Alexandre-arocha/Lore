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

-- name: SearchDocuments :many
WITH search AS (
    SELECT websearch_to_tsquery('english', @search_query::text) AS tsq
)
SELECT
    d.id,
    d.source_id,
    s.slug AS source_slug,
    s.name AS source_name,
    s.official_url,
    s.license,
    d.slug,
    d.title,
    ts_headline(
        'english',
        d.content_text,
        search.tsq,
        'StartSel=<mark>, StopSel=</mark>, MaxFragments=2, MinWords=6, MaxWords=18'
    )::text AS excerpt,
    ts_rank(d.search_vector, search.tsq)::float8 AS rank
FROM documents d
JOIN sources s ON s.id = d.source_id
CROSS JOIN search
WHERE search.tsq @@ d.search_vector
  AND (sqlc.narg('source_slug')::text IS NULL OR s.slug = sqlc.narg('source_slug')::text)
ORDER BY rank DESC, d.title
LIMIT @limit_count::int;
