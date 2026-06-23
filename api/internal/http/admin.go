package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"lore/api/internal/db"
	"lore/api/internal/ingest"
	"lore/api/internal/sourceconfig"
)

// requireAdmin gates the /api/admin routes behind a shared token. When no admin
// token is configured the routes are disabled entirely.
func (s *Server) requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if s.adminToken == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "admin desabilitado: defina ADMIN_TOKEN"})
			return
		}
		if c.GetHeader("X-Admin-Token") != s.adminToken {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token invalido"})
			return
		}
		c.Next()
	}
}

// upsertSourceRequest mirrors seed/sources.json so the same definitions work in
// both places.
type upsertSourceRequest = sourceconfig.Definition

func (s *Server) handleUpsertSource(c *gin.Context) {
	var req upsertSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON invalido: " + err.Error()})
		return
	}
	if err := validateUpsertSourceRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ingestConfig := req.IngestConfig
	if len(ingestConfig) == 0 {
		ingestConfig = json.RawMessage(`{}`)
	}

	id, err := uuid.NewV7()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "uuid"})
		return
	}

	source, err := s.queries.UpsertSource(c.Request.Context(), db.UpsertSourceParams{
		ID:           id,
		Slug:         req.Slug,
		Name:         req.Name,
		Kind:         db.SourceKind(req.Kind),
		Category:     req.Category,
		Description:  req.Description,
		LogoUrl:      req.LogoURL,
		OfficialUrl:  req.OfficialURL,
		License:      req.License,
		Version:      req.Version,
		IngestType:   db.SourceIngestTypeGithubMarkdown,
		IngestConfig: ingestConfig,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao salvar source: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, sourceDetail(source))
}

func validateUpsertSourceRequest(req upsertSourceRequest) error {
	return sourceconfig.ValidateDefinition(req)
}

func (s *Server) handleAdminSourcesStatus(c *gin.Context) {
	if !s.hasQueries(c) {
		return
	}

	sources, err := s.queries.ListSources(c.Request.Context(), db.ListSourcesParams{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]adminSourceStatusResponse, 0, len(sources))
	for _, row := range sources {
		latestRun, err := s.queries.GetLatestSyncRun(c.Request.Context(), row.Source.ID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var latest *db.SyncRun
		if err == nil {
			latest = &latestRun
		}
		items = append(items, adminSourceStatus(row, latest))
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

type adminSourceStatusResponse struct {
	Slug         string                `json:"slug"`
	Status       string                `json:"status"`
	DocCount     int32                 `json:"doc_count"`
	LastSyncedAt *time.Time            `json:"last_synced_at"`
	LatestRun    *adminSyncRunResponse `json:"latest_run"`
}

type adminSyncRunResponse struct {
	Status             string     `json:"status"`
	DocumentsProcessed int32      `json:"documents_processed"`
	ErrorMessage       *string    `json:"error_message"`
	StartedAt          *time.Time `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at"`
}

func adminSourceStatus(row db.ListSourcesRow, latest *db.SyncRun) adminSourceStatusResponse {
	resp := adminSourceStatusResponse{
		Slug:         row.Source.Slug,
		Status:       string(row.Source.Status),
		DocCount:     row.DocCount,
		LastSyncedAt: timePtr(row.Source.LastSyncedAt),
	}
	if latest != nil {
		resp.LatestRun = &adminSyncRunResponse{
			Status:             string(latest.Status),
			DocumentsProcessed: latest.DocumentsProcessed,
			ErrorMessage:       latest.ErrorMessage,
			StartedAt:          timePtr(latest.StartedAt),
			FinishedAt:         timePtr(latest.FinishedAt),
		}
	}
	return resp
}

func timePtr(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

func (s *Server) handleSyncSource(c *gin.Context) {
	slug := c.Param("slug")

	source, err := s.queries.GetSourceBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source nao encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if s.river == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fila de jobs indisponivel"})
		return
	}

	if _, err := s.river.Insert(c.Request.Context(), ingest.SyncSourceArgs{SourceID: source.ID}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao enfileirar sync: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "source": source.Slug})
}
