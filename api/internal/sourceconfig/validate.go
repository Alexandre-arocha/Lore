package sourceconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"lore/api/internal/ingest"
)

// Definition is the config-driven shape accepted by seed/sources.json and the
// admin source upsert endpoint.
type Definition struct {
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

func ValidKind(kind string) bool {
	return validKinds[kind]
}

func ValidateDefinitions(definitions []Definition) error {
	if len(definitions) == 0 {
		return errors.New("at least one source is required")
	}

	seen := map[string]struct{}{}
	for _, def := range definitions {
		if err := ValidateDefinition(def); err != nil {
			return fmt.Errorf("%s: %w", sourceLabel(def.Slug), err)
		}
		if _, ok := seen[def.Slug]; ok {
			return fmt.Errorf("%s: duplicate slug", sourceLabel(def.Slug))
		}
		seen[def.Slug] = struct{}{}
	}

	return nil
}

func ValidateDefinition(def Definition) error {
	slug := strings.TrimSpace(def.Slug)
	if slug == "" || strings.TrimSpace(def.Name) == "" || strings.TrimSpace(def.OfficialURL) == "" {
		return errors.New("slug, name and official_url are required")
	}
	if slug != def.Slug {
		return errors.New("slug must be trimmed")
	}
	if !ValidKind(def.Kind) {
		return errors.New("invalid kind (language|framework|library|tool)")
	}
	if strings.TrimSpace(def.Category) == "" {
		return errors.New("category is required")
	}
	if def.License == nil || strings.TrimSpace(*def.License) == "" {
		return errors.New("license is required")
	}
	if len(def.IngestConfig) == 0 {
		return errors.New("ingest_config is required")
	}

	var rawCfg ingest.Config
	if err := json.Unmarshal(def.IngestConfig, &rawCfg); err != nil {
		return fmt.Errorf("ingest_config: %w", err)
	}
	if len(rawCfg.IncludeGlobs) == 0 {
		return errors.New("ingest_config.include_globs is required")
	}

	cfg, err := ingest.ParseConfig(def.IngestConfig)
	if err != nil {
		return err
	}
	if strings.Count(cfg.Repo, "/") != 1 {
		return errors.New("ingest_config.repo must be owner/name")
	}
	for _, glob := range append(cfg.IncludeGlobs, cfg.ExcludeGlobs...) {
		if strings.TrimSpace(glob) == "" {
			return errors.New("empty glob is not allowed")
		}
		if strings.HasPrefix(glob, "/") || strings.Contains(glob, "\\") {
			return fmt.Errorf("glob %q must be relative and use forward slashes", glob)
		}
	}

	return nil
}

func sourceLabel(slug string) string {
	if strings.TrimSpace(slug) == "" {
		return "source"
	}
	return fmt.Sprintf("source %q", slug)
}
