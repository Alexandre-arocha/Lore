package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/google/uuid"

	"github.com/lore/atlas/api/internal/db"
)

// ProcessedDoc is a fully rendered page ready to be persisted.
type ProcessedDoc struct {
	Slug      string
	Path      string
	Title     string
	HTML      string
	Text      string
	TOC       []TOCEntry
	Position  int
	WordCount int
	Warnings  int
}

// processFile turns one raw repo file into a ProcessedDoc. MDX files are
// sanitized to GFM first (graceful degradation of JSX/components).
func processFile(r *Renderer, f RawFile) (ProcessedDoc, error) {
	fm, body := FrontMatter{}, f.Content
	ext := strings.ToLower(path.Ext(f.Path))
	if ext == ".rst" {
		body = []byte(rstToMarkdown(string(body)))
	} else if ext == ".xml" {
		body = []byte(docBookXMLToMarkdown(string(body)))
	} else {
		fm, body = splitFrontMatter(f.Content)
	}

	var exportTitle string
	warnings := 0
	if ext == ".mdx" {
		var cleaned, exportDesc string
		cleaned, exportTitle, exportDesc, warnings = sanitizeMDX(string(body))
		_ = exportDesc
		body = []byte(cleaned)
	}

	rendered, err := r.Render(body)
	if err != nil {
		return ProcessedDoc{}, err
	}

	pos := 0
	if fm.Order != nil {
		pos = *fm.Order
	}

	return ProcessedDoc{
		Slug:      DeriveSlug(f.Path),
		Path:      f.Path,
		Title:     firstNonEmpty(fm.Title, rendered.H1, exportTitle, titleFromPath(f.Path)),
		HTML:      rendered.HTML,
		Text:      rendered.Text,
		TOC:       rendered.TOC,
		Position:  pos,
		WordCount: len(strings.Fields(rendered.Text)),
		Warnings:  warnings,
	}, nil
}

// Result summarizes a sync run.
type Result struct {
	DocumentsProcessed int
	FilesSkipped       int
	Warnings           int
}

// Pipeline runs the full ingestion for a source.
type Pipeline struct {
	fetcher  *GitHubFetcher
	renderer *Renderer
	queries  db.Querier
	logger   *slog.Logger
}

func NewPipeline(fetcher *GitHubFetcher, queries db.Querier, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		fetcher:  fetcher,
		renderer: NewRenderer(),
		queries:  queries,
		logger:   logger,
	}
}

// Sync downloads the source repo, renders every matching page, upserts them,
// prunes pages that disappeared, and rebuilds the navigation tree. Individual
// file failures are logged and skipped rather than aborting the whole run.
func (p *Pipeline) Sync(ctx context.Context, source db.Source) (Result, error) {
	cfg, err := ParseConfig(source.IngestConfig)
	if err != nil {
		return Result{}, err
	}

	files, err := p.fetcher.Fetch(ctx, cfg)
	if err != nil {
		return Result{}, err
	}

	var (
		res   Result
		metas []DocMeta
		kept  []string
		seen  = map[string]string{}
	)

	for _, f := range files {
		doc, err := processFile(p.renderer, f)
		if err != nil {
			res.FilesSkipped++
			p.logger.Warn("ingest: skip file", "source", source.Slug, "path", f.Path, "err", err)
			continue
		}
		if prev, dup := seen[doc.Slug]; dup {
			res.FilesSkipped++
			p.logger.Warn("ingest: duplicate slug", "source", source.Slug, "slug", doc.Slug, "kept", prev, "dropped", f.Path)
			continue
		}
		seen[doc.Slug] = f.Path
		res.Warnings += doc.Warnings

		tocJSON := json.RawMessage("[]")
		if len(doc.TOC) > 0 {
			if b, err := json.Marshal(doc.TOC); err == nil {
				tocJSON = b
			}
		}

		id, err := uuid.NewV7()
		if err != nil {
			return res, err
		}
		if _, err := p.queries.UpsertDocument(ctx, db.UpsertDocumentParams{
			ID:          id,
			SourceID:    source.ID,
			Slug:        doc.Slug,
			Path:        doc.Path,
			Title:       doc.Title,
			ContentHtml: doc.HTML,
			ContentText: doc.Text,
			Toc:         tocJSON,
			Position:    int32(doc.Position),
			WordCount:   int32(doc.WordCount),
		}); err != nil {
			return res, fmt.Errorf("upsert %s: %w", doc.Slug, err)
		}

		metas = append(metas, DocMeta{Slug: doc.Slug, Title: doc.Title, Position: doc.Position})
		kept = append(kept, doc.Slug)
		res.DocumentsProcessed++
	}

	if err := p.queries.PruneDocuments(ctx, db.PruneDocumentsParams{SourceID: source.ID, KeptSlugs: kept}); err != nil {
		return res, fmt.Errorf("prune: %w", err)
	}

	navJSON, err := json.Marshal(BuildNav(metas))
	if err != nil {
		return res, err
	}
	if err := p.queries.SetSourceNav(ctx, db.SetSourceNavParams{ID: source.ID, Nav: navJSON}); err != nil {
		return res, fmt.Errorf("set nav: %w", err)
	}

	return res, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
