package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		p        pinger
		wantCode int
		wantDB   string
		wantStat string
	}{
		{"healthy", fakePinger{nil}, http.StatusOK, "ok", "ok"},
		{"db down", fakePinger{errors.New("boom")}, http.StatusServiceUnavailable, "error", "degraded"},
		{"not configured", nil, http.StatusOK, "not_configured", "ok"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := healthStatus(context.Background(), tt.p)
			if code != tt.wantCode {
				t.Fatalf("code = %d, want %d", code, tt.wantCode)
			}
			if got := body["db"]; got != tt.wantDB {
				t.Fatalf("db = %v, want %q", got, tt.wantDB)
			}
			if got := body["status"]; got != tt.wantStat {
				t.Fatalf("status = %v, want %q", got, tt.wantStat)
			}
		})
	}
}

func TestRouterRegistersHealth(t *testing.T) {
	s := NewServer(nil, nil, nil, "")
	r := s.Router()

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/health = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRouterRegistersRootPages(t *testing.T) {
	s := NewServer(nil, nil, nil, "")
	r := s.Router()

	for _, path := range []string{"/", "/api", "/alguma-rota"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", path, w.Code, http.StatusOK)
			}
		})
	}
}

func TestRouterHandlesFavicon(t *testing.T) {
	s := NewServer(nil, nil, nil, "")
	r := s.Router()

	req := httptest.NewRequest(http.MethodGet, "/favicon.ico", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("GET /favicon.ico = %d, want %d", w.Code, http.StatusNoContent)
	}
}
