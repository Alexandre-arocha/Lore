package ingest

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

// RawFile is a documentation file extracted from a repo tarball. Path is
// relative to the configured docs_path, using forward slashes.
type RawFile struct {
	Path    string
	Content []byte
}

// GitHubFetcher downloads repo tarballs from the GitHub API.
type GitHubFetcher struct {
	token  string
	client *http.Client
}

func NewGitHubFetcher(token string) *GitHubFetcher {
	return &GitHubFetcher{
		token:  token,
		client: &http.Client{Timeout: 10 * time.Minute},
	}
}

// maxFileSize guards against pathological files; docs pages are tiny.
const maxFileSize = 8 << 20 // 8 MiB

// Fetch downloads the repo tarball and returns supported docs files under
// DocsPath that match the include/exclude globs. The tarball is streamed, so a
// large monorepo does not need to fit in memory.
func (g *GitHubFetcher) Fetch(ctx context.Context, cfg Config) ([]RawFile, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", cfg.Repo, cfg.Branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if g.token != "" {
		req.Header.Set("Authorization", "Bearer "+g.token)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: download %s: %w", cfg.Repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github: tarball %s@%s: status %d: %s",
			cfg.Repo, cfg.Branch, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("github: gzip: %w", err)
	}
	defer gz.Close()

	docsPrefix := strings.Trim(cfg.DocsPath, "/")
	tr := tar.NewReader(gz)
	var files []RawFile

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("github: read tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		// GitHub wraps everything in a top-level "<owner>-<repo>-<sha>/" dir.
		name := hdr.Name
		slash := strings.IndexByte(name, '/')
		if slash < 0 {
			continue
		}
		name = name[slash+1:]

		if docsPrefix != "" {
			if !strings.HasPrefix(name, docsPrefix+"/") {
				continue
			}
			name = strings.TrimPrefix(name, docsPrefix+"/")
		}

		if !supportedDocExtension(path.Ext(name)) {
			continue
		}
		if !matchGlobs(name, cfg.IncludeGlobs, cfg.ExcludeGlobs) {
			continue
		}

		content, err := io.ReadAll(io.LimitReader(tr, maxFileSize))
		if err != nil {
			return nil, fmt.Errorf("github: read %s: %w", name, err)
		}
		files = append(files, RawFile{Path: name, Content: content})
	}

	return files, nil
}

func supportedDocExtension(ext string) bool {
	switch strings.ToLower(ext) {
	case ".md", ".mdx", ".rst", ".xml":
		return true
	default:
		return false
	}
}

// matchGlobs reports whether name passes the include/exclude globs. An empty
// include list matches everything. Patterns support "**" via doublestar.
func matchGlobs(name string, include, exclude []string) bool {
	included := len(include) == 0
	for _, p := range include {
		if ok, _ := doublestar.Match(p, name); ok {
			included = true
			break
		}
	}
	if !included {
		return false
	}
	for _, p := range exclude {
		if ok, _ := doublestar.Match(p, name); ok {
			return false
		}
	}
	return true
}
