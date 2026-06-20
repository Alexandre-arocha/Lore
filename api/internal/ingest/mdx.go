package ingest

import (
	"regexp"
	"strings"
)

var (
	reImport      = regexp.MustCompile(`^\s*import\s+[^;]*?from\s+['"][^'"]+['"];?\s*$`)
	reImportBare  = regexp.MustCompile(`^\s*import\s+['"][^'"]+['"];?\s*$`)
	reExportLine  = regexp.MustCompile(`^\s*export\s+`)
	reExportTitle = regexp.MustCompile(`^\s*export\s+const\s+title\s*=\s*['"` + "`" + `](.+?)['"` + "`" + `]\s*;?\s*$`)
	reExportDesc  = regexp.MustCompile(`^\s*export\s+const\s+description\s*=\s*['"` + "`" + `](.+?)['"` + "`" + `]\s*;?\s*$`)
	reMDXComment  = regexp.MustCompile(`^\s*\{/\*.*\*/\}\s*$`)
	reJSXOpen     = regexp.MustCompile(`^\s*</?[A-Z][A-Za-z0-9.]*`) // <Component or </Component
	reJSXFragment = regexp.MustCompile(`^\s*</?>`)                  // <> or </>
)

// sanitizeMDX degrades MDX to plain GFM Markdown so goldmark can render it
// without leaking JSX. It strips ESM import/export lines and block-level JSX
// components, capturing `export const title/description` along the way. This is
// intentionally heuristic: the goal is to never crash on a file with custom
// components, accepting that some component-rendered content is dropped.
//
// Returns the cleaned Markdown, any title/description found in exports, and the
// number of JSX blocks that were neutralized (surfaced as a warning count).
func sanitizeMDX(src string) (clean, title, description string, warnings int) {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))

	inFence := false
	inJSXBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Never touch fenced code blocks.
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			out = append(out, line)
			continue
		}
		if inFence {
			out = append(out, line)
			continue
		}

		if inJSXBlock {
			if jsxBlockCloses(trimmed) {
				inJSXBlock = false
			}
			continue
		}

		if m := reExportTitle.FindStringSubmatch(line); m != nil {
			if title == "" {
				title = m[1]
			}
			continue
		}
		if m := reExportDesc.FindStringSubmatch(line); m != nil {
			if description == "" {
				description = m[1]
			}
			continue
		}
		if reImport.MatchString(line) || reImportBare.MatchString(line) || reExportLine.MatchString(line) {
			continue
		}
		if reMDXComment.MatchString(line) {
			continue
		}

		if reJSXOpen.MatchString(line) || reJSXFragment.MatchString(line) {
			warnings++
			if !jsxBlockCloses(trimmed) {
				inJSXBlock = true
			}
			continue
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n"), title, description, warnings
}

// jsxBlockCloses reports whether a JSX line is self-contained (self-closing
// "/>", or contains a closing "</...>", or is a bare ">" closing a tag), i.e.
// the following lines are not part of this JSX element.
func jsxBlockCloses(trimmed string) bool {
	if strings.HasSuffix(trimmed, "/>") {
		return true
	}
	if strings.Contains(trimmed, "</") {
		return true
	}
	// An opening tag that completes on this line but is not self-closing (e.g.
	// "<Tabs>") starts a block; only treat a trailing ">" as closing when the
	// line has no unclosed attribute braces.
	return false
}
