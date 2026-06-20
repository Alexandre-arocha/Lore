package ingest

import (
	"path"
	"regexp"
	"strings"
)

var (
	reLeadingOrder = regexp.MustCompile(`^\d+[-_.]`)  // "01-intro" -> "intro"
	reNonSlug      = regexp.MustCompile(`[^a-z0-9_-]+`)
	reDashes       = regexp.MustCompile(`-{2,}`)
)

// DeriveSlug builds a clean URL slug from a file path relative to docs_path.
// Index/readme files collapse to their parent directory; the repo root index
// becomes "index".
func DeriveSlug(relPath string) string {
	p := strings.TrimSuffix(path.Clean(relPath), path.Ext(relPath))
	segs := strings.Split(p, "/")

	clean := make([]string, 0, len(segs))
	for _, s := range segs {
		if c := cleanSegment(s); c != "" {
			clean = append(clean, c)
		}
	}

	if n := len(clean); n > 0 {
		switch clean[n-1] {
		case "index", "readme":
			clean = clean[:n-1]
		}
	}

	slug := strings.Join(clean, "/")
	if slug == "" {
		return "index"
	}
	return slug
}

func cleanSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Trim(s, "()")           // prisma uses "(index)" group dirs
	s = reLeadingOrder.ReplaceAllString(s, "")
	s = reNonSlug.ReplaceAllString(s, "-")
	s = reDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// titleFromPath derives a human-ish title from a file path as a last resort.
func titleFromPath(relPath string) string {
	base := strings.TrimSuffix(path.Base(relPath), path.Ext(relPath))
	base = strings.Trim(base, "()")
	base = reLeadingOrder.ReplaceAllString(base, "")
	if base == "" || base == "index" || base == "readme" {
		dir := path.Base(path.Dir(relPath))
		if dir != "" && dir != "." && dir != "/" {
			base = dir
		}
	}
	return humanize(base)
}
