-- name: GetSourceBySlug :one
SELECT * FROM sources WHERE slug = $1;

-- name: GetSourceByID :one
SELECT * FROM sources WHERE id = $1;

-- name: ListSources :many
SELECT * FROM sources
WHERE (sqlc.narg('kind')::source_kind IS NULL OR kind = sqlc.narg('kind')::source_kind)
  AND (sqlc.narg('category')::text IS NULL OR category = sqlc.narg('category')::text)
ORDER BY category, name;

-- name: UpsertSource :one
INSERT INTO sources (
    id, slug, name, kind, category, description,
    logo_url, official_url, license, version, ingest_type, ingest_config
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
ON CONFLICT (slug) DO UPDATE SET
    name          = EXCLUDED.name,
    kind          = EXCLUDED.kind,
    category      = EXCLUDED.category,
    description   = EXCLUDED.description,
    logo_url      = EXCLUDED.logo_url,
    official_url  = EXCLUDED.official_url,
    license       = EXCLUDED.license,
    version       = EXCLUDED.version,
    ingest_type   = EXCLUDED.ingest_type,
    ingest_config = EXCLUDED.ingest_config,
    updated_at    = now()
RETURNING *;

-- name: SetSourceStatus :exec
UPDATE sources SET status = $2, updated_at = now() WHERE id = $1;

-- name: SetSourceNav :exec
UPDATE sources SET nav = $2, updated_at = now() WHERE id = $1;

-- name: MarkSourceSynced :exec
UPDATE sources
SET status = $2, last_synced_at = now(), updated_at = now()
WHERE id = $1;
