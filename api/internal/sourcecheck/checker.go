// Package sourcecheck validates source definitions before a full ingestion sync.
package sourcecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"lore/api/internal/ingest"
)

const defaultUserAgent = "LoreSourceCheck/1.0"

// Source is the subset of seed/sources.json needed to validate ingestion paths.
type Source struct {
	Slug         string          `json:"slug"`
	IngestConfig json.RawMessage `json:"ingest_config"`
}

// LoadSeed reads source definitions from a seed JSON file.
func LoadSeed(path string) ([]Source, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read seed: %w", err)
	}

	var sources []Source
	if err := json.Unmarshal(raw, &sources); err != nil {
		return nil, fmt.Errorf("parse seed: %w", err)
	}
	return sources, nil
}

// Report summarizes one source check.
type Report struct {
	Slug           string   `json:"slug"`
	Repo           string   `json:"repo"`
	Branch         string   `json:"branch"`
	DefaultBranch  string   `json:"default_branch"`
	DocsPath       string   `json:"docs_path"`
	DocsPathFiles  int      `json:"docs_path_files"`
	CandidateFiles int      `json:"candidate_files"`
	TreeTruncated  bool     `json:"tree_truncated"`
	Sample         []string `json:"sample"`
}

// Checker verifies source configs against the GitHub REST API.
type Checker struct {
	client            *http.Client
	baseURL           string
	token             string
	userAgent         string
	checkRepoMetadata bool
	repoCache         map[string]repoResponse
	treeCache         map[string]treeResponse
}

// Option customizes a Checker.
type Option func(*Checker)

func New(token string, opts ...Option) *Checker {
	c := &Checker{
		client:            &http.Client{Timeout: 30 * time.Second},
		baseURL:           "https://api.github.com",
		token:             token,
		userAgent:         defaultUserAgent,
		checkRepoMetadata: true,
		repoCache:         map[string]repoResponse{},
		treeCache:         map[string]treeResponse{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithBaseURL(baseURL string) Option {
	return func(c *Checker) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Checker) {
		if client != nil {
			c.client = client
		}
	}
}

func WithUserAgent(userAgent string) Option {
	return func(c *Checker) {
		if strings.TrimSpace(userAgent) != "" {
			c.userAgent = userAgent
		}
	}
}

func WithRepoMetadata(enabled bool) Option {
	return func(c *Checker) {
		c.checkRepoMetadata = enabled
	}
}

// Check validates that a source's docs_path exists and that its include/exclude
// globs select at least one file supported by the ingestion pipeline.
func (c *Checker) Check(ctx context.Context, source Source) (Report, error) {
	cfg, err := ingest.ParseConfig(source.IngestConfig)
	if err != nil {
		return Report{Slug: source.Slug}, fmt.Errorf("%s: %w", source.Slug, err)
	}

	var repo repoResponse
	if c.checkRepoMetadata {
		repo, err = c.fetchRepo(ctx, cfg.Repo)
		if err != nil {
			return Report{Slug: source.Slug, Repo: cfg.Repo, Branch: cfg.Branch, DocsPath: cfg.DocsPath}, err
		}
	}

	tree, err := c.fetchTree(ctx, cfg.Repo, cfg.Branch)
	if err != nil {
		return Report{Slug: source.Slug, Repo: cfg.Repo, Branch: cfg.Branch, DefaultBranch: repo.DefaultBranch, DocsPath: cfg.DocsPath}, err
	}

	report := Report{
		Slug:          source.Slug,
		Repo:          cfg.Repo,
		Branch:        cfg.Branch,
		DefaultBranch: repo.DefaultBranch,
		DocsPath:      cfg.DocsPath,
		TreeTruncated: tree.Truncated,
	}

	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		rel, ok := relativeToDocsPath(entry.Path, cfg.DocsPath)
		if !ok {
			continue
		}
		report.DocsPathFiles++
		if !ingest.SupportedDocExtension(path.Ext(rel)) {
			continue
		}
		if !ingest.MatchGlobs(rel, cfg.IncludeGlobs, cfg.ExcludeGlobs) {
			continue
		}
		report.CandidateFiles++
		report.Sample = append(report.Sample, rel)
	}

	sort.Strings(report.Sample)
	if len(report.Sample) > 5 {
		report.Sample = report.Sample[:5]
	}

	if report.TreeTruncated {
		return report, fmt.Errorf("%s: GitHub tree is truncated; cannot safely verify globs", source.Slug)
	}
	if report.DocsPathFiles == 0 {
		return report, fmt.Errorf("%s: docs_path %q matched no files in %s@%s", source.Slug, cfg.DocsPath, cfg.Repo, cfg.Branch)
	}
	if report.CandidateFiles == 0 {
		return report, fmt.Errorf("%s: include/exclude globs matched zero supported docs under %q", source.Slug, cfg.DocsPath)
	}

	return report, nil
}

// Preflight adapts Checker for the ingestion worker. It records only success or
// failure there; detailed reports remain available through Check/cmd/check-sources.
func (c *Checker) Preflight(ctx context.Context, source ingest.PreflightSource) error {
	_, err := c.Check(ctx, Source{Slug: source.Slug, IngestConfig: source.IngestConfig})
	return err
}

func relativeToDocsPath(fullPath, docsPath string) (string, bool) {
	prefix := strings.Trim(docsPath, "/")
	if prefix == "" {
		return fullPath, true
	}
	if !strings.HasPrefix(fullPath, prefix+"/") {
		return "", false
	}
	return strings.TrimPrefix(fullPath, prefix+"/"), true
}

type repoResponse struct {
	DefaultBranch string `json:"default_branch"`
}

type treeResponse struct {
	Tree      []treeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

type treeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func (c *Checker) fetchRepo(ctx context.Context, repo string) (repoResponse, error) {
	if cached, ok := c.repoCache[repo]; ok {
		return cached, nil
	}

	var out repoResponse
	if err := c.getJSON(ctx, "/repos/"+repo, &out); err != nil {
		return out, fmt.Errorf("%s: fetch repo: %w", repo, err)
	}
	c.repoCache[repo] = out
	return out, nil
}

func (c *Checker) fetchTree(ctx context.Context, repo, branch string) (treeResponse, error) {
	cacheKey := repo + "@" + branch
	if cached, ok := c.treeCache[cacheKey]; ok {
		return cached, nil
	}

	var out treeResponse
	endpoint := "/repos/" + repo + "/git/trees/" + url.PathEscape(branch) + "?recursive=1"
	if err := c.getJSON(ctx, endpoint, &out); err != nil {
		return out, fmt.Errorf("%s@%s: fetch tree: %w", repo, branch, err)
	}
	c.treeCache[cacheKey] = out
	return out, nil
}

func (c *Checker) getJSON(ctx context.Context, endpoint string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.baseURL, "/")+endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return err
	}

	return nil
}
