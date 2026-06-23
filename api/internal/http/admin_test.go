package http

import (
	"encoding/json"
	"testing"
)

func TestValidateUpsertSourceRequest(t *testing.T) {
	license := "MIT"
	valid := upsertSourceRequest{
		Slug:        "demo",
		Name:        "Demo",
		Kind:        "language",
		Category:    "backend",
		OfficialURL: "https://example.com",
		License:     &license,
		IngestConfig: json.RawMessage(`{
			"repo": "owner/repo",
			"branch": "main",
			"docs_path": "docs",
			"include_globs": ["**/*.md"],
			"exclude_globs": []
		}`),
	}

	if err := validateUpsertSourceRequest(valid); err != nil {
		t.Fatalf("validateUpsertSourceRequest(valid): %v", err)
	}

	cases := map[string]upsertSourceRequest{
		"missing license": func() upsertSourceRequest {
			req := valid
			req.License = nil
			return req
		}(),
		"missing include globs": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs"}`)
			return req
		}(),
		"bad repo shape": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"repo","branch":"main","docs_path":"docs","include_globs":["**/*.md"]}`)
			return req
		}(),
		"absolute glob": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["/docs/*.md"]}`)
			return req
		}(),
		"backslash glob": func() upsertSourceRequest {
			req := valid
			req.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["guide\\*.md"]}`)
			return req
		}(),
	}

	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateUpsertSourceRequest(req); err == nil {
				t.Fatal("validateUpsertSourceRequest returned nil, want error")
			}
		})
	}
}
