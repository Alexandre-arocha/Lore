package sourceconfig

import (
	"encoding/json"
	"testing"
)

func TestValidateDefinition(t *testing.T) {
	license := "MIT"
	valid := Definition{
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

	if err := ValidateDefinition(valid); err != nil {
		t.Fatalf("ValidateDefinition(valid): %v", err)
	}

	cases := map[string]Definition{
		"missing license": func() Definition {
			def := valid
			def.License = nil
			return def
		}(),
		"missing include globs": func() Definition {
			def := valid
			def.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs"}`)
			return def
		}(),
		"bad repo shape": func() Definition {
			def := valid
			def.IngestConfig = json.RawMessage(`{"repo":"repo","branch":"main","docs_path":"docs","include_globs":["**/*.md"]}`)
			return def
		}(),
		"absolute glob": func() Definition {
			def := valid
			def.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["/docs/*.md"]}`)
			return def
		}(),
		"backslash glob": func() Definition {
			def := valid
			def.IngestConfig = json.RawMessage(`{"repo":"owner/repo","branch":"main","docs_path":"docs","include_globs":["guide\\*.md"]}`)
			return def
		}(),
	}

	for name, def := range cases {
		t.Run(name, func(t *testing.T) {
			if err := ValidateDefinition(def); err == nil {
				t.Fatal("ValidateDefinition returned nil, want error")
			}
		})
	}
}

func TestValidateDefinitionsRejectsDuplicateSlugs(t *testing.T) {
	license := "MIT"
	def := Definition{
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
			"include_globs": ["**/*.md"]
		}`),
	}

	if err := ValidateDefinitions([]Definition{def, def}); err == nil {
		t.Fatal("ValidateDefinitions duplicate slugs returned nil, want error")
	}
}
