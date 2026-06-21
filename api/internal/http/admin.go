package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"lore/api/internal/db"
	"lore/api/internal/ingest"
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
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token inválido"})
			return
		}
		c.Next()
	}
}

// upsertSourceRequest is the config-driven shape for creating/updating a source.
// It mirrors seed/sources.json so the same definitions work in both places.
type upsertSourceRequest struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Kind         string          `json:"kind"`
	Category     string          `json:"category"`
	Description  string          `json:"description"`
	LogoURL      *string         `json:"logo_url"`
	OfficialURL  string          `json:"official_url"`
	License      *string         `json:"license"`
	Version      *string         `json:"version"`
	IngestConfig json.RawMessage `json:"ingest_config"`
}

var validKinds = map[string]bool{"language": true, "framework": true, "library": true, "tool": true}

func (s *Server) handleUpsertSource(c *gin.Context) {
	var req upsertSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON inválido: " + err.Error()})
		return
	}
	if req.Slug == "" || req.Name == "" || req.OfficialURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slug, name e official_url são obrigatórios"})
		return
	}
	if !validKinds[req.Kind] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind inválido (language|framework|library|tool)"})
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

func (s *Server) handleSyncSource(c *gin.Context) {
	slug := c.Param("slug")

	source, err := s.queries.GetSourceBySlug(c.Request.Context(), slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source não encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if s.river == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "fila de jobs indisponível"})
		return
	}

	if _, err := s.river.Insert(c.Request.Context(), ingest.SyncSourceArgs{SourceID: source.ID}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "falha ao enfileirar sync: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "queued", "source": source.Slug})
}
