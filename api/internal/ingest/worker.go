package ingest

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"lore/api/internal/db"
)

// SyncSourceArgs are the args for the SyncSourceJob River job.
type SyncSourceArgs struct {
	SourceID uuid.UUID `json:"source_id"`
}

// Kind implements river.JobArgs.
func (SyncSourceArgs) Kind() string { return "sync_source" }

// SyncWorker runs an ingestion sync for one source, updating source.status and
// writing a sync_run row for observability.
type SyncWorker struct {
	river.WorkerDefaults[SyncSourceArgs]
	queries  db.Querier
	pipeline *Pipeline
	logger   *slog.Logger
}

func NewSyncWorker(queries db.Querier, pipeline *Pipeline, logger *slog.Logger) *SyncWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncWorker{queries: queries, pipeline: pipeline, logger: logger}
}

func (w *SyncWorker) Work(ctx context.Context, job *river.Job[SyncSourceArgs]) error {
	source, err := w.queries.GetSourceByID(ctx, job.Args.SourceID)
	if err != nil {
		return err
	}

	_ = w.queries.SetSourceStatus(ctx, db.SetSourceStatusParams{
		ID: source.ID, Status: db.SourceStatusSyncing,
	})

	runID, err := uuid.NewV7()
	if err != nil {
		return err
	}
	run, err := w.queries.CreateSyncRun(ctx, db.CreateSyncRunParams{ID: runID, SourceID: source.ID})
	if err != nil {
		return err
	}

	res, syncErr := w.pipeline.Sync(ctx, source)
	if syncErr != nil {
		msg := syncErr.Error()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = w.queries.SetSourceStatus(cleanupCtx, db.SetSourceStatusParams{ID: source.ID, Status: db.SourceStatusError})
		_ = w.queries.FinishSyncRun(cleanupCtx, db.FinishSyncRunParams{
			ID:                 run.ID,
			Status:             db.SyncStatusError,
			DocumentsProcessed: int32(res.DocumentsProcessed),
			ErrorMessage:       &msg,
		})
		w.logger.Error("ingest: sync failed", "source", source.Slug, "err", syncErr)
		return syncErr
	}

	_ = w.queries.MarkSourceSynced(ctx, db.MarkSourceSyncedParams{ID: source.ID, Status: db.SourceStatusActive})
	_ = w.queries.FinishSyncRun(ctx, db.FinishSyncRunParams{
		ID:                 run.ID,
		Status:             db.SyncStatusSuccess,
		DocumentsProcessed: int32(res.DocumentsProcessed),
	})
	w.logger.Info("ingest: sync ok",
		"source", source.Slug,
		"docs", res.DocumentsProcessed,
		"skipped", res.FilesSkipped,
		"warnings", res.Warnings)
	return nil
}
