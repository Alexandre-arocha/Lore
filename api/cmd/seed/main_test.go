package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"lore/api/internal/ingest"
)

func TestValidateBundledSeed(t *testing.T) {
	seeds := loadBundledSeeds(t)

	if len(seeds) == 0 {
		t.Fatal("bundled seed is empty")
	}
	if err := validateSeeds(seeds); err != nil {
		t.Fatalf("validateSeeds: %v", err)
	}
}

func TestValidateSeedsRejectsBadInput(t *testing.T) {
	license := "MIT"
	valid := sourceSeed{
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

	cases := map[string][]sourceSeed{
		"duplicate slug": {valid, valid},
		"missing license": {
			func() sourceSeed {
				s := valid
				s.License = nil
				return s
			}(),
		},
		"missing include globs": {
			func() sourceSeed {
				s := valid
				s.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs"}`)
				return s
			}(),
		},
		"bad repo shape": {
			func() sourceSeed {
				s := valid
				s.IngestConfig = json.RawMessage(`{"repo":"repo","branch":"main","docs_path":"docs","include_globs":["**/*.md"]}`)
				return s
			}(),
		},
		"backslash glob": {
			func() sourceSeed {
				s := valid
				s.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["guide\\*.md"]}`)
				return s
			}(),
		},
	}

	for name, seeds := range cases {
		t.Run(name, func(t *testing.T) {
			if err := validateSeeds(seeds); err == nil {
				t.Fatal("validateSeeds returned nil, want error")
			}
		})
	}
}

func TestCriticalLanguageSeedConfigs(t *testing.T) {
	seeds := seedsBySlug(loadBundledSeeds(t))

	check := func(slug, repo, branch, docsPath string, include, exclude []string) {
		t.Helper()
		seed, ok := seeds[slug]
		if !ok {
			t.Fatalf("missing seed %q", slug)
		}
		cfg, err := ingest.ParseConfig(seed.IngestConfig)
		if err != nil {
			t.Fatalf("%s ParseConfig: %v", slug, err)
		}
		if cfg.Repo != repo || cfg.Branch != branch || cfg.DocsPath != docsPath {
			t.Fatalf("%s config = repo %q branch %q docs_path %q, want repo %q branch %q docs_path %q",
				slug, cfg.Repo, cfg.Branch, cfg.DocsPath, repo, branch, docsPath)
		}
		for _, glob := range include {
			if !contains(cfg.IncludeGlobs, glob) {
				t.Fatalf("%s include_globs missing %q\nhave: %s", slug, glob, strings.Join(cfg.IncludeGlobs, ", "))
			}
		}
		for _, glob := range exclude {
			if !contains(cfg.ExcludeGlobs, glob) {
				t.Fatalf("%s exclude_globs missing %q\nhave: %s", slug, glob, strings.Join(cfg.ExcludeGlobs, ", "))
			}
		}
	}

	check("javascript", "mdn/content", "main", "files/en-us/web/javascript",
		[]string{"guide/*.md", "reference/global_objects/string/*.md", "reference/operators/*.md"},
		nil)
	check("typescript", "microsoft/TypeScript-Website", "v2", "packages/documentation/copy/en",
		[]string{"handbook-v2/**/*.md", "reference/*.md", "tutorials/*.md"},
		nil)
	check("python", "python/cpython", "main", "Doc",
		[]string{"tutorial/*.rst", "reference/*.rst", "library/*.rst"},
		[]string{"whatsnew/**/*.rst"})
	check("java", "openjdk/guide", "master", "src/guide",
		[]string{"**/*.md"},
		nil)
	check("csharp", "dotnet/docs", "main", "docs/csharp",
		[]string{"tour-of-csharp/*.md", "language-reference/*.md", "programming-guide/*.md"},
		[]string{"whats-new/**/*.md", "**/snippets/**/*.md"})
	check("cpp", "MicrosoftDocs/cpp-docs", "main", "docs",
		[]string{"c-language/**/*.md", "cpp/**/*.md"},
		nil)
	check("ruby", "ruby/www.ruby-lang.org", "master", "en/documentation",
		[]string{"**/*.md"},
		nil)
	check("kotlin", "JetBrains/kotlin-web-site", "master", "docs/topics",
		[]string{"**/*.md"},
		[]string{"whatsnew/**/*.md", "compatibility-guides/**/*.md"})
	check("php", "php/doc-en", "master", "language",
		[]string{"*.xml", "**/*.xml"},
		nil)

	jsCfg, err := ingest.ParseConfig(seeds["javascript"].IngestConfig)
	if err != nil {
		t.Fatalf("javascript ParseConfig: %v", err)
	}
	if len(jsCfg.IncludeGlobs) == 1 && jsCfg.IncludeGlobs[0] == "**/*.md" {
		t.Fatal("javascript reverted to a broad MDN glob")
	}
}

func loadBundledSeeds(t *testing.T) []sourceSeed {
	t.Helper()

	raw, err := os.ReadFile("../../seed/sources.json")
	if err != nil {
		t.Fatalf("read bundled seed: %v", err)
	}

	var seeds []sourceSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		t.Fatalf("parse bundled seed: %v", err)
	}
	return seeds
}

func seedsBySlug(seeds []sourceSeed) map[string]sourceSeed {
	out := make(map[string]sourceSeed, len(seeds))
	for _, seed := range seeds {
		out[seed.Slug] = seed
	}
	return out
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
