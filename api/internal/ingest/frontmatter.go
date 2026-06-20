package ingest

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

// FrontMatter holds the fields we care about from a page's YAML front-matter.
type FrontMatter struct {
	Title       string
	Description string
	Order       *int
}

// utf8BOM is the UTF-8 byte-order mark some files start with.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// splitFrontMatter separates a leading YAML front-matter block (delimited by
// "---") from the Markdown body. If there is none (or it is malformed), the
// remaining source is returned unchanged as the body.
func splitFrontMatter(src []byte) (FrontMatter, []byte) {
	src = bytes.TrimPrefix(src, utf8BOM)
	trimmed := bytes.TrimLeft(src, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("---")) {
		return FrontMatter{}, src
	}

	// Move past the opening "---" line.
	rest := trimmed[3:]
	nl := bytes.IndexByte(rest, '\n')
	if nl < 0 {
		return FrontMatter{}, src
	}
	rest = rest[nl+1:]

	// Find the closing delimiter at the start of a line.
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return FrontMatter{}, src
	}
	yamlBytes := rest[:end]

	body := rest[end+1:] // include the newline before "---"
	if i := bytes.IndexByte(body, '\n'); i >= 0 {
		body = body[i+1:] // skip the "---" line itself
	} else {
		body = nil
	}

	var raw map[string]any
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return FrontMatter{}, src
	}

	fm := FrontMatter{}
	if t, ok := raw["title"].(string); ok {
		fm.Title = t
	}
	if d, ok := raw["description"].(string); ok {
		fm.Description = d
	}
	fm.Order = extractOrder(raw)
	return fm, body
}

// extractOrder looks for a sidebar order hint in common front-matter shapes.
func extractOrder(raw map[string]any) *int {
	if v, ok := toInt(raw["order"]); ok {
		return &v
	}
	if sidebar, ok := raw["sidebar"].(map[string]any); ok {
		if v, ok := toInt(sidebar["order"]); ok {
			return &v
		}
	}
	return nil
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
