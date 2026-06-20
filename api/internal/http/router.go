// Package http wires the Gin HTTP layer for the Atlas API.
package http

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server holds the dependencies shared by the HTTP handlers.
type Server struct {
	pool       *pgxpool.Pool
	adminToken string
}

// NewServer builds a Server with its dependencies.
func NewServer(pool *pgxpool.Pool, adminToken string) *Server {
	return &Server{pool: pool, adminToken: adminToken}
}

// Router builds the Gin engine with every route registered.
func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	api := r.Group("/api")
	{
		api.GET("/health", s.handleHealth)
	}

	return r
}
