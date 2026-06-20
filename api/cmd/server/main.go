// Command server is the entrypoint for the Atlas HTTP API. It also runs the
// River worker that processes documentation ingestion jobs.
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lore/atlas/api/internal/config"
	"github.com/lore/atlas/api/internal/db"
	atlashttp "github.com/lore/atlas/api/internal/http"
	"github.com/lore/atlas/api/internal/ingest"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("db: ping: %v", err)
	}

	// River keeps its own tables; ensure they exist before starting workers.
	if err := ingest.MigrateRiver(ctx, pool); err != nil {
		log.Fatalf("%v", err)
	}

	queries := db.New(pool)
	logger := slog.Default()

	fetcher := ingest.NewGitHubFetcher(cfg.GithubToken)
	pipeline := ingest.NewPipeline(fetcher, queries, logger)
	worker := ingest.NewSyncWorker(queries, pipeline, logger)

	riverClient, err := ingest.NewRiverClient(pool, worker)
	if err != nil {
		log.Fatalf("river: client: %v", err)
	}
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("river: start: %v", err)
	}

	srv := atlashttp.NewServer(pool, queries, riverClient, cfg.AdminToken)
	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("atlas api listening on :%s (env=%s)", cfg.Port, cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("river stop: %v", err)
	}
}
