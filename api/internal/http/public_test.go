package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"lore/api/internal/db"
)

func TestGetSourceIncludesDocCount(t *testing.T) {
	sourceID := uuid.New()
	queries := &adminStatusQueries{
		sourceBySlug: map[string]db.Source{
			"typescript": {
				ID:          sourceID,
				Slug:        "typescript",
				Name:        "TypeScript",
				Kind:        db.SourceKindLanguage,
				Category:    "frontend",
				Description: "Docs",
				OfficialUrl: "https://www.typescriptlang.org/docs/",
				Status:      db.SourceStatusActive,
			},
		},
		docCounts: map[uuid.UUID]int64{sourceID: 77},
	}
	server := NewServer(nil, queries, nil, "").Router()

	req := httptest.NewRequest(http.MethodGet, "/api/sources/typescript", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/sources/typescript = %d, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"doc_count":77`) {
		t.Fatalf("response missing doc_count 77: %s", w.Body.String())
	}
}
