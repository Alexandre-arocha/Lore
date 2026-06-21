// Command seed upserts source definitions from a JSON config file into the
// database (config only — no document content). Run from the api/ directory:
//
//	go run ./cmd/seed                  # uses seed/sources.json
//	go run ./cmd/seed path/to/file.json
//
// Source ingestion is config-driven: adding or removing a source means editing
// the JSON, not the code. The same shape is accepted by POST /api/admin/sources.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"lore/api/internal/config"
	"lore/api/internal/db"
)

type sourceSeed struct {
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

func main() {
	path := "seed/sources.json"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("seed: read %s: %v", path, err)
	}

	var seeds []sourceSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		log.Fatalf("seed: parse %s: %v", path, err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)

	for _, s := range seeds {
		if s.Slug == "" || s.Name == "" || s.OfficialURL == "" {
			log.Fatalf("seed: %q: slug, name and official_url are required", s.Slug)
		}
		if !validKinds[s.Kind] {
			log.Fatalf("seed: %q: invalid kind %q", s.Slug, s.Kind)
		}

		ingestConfig := s.IngestConfig
		if len(ingestConfig) == 0 {
			ingestConfig = json.RawMessage(`{}`)
		}

		id, err := uuid.NewV7()
		if err != nil {
			log.Fatalf("seed: uuid: %v", err)
		}

		src, err := q.UpsertSource(ctx, db.UpsertSourceParams{
			ID:           id,
			Slug:         s.Slug,
			Name:         s.Name,
			Kind:         db.SourceKind(s.Kind),
			Category:     s.Category,
			Description:  s.Description,
			LogoUrl:      s.LogoURL,
			OfficialUrl:  s.OfficialURL,
			License:      s.License,
			Version:      s.Version,
			IngestType:   db.SourceIngestTypeGithubMarkdown,
			IngestConfig: ingestConfig,
		})
		if err != nil {
			log.Fatalf("seed: upsert %q: %v", s.Slug, err)
		}
		fmt.Printf("  upserted %-10s %s (%s/%s)\n", src.Slug, src.ID, src.Kind, src.Category)
	}

	fmt.Printf("seed: %d sources upserted\n", len(seeds))
}
