-- name: CreateSyncRun :one
INSERT INTO sync_runs (id, source_id, status)
VALUES ($1, $2, 'running')
RETURNING *;

-- name: FinishSyncRun :exec
UPDATE sync_runs
SET status = $2, documents_processed = $3, error_message = $4, finished_at = now()
WHERE id = $1;

-- name: ListSyncRunsBySource :many
SELECT * FROM sync_runs
WHERE source_id = $1
ORDER BY started_at DESC
LIMIT $2;

-- name: GetLatestSyncRun :one
SELECT * FROM sync_runs
WHERE source_id = $1
ORDER BY started_at DESC
LIMIT 1;
