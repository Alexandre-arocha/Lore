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
	"lore/api/internal/sourceconfig"
)

type sourceSeed = sourceconfig.Definition

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
	if err := validateSeeds(seeds); err != nil {
		log.Fatalf("seed: validate %s: %v", path, err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: connect: %v", err)
	}
	defer pool.Close()

	q := db.New(pool)

	for _, s := range seeds {
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

func validateSeeds(seeds []sourceSeed) error {
	return sourceconfig.ValidateDefinitions(seeds)
}
