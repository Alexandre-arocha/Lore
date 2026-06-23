package sourcecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCheckCountsCandidateDocs(t *testing.T) {
	server := newGitHubFixture(t, false)
	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithUserAgent("test-agent"))

	report, err := checker.Check(context.Background(), Source{
		Slug: "demo",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["index.md", "guide/*.md", "notes.txt", "skip/*.md"],
			"exclude_globs": ["skip/**"]
		}`),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}

	if report.DefaultBranch != "main" {
		t.Fatalf("DefaultBranch = %q, want main", report.DefaultBranch)
	}
	if report.DocsPathFiles != 5 {
		t.Fatalf("DocsPathFiles = %d, want 5", report.DocsPathFiles)
	}
	if report.CandidateFiles != 3 {
		t.Fatalf("CandidateFiles = %d, want 3", report.CandidateFiles)
	}
	wantSample := []string{"guide/intro.md", "index.md", "notes.txt"}
	if strings.Join(report.Sample, ",") != strings.Join(wantSample, ",") {
		t.Fatalf("Sample = %#v, want %#v", report.Sample, wantSample)
	}
}

func TestCheckRejectsZeroCandidateDocs(t *testing.T) {
	server := newGitHubFixture(t, false)
	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	_, err := checker.Check(context.Background(), Source{
		Slug: "empty",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["missing/**/*.md"],
			"exclude_globs": []
		}`),
	})
	if err == nil || !strings.Contains(err.Error(), "matched zero") {
		t.Fatalf("Check error = %v, want zero-match error", err)
	}
}

func TestCheckRejectsMissingDocsPath(t *testing.T) {
	server := newGitHubFixture(t, false)
	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	_, err := checker.Check(context.Background(), Source{
		Slug: "missing-path",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "missing",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	})
	if err == nil || !strings.Contains(err.Error(), "docs_path") {
		t.Fatalf("Check error = %v, want docs_path error", err)
	}
}

func TestCheckRejectsTruncatedTree(t *testing.T) {
	server := newGitHubFixture(t, true)
	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()))

	_, err := checker.Check(context.Background(), Source{
		Slug: "truncated",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	})
	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("Check error = %v, want truncated error", err)
	}
}

func TestCheckCanSkipRepoMetadata(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "repo metadata should not be fetched", http.StatusInternalServerError)
	})
	mux.HandleFunc("/repos/owner/repo/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"truncated": false,
			"tree": []map[string]string{
				{"path": "docs/index.md", "type": "blob"},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()), WithRepoMetadata(false))
	report, err := checker.Check(context.Background(), Source{
		Slug: "demo",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if report.DefaultBranch != "" {
		t.Fatalf("DefaultBranch = %q, want empty when metadata is skipped", report.DefaultBranch)
	}
}

func TestCheckerCachesGitHubResponses(t *testing.T) {
	var repoHits int32
	var treeHits int32
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&repoHits, 1)
		_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	mux.HandleFunc("/repos/owner/repo/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&treeHits, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"truncated": false,
			"tree": []map[string]string{
				{"path": "docs/index.md", "type": "blob"},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	checker := New("", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	source := Source{
		Slug: "demo",
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	}
	if _, err := checker.Check(context.Background(), source); err != nil {
		t.Fatalf("first Check: %v", err)
	}
	if _, err := checker.Check(context.Background(), source); err != nil {
		t.Fatalf("second Check: %v", err)
	}
	if got := atomic.LoadInt32(&repoHits); got != 1 {
		t.Fatalf("repo hits = %d, want 1", got)
	}
	if got := atomic.LoadInt32(&treeHits); got != 1 {
		t.Fatalf("tree hits = %d, want 1", got)
	}
}

func newGitHubFixture(t *testing.T, truncated bool) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/owner/repo", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Fatal("missing User-Agent")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"default_branch": "main"})
	})
	mux.HandleFunc("/repos/owner/repo/git/trees/main", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("recursive"); got != "1" {
			t.Fatalf("recursive query = %q, want 1", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"truncated": truncated,
			"tree": []map[string]string{
				{"path": "docs/index.md", "type": "blob"},
				{"path": "docs/guide/intro.md", "type": "blob"},
				{"path": "docs/notes.txt", "type": "blob"},
				{"path": "docs/skip/draft.md", "type": "blob"},
				{"path": "docs/assets/logo.png", "type": "blob"},
				{"path": "other/index.md", "type": "blob"},
				{"path": "docs/reference", "type": "tree"},
			},
		})
	})

	return httptest.NewServer(mux)
}
