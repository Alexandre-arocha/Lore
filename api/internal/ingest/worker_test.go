package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"lore/api/internal/db"
)

func TestSyncWorkerPreflightFailureRecordsErrorBeforePipeline(t *testing.T) {
	sourceID := uuid.New()
	runID := uuid.New()
	queries := &fakeSyncQueries{
		source: db.Source{
			ID:   sourceID,
			Slug: "demo",
			IngestConfig: json.RawMessage(`{
				"repo": "owner/repo",
				"branch": "main",
				"docs_path": "docs",
				"include_globs": ["**/*.md"]
			}`),
		},
		run: db.SyncRun{ID: runID, SourceID: sourceID, Status: db.SyncStatusRunning},
	}
	pipeline := &fakeSyncPipeline{}
	preflight := failingPreflight{err: errors.New("include/exclude globs matched zero supported docs")}
	worker := NewSyncWorker(queries, pipeline, nil, WithPreflightChecker(preflight))

	err := worker.Work(context.Background(), &river.Job[SyncSourceArgs]{
		Args: SyncSourceArgs{SourceID: sourceID},
	})
	if err == nil || !strings.Contains(err.Error(), "preflight") {
		t.Fatalf("Work error = %v, want preflight error", err)
	}
	if pipeline.called {
		t.Fatal("pipeline Sync was called after preflight failure")
	}
	if got := queries.statuses[len(queries.statuses)-1]; got != db.SourceStatusError {
		t.Fatalf("last status = %q, want error", got)
	}
	if queries.finished == nil {
		t.Fatal("FinishSyncRun was not called")
	}
	if queries.finished.Status != db.SyncStatusError {
		t.Fatalf("sync run status = %q, want error", queries.finished.Status)
	}
	if queries.finished.DocumentsProcessed != 0 {
		t.Fatalf("documents processed = %d, want 0", queries.finished.DocumentsProcessed)
	}
	if queries.finished.ErrorMessage == nil || !strings.Contains(*queries.finished.ErrorMessage, "matched zero") {
		t.Fatalf("error message = %v, want preflight detail", queries.finished.ErrorMessage)
	}
}

type failingPreflight struct {
	err error
}

func (f failingPreflight) Preflight(context.Context, PreflightSource) error {
	return f.err
}

type fakeSyncPipeline struct {
	called bool
}

func (f *fakeSyncPipeline) Sync(context.Context, db.Source) (Result, error) {
	f.called = true
	return Result{DocumentsProcessed: 1}, nil
}

type fakeSyncQueries struct {
	source   db.Source
	run      db.SyncRun
	statuses []db.SourceStatus
	finished *db.FinishSyncRunParams
}

func (f *fakeSyncQueries) GetSourceByID(_ context.Context, id uuid.UUID) (db.Source, error) {
	if id != f.source.ID {
		return db.Source{}, errors.New("unexpected source id")
	}
	return f.source, nil
}

func (f *fakeSyncQueries) SetSourceStatus(_ context.Context, arg db.SetSourceStatusParams) error {
	f.statuses = append(f.statuses, arg.Status)
	return nil
}

func (f *fakeSyncQueries) CreateSyncRun(context.Context, db.CreateSyncRunParams) (db.SyncRun, error) {
	return f.run, nil
}

func (f *fakeSyncQueries) FinishSyncRun(_ context.Context, arg db.FinishSyncRunParams) error {
	f.finished = &arg
	return nil
}

func (f *fakeSyncQueries) MarkSourceSynced(context.Context, db.MarkSourceSyncedParams) error {
	return nil
}
