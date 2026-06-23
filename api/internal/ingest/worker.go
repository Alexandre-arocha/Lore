package ingest

import (
	"context"
	"encoding/json"
	"fmt"
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
	queries   syncQueries
	pipeline  syncPipeline
	preflight PreflightChecker
	logger    *slog.Logger
}

type syncQueries interface {
	GetSourceByID(context.Context, uuid.UUID) (db.Source, error)
	SetSourceStatus(context.Context, db.SetSourceStatusParams) error
	CreateSyncRun(context.Context, db.CreateSyncRunParams) (db.SyncRun, error)
	FinishSyncRun(context.Context, db.FinishSyncRunParams) error
	MarkSourceSynced(context.Context, db.MarkSourceSyncedParams) error
}

type syncPipeline interface {
	Sync(context.Context, db.Source) (Result, error)
}

type PreflightSource struct {
	Slug         string
	IngestConfig json.RawMessage
}

type PreflightChecker interface {
	Preflight(context.Context, PreflightSource) error
}

type SyncWorkerOption func(*SyncWorker)

func WithPreflightChecker(checker PreflightChecker) SyncWorkerOption {
	return func(w *SyncWorker) {
		w.preflight = checker
	}
}

func NewSyncWorker(queries syncQueries, pipeline syncPipeline, logger *slog.Logger, opts ...SyncWorkerOption) *SyncWorker {
	if logger == nil {
		logger = slog.Default()
	}
	w := &SyncWorker{queries: queries, pipeline: pipeline, logger: logger}
	for _, opt := range opts {
		opt(w)
	}
	return w
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

	if w.preflight != nil {
		if err := w.preflight.Preflight(ctx, PreflightSource{Slug: source.Slug, IngestConfig: source.IngestConfig}); err != nil {
			syncErr := fmt.Errorf("preflight: %w", err)
			w.finishSyncError(run, source, Result{}, syncErr)
			w.logger.Error("ingest: preflight failed", "source", source.Slug, "err", err)
			return syncErr
		}
	}

	res, syncErr := w.pipeline.Sync(ctx, source)
	if syncErr != nil {
		w.finishSyncError(run, source, res, syncErr)
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

func (w *SyncWorker) finishSyncError(run db.SyncRun, source db.Source, res Result, syncErr error) {
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
}
