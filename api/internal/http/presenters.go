package http

import (
	"encoding/json"
	"time"

	"github.com/lore/atlas/api/internal/db"
)

// sourceResponse is the JSON shape returned for a source (detail view includes
// the navigation tree).
type sourceResponse struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Kind         string          `json:"kind"`
	Category     string          `json:"category"`
	Description  string          `json:"description"`
	LogoURL      *string         `json:"logo_url"`
	OfficialURL  string          `json:"official_url"`
	License      *string         `json:"license"`
	Version      *string         `json:"version"`
	Status       string          `json:"status"`
	Nav          json.RawMessage `json:"nav,omitempty"`
	LastSyncedAt *time.Time      `json:"last_synced_at"`
}

func sourceSummary(s db.Source) sourceResponse {
	r := sourceDetail(s)
	r.Nav = nil
	return r
}

func sourceDetail(s db.Source) sourceResponse {
	r := sourceResponse{
		Slug:        s.Slug,
		Name:        s.Name,
		Kind:        string(s.Kind),
		Category:    s.Category,
		Description: s.Description,
		LogoURL:     s.LogoUrl,
		OfficialURL: s.OfficialUrl,
		License:     s.License,
		Version:     s.Version,
		Status:      string(s.Status),
		Nav:         s.Nav,
	}
	if s.LastSyncedAt.Valid {
		t := s.LastSyncedAt.Time
		r.LastSyncedAt = &t
	}
	return r
}

type documentResponse struct {
	Source      sourceAttribution `json:"source"`
	Slug        string            `json:"slug"`
	Path        string            `json:"path"`
	Title       string            `json:"title"`
	ContentHTML string            `json:"content_html"`
	TOC         json.RawMessage   `json:"toc"`
	Position    int32             `json:"position"`
	WordCount   int32             `json:"word_count"`
	UpdatedAt   *time.Time        `json:"updated_at"`
}

type sourceAttribution struct {
	Slug        string  `json:"slug"`
	Name        string  `json:"name"`
	OfficialURL string  `json:"official_url"`
	License     *string `json:"license"`
}

func documentDetail(source db.Source, doc db.GetDocumentRow) documentResponse {
	r := documentResponse{
		Source: sourceAttribution{
			Slug:        source.Slug,
			Name:        source.Name,
			OfficialURL: source.OfficialUrl,
			License:     source.License,
		},
		Slug:        doc.Slug,
		Path:        doc.Path,
		Title:       doc.Title,
		ContentHTML: doc.ContentHtml,
		TOC:         doc.Toc,
		Position:    doc.Position,
		WordCount:   doc.WordCount,
	}
	if doc.UpdatedAt.Valid {
		t := doc.UpdatedAt.Time
		r.UpdatedAt = &t
	}
	return r
}

type searchResultResponse struct {
	Source      sourceAttribution `json:"source"`
	DocumentURL string            `json:"document_url"`
	Slug        string            `json:"slug"`
	Title       string            `json:"title"`
	Excerpt     string            `json:"excerpt"`
	Rank        float64           `json:"rank"`
}

func searchResult(row db.SearchDocumentsRow) searchResultResponse {
	return searchResultResponse{
		Source: sourceAttribution{
			Slug:        row.SourceSlug,
			Name:        row.SourceName,
			OfficialURL: row.OfficialUrl,
			License:     row.License,
		},
		DocumentURL: "/docs/" + row.SourceSlug + "/" + row.Slug,
		Slug:        row.Slug,
		Title:       row.Title,
		Excerpt:     row.Excerpt,
		Rank:        row.Rank,
	}
}
