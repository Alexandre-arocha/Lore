// Command migrate applies golang-migrate migrations from the ./migrations
// directory. Run it from the api/ directory:
//
//	go run ./cmd/migrate up      # apply all pending migrations
//	go run ./cmd/migrate down    # roll back the most recent migration
//	go run ./cmd/migrate drop    # drop everything (dev only)
//
// It reads DATABASE_URL from the environment (or api/.env).
package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"lore/api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// Read migrations from the real ./migrations directory via an fs.FS. Using
	// iofs (instead of a file:// URL) avoids Windows path/URL parsing issues.
	src, err := iofs.New(os.DirFS("migrations"), ".")
	if err != nil {
		log.Fatalf("migrate: source: %v", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("migrate: init: %v", err)
	}
	defer m.Close()

	switch cmd {
	case "up":
		err = m.Up()
	case "down":
		err = m.Steps(-1)
	case "drop":
		err = m.Drop()
	default:
		log.Fatalf("migrate: unknown command %q (use: up | down | drop)", cmd)
	}

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("migrate: %s: %v", cmd, err)
	}
	log.Printf("migrate: %s ok", cmd)
}
