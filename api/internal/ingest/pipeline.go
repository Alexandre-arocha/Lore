package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/google/uuid"

	"lore/api/internal/db"
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
	sourceTitle := ""
	if ext == ".rst" || ext == ".txt" {
		// Sphinx-style docs (e.g. Django) ship reStructuredText as .txt.
		body = []byte(rstToMarkdown(string(body)))
	} else if ext == ".xml" || ext == ".sgml" {
		raw := string(body)
		sourceTitle = docBookXMLTitle(raw)
		body = []byte(docBookXMLToMarkdown(raw))
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
		Title:     firstNonEmpty(fm.Title, sourceTitle, rendered.H1, exportTitle, firstTOCTitle(rendered.TOC), titleFromPath(f.Path)),
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

	// Process in path order so books that encode order in their filenames
	// (e.g. ch01-, ch02-) get a sensible default navigation order even when
	// pages carry no explicit front-matter order.
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	var (
		res   Result
		metas []DocMeta
		kept  []string
		seen  = map[string]string{}
		seq   int
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

		// Explicit front-matter order wins; otherwise fall back to path order.
		seq++
		position := doc.Position
		if position == 0 {
			position = seq
		}

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
			Position:    int32(position),
			WordCount:   int32(doc.WordCount),
		}); err != nil {
			return res, fmt.Errorf("upsert %s: %w", doc.Slug, err)
		}

		metas = append(metas, DocMeta{Slug: doc.Slug, Title: doc.Title, Position: position})
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

func firstTOCTitle(toc []TOCEntry) string {
	if len(toc) == 0 {
		return ""
	}
	return toc[0].Title
}
