package ingest

import (
	"bytes"
	stdhtml "html"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// TOCEntry is one heading in a page's table of contents (H2 with H3 children).
type TOCEntry struct {
	Title    string     `json:"title"`
	Anchor   string     `json:"anchor"`
	Children []TOCEntry `json:"children,omitempty"`
}

// Rendered is the output of rendering one Markdown body.
type Rendered struct {
	HTML string     // sanitized HTML with chroma-highlighted code
	Text string     // plain text, for the search vector
	TOC  []TOCEntry // H2/H3 tree with anchors
	H1   string     // first H1 in the body, if any
}

// ChromaStyle is the chroma style whose CSS the frontend must ship.
const ChromaStyle = "github"

// Renderer converts Markdown to highlighted, sanitized HTML. It is safe for
// concurrent use.
type Renderer struct {
	md         goldmark.Markdown
	htmlPolicy *bluemonday.Policy
	textPolicy *bluemonday.Policy
}

func NewRenderer() *Renderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle(ChromaStyle),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)

	return &Renderer{
		md:         md,
		htmlPolicy: newHTMLPolicy(),
		textPolicy: bluemonday.StrictPolicy(),
	}
}

// Render parses the body once, extracting the TOC and first H1, then renders and
// sanitizes the HTML and derives the plain-text search content.
func (r *Renderer) Render(body []byte) (Rendered, error) {
	reader := text.NewReader(body)
	doc := r.md.Parser().Parse(reader)

	toc, h1 := buildTOC(doc, body)

	var buf bytes.Buffer
	if err := r.md.Renderer().Render(&buf, body, doc); err != nil {
		return Rendered{}, err
	}

	safe := r.htmlPolicy.SanitizeBytes(buf.Bytes())
	plain := r.textPolicy.SanitizeBytes(safe)

	return Rendered{
		HTML: string(safe),
		Text: normalizeSpace(stdhtml.UnescapeString(string(plain))),
		TOC:  toc,
		H1:   h1,
	}, nil
}

func buildTOC(doc ast.Node, src []byte) ([]TOCEntry, string) {
	var toc []TOCEntry
	var h1 string

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		title := nodeText(h, src)
		switch h.Level {
		case 1:
			if h1 == "" {
				h1 = title
			}
		case 2:
			toc = append(toc, TOCEntry{Title: title, Anchor: headingID(h)})
		case 3:
			if len(toc) > 0 {
				last := &toc[len(toc)-1]
				last.Children = append(last.Children, TOCEntry{Title: title, Anchor: headingID(h)})
			}
		}
		return ast.WalkSkipChildren, nil
	})

	return toc, h1
}

func headingID(h *ast.Heading) string {
	if v, ok := h.AttributeString("id"); ok {
		switch s := v.(type) {
		case []byte:
			return string(s)
		case string:
			return s
		}
	}
	return ""
}

func nodeText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch t := c.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		default:
			b.WriteString(nodeText(c, src))
		}
	}
	return b.String()
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func newHTMLPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	// chroma wraps tokens in <span>/<div class="...">; keep them plus classes so
	// the shipped highlight CSS applies.
	p.AllowElements("span", "div")
	p.AllowAttrs("class").Globally()
	// keep heading anchors for the table of contents.
	p.AllowAttrs("id").OnElements("h1", "h2", "h3", "h4", "h5", "h6")
	// GFM tables
	p.AllowElements("table", "thead", "tbody", "tfoot", "tr", "td", "th", "caption")
	p.AllowAttrs("colspan", "rowspan", "align").OnElements("td", "th")
	return p
}
