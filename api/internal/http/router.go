// Package http wires the Gin HTTP layer for the Atlas API.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/lore/atlas/api/internal/db"
)

// Server holds the dependencies shared by the HTTP handlers.
type Server struct {
	pool       *pgxpool.Pool
	queries    *db.Queries
	river      *river.Client[pgx.Tx]
	adminToken string
}

// NewServer builds a Server with its dependencies. river may be nil (e.g. in
// tests) — admin sync endpoints then report the queue as unavailable.
func NewServer(pool *pgxpool.Pool, queries *db.Queries, riverClient *river.Client[pgx.Tx], adminToken string) *Server {
	return &Server{
		pool:       pool,
		queries:    queries,
		river:      riverClient,
		adminToken: adminToken,
	}
}

// Router builds the Gin engine with every route registered.
func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	{
		api.GET("/health", s.handleHealth)
		api.GET("/sources", s.handleListSources)
		api.GET("/sources/:slug", s.handleGetSource)
		api.GET("/sources/:slug/docs/*docSlug", s.handleGetDocument)
		api.GET("/search", s.handleSearch)

		admin := api.Group("/admin", s.requireAdmin())
		{
			admin.POST("/sources", s.handleUpsertSource)
			admin.POST("/sources/:slug/sync", s.handleSyncSource)
		}
	}

	return r
}
