package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRoot(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `<!doctype html>
<html lang="pt-BR">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Lore API</title>
  <style>
    body { font-family: system-ui, -apple-system, Segoe UI, sans-serif; margin: 40px; line-height: 1.5; color: #111827; }
    main { max-width: 720px; }
    a { color: #2563eb; }
    code { background: #f3f4f6; border-radius: 4px; padding: 2px 5px; }
  </style>
</head>
<body>
  <main>
    <h1>Lore API</h1>
    <p>API rodando. Para ver dados, use os endpoints abaixo:</p>
    <ul>
      <li><a href="/api/health"><code>/api/health</code></a></li>
      <li><a href="/api/sources"><code>/api/sources</code></a></li>
      <li><a href="/api/search?q=javascript"><code>/api/search?q=javascript</code></a></li>
    </ul>
  </main>
</body>
</html>`)
}

func (s *Server) handleAPIRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":   "Lore API",
		"status": "ok",
		"endpoints": []string{
			"/api/health",
			"/api/sources",
			"/api/search?q=javascript",
		},
	})
}

func (s *Server) handleFavicon(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func (s *Server) handleNotFound(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "endpoint nao encontrado",
			"endpoints": []string{
				"/api/health",
				"/api/sources",
				"/api/search?q=javascript",
			},
		})
		return
	}
	s.handleRoot(c)
}
