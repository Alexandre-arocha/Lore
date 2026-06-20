// Package ingest implements the config-driven documentation ingestion pipeline:
// download a repo tarball from GitHub, render Markdown/MDX to highlighted HTML,
// extract a table of contents, and upsert pages plus a navigation tree.
package ingest

import (
	"encoding/json"
	"fmt"
)

// Config is the per-source ingest_config (stored as jsonb on sources).
type Config struct {
	Repo         string   `json:"repo"`          // "owner/name"
	Branch       string   `json:"branch"`        // defaults to "main"
	DocsPath     string   `json:"docs_path"`     // path within the repo to ingest
	IncludeGlobs []string `json:"include_globs"` // globs (relative to docs_path)
	ExcludeGlobs []string `json:"exclude_globs"`
}

// ParseConfig decodes and validates a source's ingest_config, filling defaults.
func ParseConfig(raw json.RawMessage) (Config, error) {
	var c Config
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &c); err != nil {
			return c, fmt.Errorf("ingest_config: %w", err)
		}
	}
	if c.Repo == "" {
		return c, fmt.Errorf("ingest_config.repo is required")
	}
	if c.Branch == "" {
		c.Branch = "main"
	}
	if len(c.IncludeGlobs) == 0 {
		c.IncludeGlobs = []string{"**/*.md", "**/*.mdx"}
	}
	return c, nil
}
