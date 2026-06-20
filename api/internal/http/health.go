package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// pinger is the subset of *pgxpool.Pool used by the health check. Keeping it as
// an interface makes the health logic testable without a live database.
type pinger interface {
	Ping(ctx context.Context) error
}

func (s *Server) handleHealth(c *gin.Context) {
	var p pinger
	if s.pool != nil {
		p = s.pool
	}
	code, body := healthStatus(c.Request.Context(), p)
	c.JSON(code, body)
}

// healthStatus reports the API/database health. A nil pinger means the database
// was not configured (only happens in tests); the API still reports as up.
func healthStatus(ctx context.Context, p pinger) (int, gin.H) {
	body := gin.H{"status": "ok", "db": "ok"}

	if p == nil {
		body["db"] = "not_configured"
		return http.StatusOK, body
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := p.Ping(pingCtx); err != nil {
		body["status"] = "degraded"
		body["db"] = "error"
		return http.StatusServiceUnavailable, body
	}

	return http.StatusOK, body
}
