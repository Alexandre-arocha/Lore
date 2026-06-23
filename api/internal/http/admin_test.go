package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"lore/api/internal/db"
)

func TestValidateUpsertSourceRequest(t *testing.T) {
	license := "MIT"
	valid := upsertSourceRequest{
		Slug:        "demo",
		Name:        "Demo",
		Kind:        "language",
		Category:    "backend",
		OfficialURL: "https://example.com",
		License:     &license,
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	}

	if err := validateUpsertSourceRequest(valid); err != nil {
		t.Fatalf("validateUpsertSourceRequest(valid): %v", err)
	}

	cases := map[string]upsertSourceRequest{
		"missing license": func() upsertSourceRequest {
			req := valid
			req.License = nil
			return req
		}(),
		"missing include globs": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs"}`)
			return req
		}(),
		"bad repo shape": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"repo","branch":"main","docs_path":"docs","include_globs":["**/*.md"]}`)
			return req
		}(),
		"absolute glob": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["/docs/*.md"]}`)
			return req
		}(),
		"backslash glob": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["guide\\*.md"]}`)
			return req
		}(),
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateUpsertSourceRequest(req); err == nil {
				t.Fatal("validateUpsertSourceRequest returned nil, want error")
			}
		})
	}
}

func TestAdminSourcesStatus(t *testing.T) {
	sourceID := uuid.New()
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	errMessage := "preflight: include/exclude globs matched zero supported docs"
	queries := &adminStatusQueries{
		sources: []db.ListSourcesRow{
			{
				Source: db.Source{
					ID:           sourceID,
					Slug:         "demo",
					Status:       db.SourceStatusError,
					LastSyncedAt: pgtype.Timestamptz{Time: now, Valid: true},
				},
				DocCount: 42,
			},
		},
		latest: map[uuid.UUID]db.SyncRun{
			sourceID: {
				ID:                 uuid.New(),
				SourceID:           sourceID,
				Status:             db.SyncStatusError,
				DocumentsProcessed: 0,
				ErrorMessage:       &errMessage,
				StartedAt:          pgtype.Timestamptz{Time: now.Add(-time.Minute), Valid: true},
				FinishedAt:         pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
	}
	server := NewServer(nil, queries, nil, "secret").Router()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/sources/status", nil)
	req.Header.Set("X-Admin-Token", "secret")
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/admin/sources/status = %d, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`"slug":"demo"`,
		`"status":"error"`,
		`"doc_count":42`,
		`"latest_run"`,
		`"documents_processed":0`,
		`"error_message":"preflight: include/exclude globs matched zero supported docs"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
}

type adminStatusQueries struct {
	sources []db.ListSourcesRow
	latest  map[uuid.UUID]db.SyncRun
}

func (q *adminStatusQueries) CountDocumentsBySource(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

func (q *adminStatusQueries) CreateSyncRun(context.Context, db.CreateSyncRunParams) (db.SyncRun, error) {
	return db.SyncRun{}, nil
}

func (q *adminStatusQueries) DeleteDocumentsBySource(context.Context, uuid.UUID) error {
	return nil
}

func (q *adminStatusQueries) FinishSyncRun(context.Context, db.FinishSyncRunParams) error {
	return nil
}

func (q *adminStatusQueries) GetDocument(context.Context, db.GetDocumentParams) (db.GetDocumentRow, error) {
	return db.GetDocumentRow{}, nil
}

func (q *adminStatusQueries) GetLatestSyncRun(_ context.Context, sourceID uuid.UUID) (db.SyncRun, error) {
	if run, ok := q.latest[sourceID]; ok {
		return run, nil
	}
	return db.SyncRun{}, pgx.ErrNoRows
}

func (q *adminStatusQueries) GetSourceByID(context.Context, uuid.UUID) (db.Source, error) {
	return db.Source{}, nil
}

func (q *adminStatusQueries) GetSourceBySlug(context.Context, string) (db.Source, error) {
	return db.Source{}, nil
}

func (q *adminStatusQueries) ListDocumentsBySource(context.Context, uuid.UUID) ([]db.ListDocumentsBySourceRow, error) {
	return nil, nil
}

func (q *adminStatusQueries) ListSources(context.Context, db.ListSourcesParams) ([]db.ListSourcesRow, error) {
	return q.sources, nil
}

func (q *adminStatusQueries) ListSyncRunsBySource(context.Context, db.ListSyncRunsBySourceParams) ([]db.SyncRun, error) {
	return nil, nil
}

func (q *adminStatusQueries) MarkSourceSynced(context.Context, db.MarkSourceSyncedParams) error {
	return nil
}

func (q *adminStatusQueries) PruneDocuments(context.Context, db.PruneDocumentsParams) error {
	return nil
}

func (q *adminStatusQueries) SearchDocuments(context.Context, db.SearchDocumentsParams) ([]db.SearchDocumentsRow, error) {
	return nil, nil
}

func (q *adminStatusQueries) SetSourceNav(context.Context, db.SetSourceNavParams) error {
	return nil
}

func (q *adminStatusQueries) SetSourceStatus(context.Context, db.SetSourceStatusParams) error {
	return nil
}

func (q *adminStatusQueries) UpsertDocument(context.Context, db.UpsertDocumentParams) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (q *adminStatusQueries) UpsertSource(context.Context, db.UpsertSourceParams) (db.Source, error) {
	return db.Source{}, nil
}
