package http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"lore/api/internal/db"
)

const (
	defaultSearchLimit = 20
	maxSearchLimit     = 50
)

func (s *Server) handleListSources(c *gin.Context) {
	if !s.hasQueries(c) {
		return
	}

	var kind *db.SourceKind
	if raw := strings.TrimSpace(c.Query("kind")); raw != "" {
		if !validKinds[raw] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kind invalido (language|framework|library|tool)"})
			return
		}
		k := db.SourceKind(raw)
		kind = &k
	}

	var category *string
	if raw := strings.TrimSpace(c.Query("category")); raw != "" {
		category = &raw
	}

	sources, err := s.queries.ListSources(c.Request.Context(), db.ListSourcesParams{
		Kind:     kind,
		Category: category,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]sourceResponse, 0, len(sources))
	for _, row := range sources {
		items = append(items, sourceSummary(row.Source, row.DocCount))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) handleGetSource(c *gin.Context) {
	if !s.hasQueries(c) {
		return
	}

	source, err := s.queries.GetSourceBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source nao encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sourceDetail(source))
}

func (s *Server) handleGetDocument(c *gin.Context) {
	if !s.hasQueries(c) {
		return
	}

	source, err := s.queries.GetSourceBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "source nao encontrada"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	docSlug := strings.Trim(strings.TrimSpace(c.Param("docSlug")), "/")
	if docSlug == "" {
		docSlug = "index"
	}

	doc, err := s.queries.GetDocument(c.Request.Context(), db.GetDocumentParams{
		SourceID: source.ID,
		Slug:     docSlug,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "documento nao encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, documentDetail(source, doc))
}

func (s *Server) handleSearch(c *gin.Context) {
	if !s.hasQueries(c) {
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"items": []searchResultResponse{}})
		return
	}

	limit := parseSearchLimit(c.Query("limit"))
	var sourceSlug *string
	if raw := strings.TrimSpace(c.Query("source")); raw != "" {
		sourceSlug = &raw
	}

	rows, err := s.queries.SearchDocuments(c.Request.Context(), db.SearchDocumentsParams{
		SearchQuery: q,
		SourceSlug:  sourceSlug,
		LimitCount:  int32(limit),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	items := make([]searchResultResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, searchResult(row))
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Server) hasQueries(c *gin.Context) bool {
	if s.queries != nil {
		return true
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "banco indisponivel"})
	return false
}

func parseSearchLimit(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return defaultSearchLimit
	}
	if n > maxSearchLimit {
		return maxSearchLimit
	}
	return n
}
